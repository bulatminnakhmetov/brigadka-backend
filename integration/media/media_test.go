package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/media"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MediaIntegrationTestSuite определяет набор интеграционных тестов для медиа
type MediaIntegrationTestSuite struct {
	suite.Suite
	appUrl     string
	bucketName string
}

// SetupSuite подготавливает общее окружение перед запуском всех тестов
func (s *MediaIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Значение по умолчанию для локального тестирования
	}
}

// Вспомогательная функция для создания тестового пользователя и профиля
func (s *MediaIntegrationTestSuite) createTestUserAndProfile() (int, int, string, error) {
	// Генерируем уникальный email для пользователя
	testEmail := fmt.Sprintf("media_test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
	testPassword := "TestPassword123"

	// Регистрируем тестового пользователя
	registerData := auth.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
		FullName: "Media Test User",
		Gender:   "male",
		Age:      30,
		CityID:   1,
	}

	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to send register request: %w", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registerResp.Body)
		return 0, 0, "", fmt.Errorf("failed to register user: %s", body)
	}

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to decode register response: %w", err)
	}

	// Создаем тестовый профиль для пользователя
	createProfileData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      registerResult.User.ID,
			Description: "Test profile for media tests",
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createProfileJSON, _ := json.Marshal(createProfileData)
	createProfileReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createProfileJSON))
	createProfileReq.Header.Set("Content-Type", "application/json")
	createProfileReq.Header.Set("Authorization", "Bearer "+registerResult.Token)

	createProfileResp, err := client.Do(createProfileReq)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to send create profile request: %w", err)
	}
	defer createProfileResp.Body.Close()

	if createProfileResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(createProfileResp.Body)
		return 0, 0, "", fmt.Errorf("failed to create profile: %s", body)
	}

	var createdProfile profile.Profile
	err = json.NewDecoder(createProfileResp.Body).Decode(&createdProfile)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to decode create profile response: %w", err)
	}

	return registerResult.User.ID, createdProfile.ProfileID, registerResult.Token, nil
}

// Вспомогательная функция для создания временного файла изображения
func createTempImageFile() (string, error) {
	// Создаем временный файл
	tmpFile, err := os.CreateTemp("", "test-image-*.jpg")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Тестовое содержимое файла (минимальный валидный JPEG)
	jpegHeader := []byte{
		0xFF, 0xD8, // SOI marker
		0xFF, 0xE0, // APP0 marker
		0x00, 0x10, // APP0 length (16 bytes)
		0x4A, 0x46, 0x49, 0x46, 0x00, // Identifier: "JFIF\0"
		0x01, 0x01, // JFIF version 1.1
		0x00,       // Density units: no units
		0x00, 0x01, // X density: 1
		0x00, 0x01, // Y density: 1
		0x00, // No thumbnail
		0x00, // No thumbnail
		// Минимальное изображение (1x1 пиксель)
		0xFF, 0xDB, // DQT marker
		0x00, 0x43, // DQT length (67 bytes)
		0x00, // Precision and table ID
	}

	// Добавляем таблицу квантования (просто для полноты)
	for i := 0; i < 64; i++ {
		jpegHeader = append(jpegHeader, 0x10) // Простая константа
	}

	// Добавляем SOF0 (начало кадра)
	sofMarker := []byte{
		0xFF, 0xC0, // SOF0 marker
		0x00, 0x0B, // Length (11 bytes)
		0x08,       // Precision (8 bits)
		0x00, 0x01, // Height (1 pixel)
		0x00, 0x01, // Width (1 pixel)
		0x01,             // Number of components (1, монохромное)
		0x01, 0x11, 0x00, // Component 1 parameters
	}
	jpegHeader = append(jpegHeader, sofMarker...)

	// Добавляем DHT (таблица Хаффмана)
	dhtMarker := []byte{
		0xFF, 0xC4, // DHT marker
		0x00, 0x14, // Length (20 bytes)
		0x00, // Table ID and type
	}
	// Счетчики символов для каждой длины кода
	for i := 0; i < 16; i++ {
		if i == 0 {
			dhtMarker = append(dhtMarker, 0x01) // Один символ длины 1
		} else {
			dhtMarker = append(dhtMarker, 0x00) // Нет символов для других длин
		}
	}
	dhtMarker = append(dhtMarker, 0x00) // Значение символа
	jpegHeader = append(jpegHeader, dhtMarker...)

	// Добавляем SOS (начало сканирования)
	sosMarker := []byte{
		0xFF, 0xDA, // SOS marker
		0x00, 0x08, // Length (8 bytes)
		0x01,       // Number of components (1)
		0x01, 0x00, // Component 1 parameters
		0x00, 0x3F, 0x00, // Spectral selection and approximation
	}
	jpegHeader = append(jpegHeader, sosMarker...)

	// Добавляем минимальные данные и EOI (конец изображения)
	imageData := []byte{0x00, 0xFF, 0xD9} // Простые данные и EOI marker
	jpegHeader = append(jpegHeader, imageData...)

	// Записываем в файл
	if _, err := tmpFile.Write(jpegHeader); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// TestUploadMedia проверяет загрузку медиа файла
func (s *MediaIntegrationTestSuite) TestUploadMedia() {
	t := s.T()

	// Создаем пользователя и профиль для теста
	_, profileID, token, err := s.createTestUserAndProfile()
	assert.NoError(t, err, "Failed to create test user and profile")

	// Создаем временный файл для загрузки
	tmpFilePath, err := createTempImageFile()
	assert.NoError(t, err, "Failed to create temp image file")
	defer os.Remove(tmpFilePath)

	// Создаем multipart форму для загрузки файла
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Добавляем profile_id в форму
	err = writer.WriteField("profile_id", fmt.Sprintf("%d", profileID))
	assert.NoError(t, err)

	// Добавляем role в форму
	err = writer.WriteField("role", "avatar")
	assert.NoError(t, err)

	// Добавляем файл в форму
	file, err := os.Open(tmpFilePath)
	assert.NoError(t, err)
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(tmpFilePath))
	assert.NoError(t, err)

	_, err = io.Copy(part, file)
	assert.NoError(t, err)

	err = writer.Close()
	assert.NoError(t, err)

	// Создаем запрос для загрузки файла
	uploadReq, err := http.NewRequest("POST", s.appUrl+"/api/media/upload", &requestBody)
	assert.NoError(t, err)

	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	uploadResp, err := client.Do(uploadReq)
	assert.NoError(t, err)
	defer uploadResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusCreated, uploadResp.StatusCode, "Should return status 201 Created")

	// Проверяем содержимое ответа
	var mediaResp media.MediaResponse
	err = json.NewDecoder(uploadResp.Body).Decode(&mediaResp)
	assert.NoError(t, err)

	// Проверяем поля созданного медиа
	assert.NotZero(t, mediaResp.Media.ID, "Media ID should not be zero")
	assert.Equal(t, profileID, mediaResp.Media.ProfileID, "Profile ID should match")
	assert.Equal(t, "avatar", mediaResp.Media.Role, "Media role should match")
	assert.NotEmpty(t, mediaResp.Media.URL, "Media URL should not be empty")
	assert.False(t, mediaResp.Media.CreatedAt.IsZero(), "Created at should not be zero")

	// Сохраняем ID медиа для последующих тестов
	mediaID := mediaResp.Media.ID

	// Проверяем получение медиа по ID
	t.Run("Get media by ID", func(t *testing.T) {
		getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/media/%d", s.appUrl, mediaID), nil)
		getReq.Header.Set("Authorization", "Bearer "+token)

		getResp, err := client.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode, "Should return status 200 OK")

		var getMediaResp media.MediaResponse
		err = json.NewDecoder(getResp.Body).Decode(&getMediaResp)
		assert.NoError(t, err)

		assert.Equal(t, mediaID, getMediaResp.Media.ID, "Media ID should match")
		assert.Equal(t, profileID, getMediaResp.Media.ProfileID, "Profile ID should match")
		assert.Equal(t, "avatar", getMediaResp.Media.Role, "Media role should match")
	})

	// Проверяем получение медиа по профилю
	t.Run("Get media by profile", func(t *testing.T) {
		getProfileMediaReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d/media", s.appUrl, profileID), nil)
		getProfileMediaReq.Header.Set("Authorization", "Bearer "+token)

		getProfileMediaResp, err := client.Do(getProfileMediaReq)
		assert.NoError(t, err)
		defer getProfileMediaResp.Body.Close()

		assert.Equal(t, http.StatusOK, getProfileMediaResp.StatusCode, "Should return status 200 OK")

		var mediaListResp media.MediaListResponse
		err = json.NewDecoder(getProfileMediaResp.Body).Decode(&mediaListResp)
		assert.NoError(t, err)

		assert.GreaterOrEqual(t, len(mediaListResp.Media), 1, "Should have at least one media")

		// Ищем наше медиа в списке
		found := false
		for _, m := range mediaListResp.Media {
			if m.ID == mediaID {
				found = true
				assert.Equal(t, profileID, m.ProfileID, "Profile ID should match")
				assert.Equal(t, "avatar", m.Role, "Media role should match")
				break
			}
		}
		assert.True(t, found, "Uploaded media should be found in profile media list")
	})

	// Проверяем получение медиа по профилю с фильтрацией по роли
	t.Run("Get media by profile with role filter", func(t *testing.T) {
		getProfileMediaWithRoleReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d/media?role=avatar", s.appUrl, profileID), nil)
		getProfileMediaWithRoleReq.Header.Set("Authorization", "Bearer "+token)

		getProfileMediaWithRoleResp, err := client.Do(getProfileMediaWithRoleReq)
		assert.NoError(t, err)
		defer getProfileMediaWithRoleResp.Body.Close()

		assert.Equal(t, http.StatusOK, getProfileMediaWithRoleResp.StatusCode, "Should return status 200 OK")

		var mediaListRoleResp media.MediaListResponse
		err = json.NewDecoder(getProfileMediaWithRoleResp.Body).Decode(&mediaListRoleResp)
		assert.NoError(t, err)

		assert.GreaterOrEqual(t, len(mediaListRoleResp.Media), 1, "Should have at least one avatar media")

		for _, m := range mediaListRoleResp.Media {
			assert.Equal(t, "avatar", m.Role, "All media should have avatar role")
		}
	})

	// Проверяем удаление медиа
	t.Run("Delete media", func(t *testing.T) {
		deleteReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/media/%d", s.appUrl, mediaID), nil)
		deleteReq.Header.Set("Authorization", "Bearer "+token)

		deleteResp, err := client.Do(deleteReq)
		assert.NoError(t, err)
		defer deleteResp.Body.Close()

		assert.Equal(t, http.StatusNoContent, deleteResp.StatusCode, "Should return status 204 No Content")

		// Проверяем, что медиа действительно удалено
		getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/media/%d", s.appUrl, mediaID), nil)
		getReq.Header.Set("Authorization", "Bearer "+token)

		getResp, err := client.Do(getReq)
		assert.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Should return status 404 Not Found")
	})
}

// TestUploadInvalidMedia проверяет обработку ошибок при загрузке невалидных медиа
func (s *MediaIntegrationTestSuite) TestUploadInvalidMedia() {
	t := s.T()

	// Создаем пользователя и профиль для теста
	_, profileID, token, err := s.createTestUserAndProfile()
	assert.NoError(t, err, "Failed to create test user and profile")

	// Тестовые случаи с невалидными данными
	testCases := []struct {
		name        string
		setup       func(writer *multipart.Writer) error
		statusCode  int
		errorString string
	}{
		{
			name: "Missing profile_id",
			setup: func(writer *multipart.Writer) error {
				// Добавляем только role в форму
				if err := writer.WriteField("role", "avatar"); err != nil {
					return err
				}

				// Создаем временный файл для загрузки
				tmpFilePath, err := createTempImageFile()
				if err != nil {
					return err
				}
				defer os.Remove(tmpFilePath)

				file, err := os.Open(tmpFilePath)
				if err != nil {
					return err
				}
				defer file.Close()

				part, err := writer.CreateFormFile("file", filepath.Base(tmpFilePath))
				if err != nil {
					return err
				}

				_, err = io.Copy(part, file)
				return err
			},
			statusCode:  http.StatusBadRequest,
			errorString: "Profile ID is required",
		},
		{
			name: "Missing role",
			setup: func(writer *multipart.Writer) error {
				// Добавляем только profile_id в форму
				if err := writer.WriteField("profile_id", fmt.Sprintf("%d", profileID)); err != nil {
					return err
				}

				// Создаем временный файл для загрузки
				tmpFilePath, err := createTempImageFile()
				if err != nil {
					return err
				}
				defer os.Remove(tmpFilePath)

				file, err := os.Open(tmpFilePath)
				if err != nil {
					return err
				}
				defer file.Close()

				part, err := writer.CreateFormFile("file", filepath.Base(tmpFilePath))
				if err != nil {
					return err
				}

				_, err = io.Copy(part, file)
				return err
			},
			statusCode:  http.StatusBadRequest,
			errorString: "Media role is required",
		},
		{
			name: "Invalid role",
			setup: func(writer *multipart.Writer) error {
				// Добавляем profile_id и невалидный role в форму
				if err := writer.WriteField("profile_id", fmt.Sprintf("%d", profileID)); err != nil {
					return err
				}
				if err := writer.WriteField("role", "invalid_role"); err != nil {
					return err
				}

				// Создаем временный файл для загрузки
				tmpFilePath, err := createTempImageFile()
				if err != nil {
					return err
				}
				defer os.Remove(tmpFilePath)

				file, err := os.Open(tmpFilePath)
				if err != nil {
					return err
				}
				defer file.Close()

				part, err := writer.CreateFormFile("file", filepath.Base(tmpFilePath))
				if err != nil {
					return err
				}

				_, err = io.Copy(part, file)
				return err
			},
			statusCode:  http.StatusBadRequest,
			errorString: "Invalid media role",
		},
		{
			name: "Missing file",
			setup: func(writer *multipart.Writer) error {
				// Добавляем profile_id и role в форму, но не файл
				if err := writer.WriteField("profile_id", fmt.Sprintf("%d", profileID)); err != nil {
					return err
				}
				return writer.WriteField("role", "avatar")
			},
			statusCode:  http.StatusBadRequest,
			errorString: "Failed to get file",
		},
	}

	client := &http.Client{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var requestBody bytes.Buffer
			writer := multipart.NewWriter(&requestBody)

			err := tc.setup(writer)
			assert.NoError(t, err)

			err = writer.Close()
			assert.NoError(t, err)

			uploadReq, err := http.NewRequest("POST", s.appUrl+"/api/media/upload", &requestBody)
			assert.NoError(t, err)

			uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
			uploadReq.Header.Set("Authorization", "Bearer "+token)

			uploadResp, err := client.Do(uploadReq)
			assert.NoError(t, err)
			defer uploadResp.Body.Close()

			// Проверяем статус ответа
			assert.Equal(t, tc.statusCode, uploadResp.StatusCode)

			// Проверяем содержимое ответа на наличие ожидаемой ошибки
			body, err := io.ReadAll(uploadResp.Body)
			assert.NoError(t, err)
			assert.Contains(t, string(body), tc.errorString)
		})
	}
}

// TestGetNonexistentMedia проверяет получение несуществующего медиа
func (s *MediaIntegrationTestSuite) TestGetNonexistentMedia() {
	t := s.T()

	// Создаем пользователя только для получения токена
	_, _, token, err := s.createTestUserAndProfile()
	assert.NoError(t, err, "Failed to create test user and profile")

	// Запрашиваем несуществующее медиа
	nonexistentMediaID := 999999 // Предполагаем, что такого ID нет в базе
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/media/%d", s.appUrl, nonexistentMediaID), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	defer getResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Should return status 404 Not Found")
}

// TestUnauthorizedAccess проверяет доступ без авторизации
func (s *MediaIntegrationTestSuite) TestUnauthorizedAccess() {
	t := s.T()

	// Создаем запрос без токена авторизации
	uploadReq, _ := http.NewRequest("POST", s.appUrl+"/api/media/upload", nil)
	uploadReq.Header.Set("Content-Type", "multipart/form-data")

	client := &http.Client{}
	uploadResp, err := client.Do(uploadReq)
	assert.NoError(t, err)
	defer uploadResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusUnauthorized, uploadResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestMediaIntegration запускает набор интеграционных тестов для медиа
func TestMediaIntegration(t *testing.T) {
	// Пропускаем тесты, если задана переменная окружения SKIP_INTEGRATION_TESTS
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(MediaIntegrationTestSuite))
}
