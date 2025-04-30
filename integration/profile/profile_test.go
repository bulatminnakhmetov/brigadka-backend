package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/profile"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ProfileIntegrationTestSuite определяет набор интеграционных тестов для профилей
type ProfileIntegrationTestSuite struct {
	suite.Suite
	appUrl string
}

// SetupSuite подготавливает общее окружение перед запуском всех тестов
func (s *ProfileIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Значение по умолчанию для локального тестирования
	}
}

// Вспомогательная функция для создания тестового пользователя
func (s *ProfileIntegrationTestSuite) createTestUser() (int, string, error) {
	// Генерируем уникальный email для пользователя
	testEmail := fmt.Sprintf("profile_test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
	testPassword := "TestPassword123"

	// Регистрируем тестового пользователя
	registerData := auth.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
		FullName: "Profile Test User",
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
		return 0, "", fmt.Errorf("failed to register test user: %v", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registerResp.Body)
		return 0, "", fmt.Errorf("failed to register test user. Status: %d, Body: %s", registerResp.StatusCode, string(body))
	}

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode register response: %v", err)
	}

	return registerResult.User.ID, registerResult.Token, nil
}

// TestCreateImprovProfile тестирует создание профиля импровизации
func (s *ProfileIntegrationTestSuite) TestCreateImprovProfile() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Данные для создания профиля импровизации
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test improv profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: true,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "Should return status 201 Created")

	// Проверяем содержимое ответа
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)

	// Проверяем поля созданного профиля
	assert.NotZero(t, createdProfile.ProfileID, "Profile ID should not be zero")
	assert.Equal(t, userID, createdProfile.UserID, "User ID should match")
	assert.Equal(t, "Test improv profile description", createdProfile.Description, "Description should match")
	assert.Equal(t, profile.ActivityTypeImprov, createdProfile.ActivityType, "Activity type should match")
	assert.False(t, createdProfile.CreatedAt.IsZero(), "Created at should not be zero")
}

// TestCreateMusicProfile тестирует создание музыкального профиля
func (s *ProfileIntegrationTestSuite) TestCreateMusicProfile() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Данные для создания музыкального профиля
	createData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test music profile description",
			ActivityType: profile.ActivityTypeMusic,
		},
		Genres:      []string{"rock", "jazz"},
		Instruments: []string{"acoustic_guitar", "piano"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "Should return status 201 Created")

	// Проверяем содержимое ответа
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)

	// Проверяем поля созданного профиля
	assert.NotZero(t, createdProfile.ProfileID, "Profile ID should not be zero")
	assert.Equal(t, userID, createdProfile.UserID, "User ID should match")
	assert.Equal(t, "Test music profile description", createdProfile.Description, "Description should match")
	assert.Equal(t, profile.ActivityTypeMusic, createdProfile.ActivityType, "Activity type should match")
	assert.False(t, createdProfile.CreatedAt.IsZero(), "Created at should not be zero")
}

// TestCreateProfileUnauthorized тестирует попытку создания профиля без авторизации
func (s *ProfileIntegrationTestSuite) TestCreateProfileUnauthorized() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Данные для создания профиля импровизации
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	// Намеренно не устанавливаем заголовок Authorization

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем статус ответа - должен быть "Unauthorized"
	assert.Equal(t, http.StatusUnauthorized, createResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestCreateProfileWithInvalidData тестирует создание профилей с невалидными данными
func (s *ProfileIntegrationTestSuite) TestCreateProfileWithInvalidData() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Тестовые случаи с невалидными данными
	testCases := []struct {
		name        string
		requestData interface{}
	}{
		{
			name: "Unsupported activity type",
			requestData: profile.CreateProfileRequest{
				UserID:       userID,
				Description:  "Test description",
				ActivityType: "unsupported_type", // Неподдерживаемый тип активности
			},
		},
		{
			name: "Invalid user ID for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:       0, // Невалидный ID пользователя
					Description:  "Test description",
					ActivityType: profile.ActivityTypeImprov,
				},
				Goal:   "Hobby",
				Styles: []string{"Short Form"},
			},
		},
		{
			name: "Empty goal for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:       userID,
					Description:  "Test description",
					ActivityType: profile.ActivityTypeImprov,
				},
				Goal:   "", // Пустая цель
				Styles: []string{"Short Form"},
			},
		},
		{
			name: "Empty styles for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:       userID,
					Description:  "Test description",
					ActivityType: profile.ActivityTypeImprov,
				},
				Goal:   "Hobby",
				Styles: []string{}, // Пустой список стилей
			},
		},
		{
			name: "Invalid user ID for music profile",
			requestData: profile.CreateMusicProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:       0, // Невалидный ID пользователя
					Description:  "Test description",
					ActivityType: profile.ActivityTypeMusic,
				},
				Instruments: []string{"guitar"},
			},
		},
		{
			name: "Empty instruments for music profile",
			requestData: profile.CreateMusicProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:       userID,
					Description:  "Test description",
					ActivityType: profile.ActivityTypeMusic,
				},
				Instruments: []string{}, // Пустой список инструментов
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			createJSON, _ := json.Marshal(tc.requestData)
			createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
			createReq.Header.Set("Content-Type", "application/json")
			createReq.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			createResp, err := client.Do(createReq)
			assert.NoError(t, err)
			defer createResp.Body.Close()

			// Проверяем статус ответа - должен быть "Bad Request"
			assert.Equal(t, http.StatusBadRequest, createResp.StatusCode, "Should return status 400 Bad Request for case: "+tc.name)
		})
	}
}

// TestCreateDuplicateProfile тестирует повторное создание профиля для одного пользователя
func (s *ProfileIntegrationTestSuite) TestCreateDuplicateProfile() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Создаем первый профиль (импровизация)
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "First test profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем, что первый профиль создан успешно
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "First profile should be created successfully")

	// Пытаемся создать второй профиль для того же пользователя (музыкальный)
	duplicateData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Second test profile description",
			ActivityType: profile.ActivityTypeMusic,
		},
		Instruments: []string{"guitar"},
	}

	duplicateJSON, _ := json.Marshal(duplicateData)
	duplicateReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(duplicateJSON))
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateReq.Header.Set("Authorization", "Bearer "+token)

	duplicateResp, err := client.Do(duplicateReq)
	assert.NoError(t, err)
	defer duplicateResp.Body.Close()

	// Проверяем статус ответа - должен быть "Conflict", так как профиль уже создан
	assert.Equal(t, http.StatusConflict, duplicateResp.StatusCode, "Should return status 409 Conflict")
}

// TestCreateProfileForNonExistentUser тестирует создание профиля для несуществующего пользователя
func (s *ProfileIntegrationTestSuite) TestCreateProfileForNonExistentUser() {
	t := s.T()

	// Создаем нового пользователя только для получения токена
	_, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Используем заведомо несуществующий ID пользователя
	nonExistentUserID := 999999 // Предполагаем, что такого ID нет в базе

	// Данные для создания профиля с несуществующим ID пользователя
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       nonExistentUserID,
			Description:  "Test profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем статус ответа - должен быть "Not Found"
	assert.Equal(t, http.StatusNotFound, createResp.StatusCode, "Should return status 404 Not Found")
}

// TestGetProfile тестирует получение профиля
func (s *ProfileIntegrationTestSuite) TestGetProfile() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Создаем профиль импровизации
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test improv profile for get",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: true,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Убеждаемся, что профиль создан успешно
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Получаем ID созданного профиля
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Запрашиваем профиль по ID
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	defer getResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusOK, getResp.StatusCode, "Should return status 200 OK")

	// Проверяем содержимое ответа
	var profileResp profile.ProfileResponse
	err = json.NewDecoder(getResp.Body).Decode(&profileResp)
	assert.NoError(t, err)

	// Проверяем поля полученного профиля
	assert.NotNil(t, profileResp.ImprovProfile)
	assert.Equal(t, profileID, profileResp.ImprovProfile.ProfileID)
	assert.Equal(t, userID, profileResp.ImprovProfile.UserID)
	assert.Equal(t, "Test improv profile for get", profileResp.ImprovProfile.Description)
	assert.Equal(t, profile.ActivityTypeImprov, profileResp.ImprovProfile.ActivityType)

	// Проверяем детали профиля импровизации
	assert.Equal(t, "Hobby", profileResp.ImprovProfile.Goal)
	assert.Contains(t, profileResp.ImprovProfile.Styles, "Short Form")
	assert.True(t, profileResp.ImprovProfile.LookingForTeam)
}

// TestGetNonExistentProfile тестирует получение несуществующего профиля
func (s *ProfileIntegrationTestSuite) TestGetNonExistentProfile() {
	t := s.T()

	// Создаем нового пользователя только для получения токена
	_, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Запрашиваем несуществующий профиль по ID
	nonExistentID := 999999 // Предполагаем, что такого ID нет в базе
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, nonExistentID), nil)
	getReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	defer getResp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode, "Should return status 404 Not Found")
}

// TestGetCatalogs тестирует получение различных каталогов
func (s *ProfileIntegrationTestSuite) TestGetCatalogs() {
	t := s.T()

	// Создаем пользователя для авторизации
	_, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Тестовые URL для каталогов
	catalogURLs := []string{
		"/api/profiles/catalog/activity-types",
		"/api/profiles/catalog/improv-styles",
		"/api/profiles/catalog/improv-goals",
		"/api/profiles/catalog/music-genres",
		"/api/profiles/catalog/music-instruments",
	}

	// Проверяем все каталоги
	for _, url := range catalogURLs {
		t.Run(url, func(t *testing.T) {
			req, _ := http.NewRequest("GET", s.appUrl+url+"?lang=ru", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			// Проверяем статус ответа
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Проверяем, что в ответе есть данные
			body, _ := io.ReadAll(resp.Body)
			assert.True(t, len(body) > 2, "Response body should not be empty")
			assert.Contains(t, string(body), "[") // Проверяем, что это массив
		})
	}
}

// TestProfileIntegration запускает набор интеграционных тестов для профилей
func TestProfileIntegration(t *testing.T) {
	// Пропускаем тесты, если задана переменная окружения SKIP_INTEGRATION_TESTS
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(ProfileIntegrationTestSuite))
}

// TestUpdateImprovProfile tests updating an improv profile
func (s *ProfileIntegrationTestSuite) TestUpdateImprovProfile() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile first
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Initial improv profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created successfully
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Update the improv profile
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description:  "Updated improv profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:           "Career",
		Styles:         []string{"Short Form", "Long Form"},
		LookingForTeam: true,
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update status
	assert.Equal(t, http.StatusOK, updateResp.StatusCode, "Should return status 200 OK")

	// Check the updated profile data
	var updatedProfileResp profile.ProfileResponse
	err = json.NewDecoder(updateResp.Body).Decode(&updatedProfileResp)
	assert.NoError(t, err)

	// Verify the fields were updated correctly
	assert.NotNil(t, updatedProfileResp.ImprovProfile)
	assert.Equal(t, profileID, updatedProfileResp.ImprovProfile.ProfileID)
	assert.Equal(t, updateData.UpdateProfileRequest.Description, updatedProfileResp.ImprovProfile.Description)
	assert.Equal(t, updateData.Goal, updatedProfileResp.ImprovProfile.Goal)
	assert.ElementsMatch(t, updateData.Styles, updatedProfileResp.ImprovProfile.Styles)
	assert.True(t, updatedProfileResp.ImprovProfile.LookingForTeam)
}

// TestUpdateMusicProfile tests updating a music profile
func (s *ProfileIntegrationTestSuite) TestUpdateMusicProfile() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create a music profile first
	createData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Initial music profile description",
			ActivityType: profile.ActivityTypeMusic,
		},
		Genres:      []string{"rock"},
		Instruments: []string{"acoustic_guitar"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created successfully
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Update the music profile
	updateData := profile.UpdateMusicProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description:  "Updated music profile description",
			ActivityType: profile.ActivityTypeMusic,
		},
		Genres:      []string{"rock", "jazz", "pop"},
		Instruments: []string{"acoustic_guitar", "electric_guitar", "piano"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update status
	assert.Equal(t, http.StatusOK, updateResp.StatusCode, "Should return status 200 OK")

	// Check the updated profile data
	var updatedProfileResp profile.ProfileResponse
	err = json.NewDecoder(updateResp.Body).Decode(&updatedProfileResp)
	assert.NoError(t, err)

	// Verify the fields were updated correctly
	assert.NotNil(t, updatedProfileResp.MusicProfile)
	assert.Equal(t, profileID, updatedProfileResp.MusicProfile.ProfileID)
	assert.Equal(t, updateData.UpdateProfileRequest.Description, updatedProfileResp.MusicProfile.Description)
	assert.ElementsMatch(t, updateData.Genres, updatedProfileResp.MusicProfile.Genres)
	assert.ElementsMatch(t, updateData.Instruments, updatedProfileResp.MusicProfile.Instruments)
}

// TestUpdateProfileUnauthorized tests attempting to update a profile without authorization
func (s *ProfileIntegrationTestSuite) TestUpdateProfileUnauthorized() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile first
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Get the profile ID
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Attempt to update without authorization token
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description:  "Updated description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Career",
		Styles: []string{"Short Form"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	// Intentionally NOT setting Authorization header

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify status - should be Unauthorized
	assert.Equal(t, http.StatusUnauthorized, updateResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestUpdateProfileForbidden tests attempting to update another user's profile
func (s *ProfileIntegrationTestSuite) TestUpdateProfileForbidden() {
	t := s.T()

	// Create two users: one profile owner and one unauthorized user
	ownerID, ownerToken, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create profile owner")

	_, unauthorizedToken, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create unauthorized user")

	// Create a profile for the first user
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       ownerID,
			Description:  "Owner's profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+ownerToken)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Get the profile ID
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Second user attempts to update the first user's profile
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description:  "Unauthorized update",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Career",
		Styles: []string{"Short Form"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+unauthorizedToken) // Using unauthorized user's token

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify status - should be Forbidden
	assert.Equal(t, http.StatusForbidden, updateResp.StatusCode, "Should return status 403 Forbidden")
}

// TestUpdateProfileWithInvalidData tests updating profiles with invalid data
func (s *ProfileIntegrationTestSuite) TestUpdateProfileWithInvalidData() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create a profile first
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  "Test profile description",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Get the profile ID
	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Test cases with invalid data
	testCases := []struct {
		name        string
		requestData interface{}
	}{
		{
			name: "Unsupported activity type",
			requestData: profile.UpdateProfileRequest{
				Description:  "Test description",
				ActivityType: "unsupported_type", // Unsupported activity type
			},
		},
		{
			name: "Empty goal for improv profile",
			requestData: profile.UpdateImprovProfileRequest{
				UpdateProfileRequest: profile.UpdateProfileRequest{
					Description:  "Test description",
					ActivityType: profile.ActivityTypeImprov,
				},
				Goal:   "", // Empty goal
				Styles: []string{"Short Form"},
			},
		},
		{
			name: "Empty styles for improv profile",
			requestData: profile.UpdateImprovProfileRequest{
				UpdateProfileRequest: profile.UpdateProfileRequest{
					Description:  "Test description",
					ActivityType: profile.ActivityTypeImprov,
				},
				Goal:   "Hobby",
				Styles: []string{}, // Empty styles list
			},
		},
		{
			name: "Empty instruments for music profile",
			requestData: profile.UpdateMusicProfileRequest{
				UpdateProfileRequest: profile.UpdateProfileRequest{
					Description:  "Test description",
					ActivityType: profile.ActivityTypeMusic,
				},
				Instruments: []string{}, // Empty instruments list
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateJSON, _ := json.Marshal(tc.requestData)
			updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
			updateReq.Header.Set("Content-Type", "application/json")
			updateReq.Header.Set("Authorization", "Bearer "+token)

			updateResp, err := client.Do(updateReq)
			assert.NoError(t, err)
			defer updateResp.Body.Close()

			// Verify status - should be Bad Request
			assert.Equal(t, http.StatusBadRequest, updateResp.StatusCode, "Should return status 400 Bad Request for case: "+tc.name)
		})
	}
}

// TestUpdateNonExistentProfile tests updating a profile that doesn't exist
func (s *ProfileIntegrationTestSuite) TestUpdateNonExistentProfile() {
	t := s.T()

	// Create a new user just for the token
	_, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Use a presumably non-existent profile ID
	nonExistentProfileID := 999999

	// Try to update a non-existent profile
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description:  "Update for non-existent profile",
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, nonExistentProfileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify status - should be Not Found
	assert.Equal(t, http.StatusNotFound, updateResp.StatusCode, "Should return status 404 Not Found")
}
