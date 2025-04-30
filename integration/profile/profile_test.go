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

// TestCreateProfileUnauthorized тестирует попытку создания профиля без авторизации
func (s *ProfileIntegrationTestSuite) TestCreateProfileUnauthorized() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Данные для создания профиля импровизации
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Test profile description",
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	// Test both endpoints
	endpoints := []string{
		"/api/profiles/improv",
		"/api/profiles/music",
	}

	for _, endpoint := range endpoints {
		createJSON, _ := json.Marshal(createData)
		createReq, _ := http.NewRequest("POST", s.appUrl+endpoint, bytes.NewBuffer(createJSON))
		createReq.Header.Set("Content-Type", "application/json")
		// Намеренно не устанавливаем заголовок Authorization

		client := &http.Client{}
		createResp, err := client.Do(createReq)
		assert.NoError(t, err)
		defer createResp.Body.Close()

		// Проверяем статус ответа - должен быть "Unauthorized"
		assert.Equal(t, http.StatusUnauthorized, createResp.StatusCode, "Should return status 401 Unauthorized for "+endpoint)
	}
}

// TestCreateProfileWithInvalidData тестирует создание профилей с невалидными данными
func (s *ProfileIntegrationTestSuite) TestCreateProfileWithInvalidData() {
	t := s.T()

	// Создаем нового пользователя для этого теста
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Тестовые случаи с невалидными данными
	improvTestCases := []struct {
		name        string
		requestData profile.CreateImprovProfileRequest
	}{
		{
			name: "Invalid user ID for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:      0, // Невалидный ID пользователя
					Description: "Test description",
				},
				Goal:   "Hobby",
				Styles: []string{"Short Form"},
			},
		},
		{
			name: "Empty goal for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:      userID,
					Description: "Test description",
				},
				Goal:   "", // Пустая цель
				Styles: []string{"Short Form"},
			},
		},
		{
			name: "Empty styles for improv profile",
			requestData: profile.CreateImprovProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:      userID,
					Description: "Test description",
				},
				Goal:   "Hobby",
				Styles: []string{}, // Пустой список стилей
			},
		},
	}

	musicTestCases := []struct {
		name        string
		requestData profile.CreateMusicProfileRequest
	}{
		{
			name: "Invalid user ID for music profile",
			requestData: profile.CreateMusicProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:      0, // Невалидный ID пользователя
					Description: "Test description",
				},
				Instruments: []string{"electric_guitar"},
			},
		},
		{
			name: "Empty instruments for music profile",
			requestData: profile.CreateMusicProfileRequest{
				CreateProfileRequest: profile.CreateProfileRequest{
					UserID:      userID,
					Description: "Test description",
				},
				Instruments: []string{}, // Пустой список инструментов
			},
		},
	}

	// Test improv profile invalid cases
	for _, tc := range improvTestCases {
		t.Run("Improv: "+tc.name, func(t *testing.T) {
			createJSON, _ := json.Marshal(tc.requestData)
			createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
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

	// Test music profile invalid cases
	for _, tc := range musicTestCases {
		t.Run("Music: "+tc.name, func(t *testing.T) {
			createJSON, _ := json.Marshal(tc.requestData)
			createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/music", bytes.NewBuffer(createJSON))
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
			UserID:      userID,
			Description: "First test profile description",
		},
		Goal:   "Hobby",
		Styles: []string{"Short Form"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Проверяем, что первый профиль создан успешно
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "First profile should be created successfully")

	// Пытаемся создать второй профиль для того же пользователя (музыкальный)
	duplicateData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Second test profile description",
		},
		Goal:   "Career",
		Styles: []string{"Long Form"},
	}

	duplicateJSON, _ := json.Marshal(duplicateData)
	duplicateReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(duplicateJSON))
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateReq.Header.Set("Authorization", "Bearer "+token)

	duplicateResp, err := client.Do(duplicateReq)
	assert.NoError(t, err)
	defer duplicateResp.Body.Close()

	// Проверяем статус ответа - должен быть "Conflict", так как профиль уже создан
	assert.Equal(t, http.StatusConflict, duplicateResp.StatusCode, "Should return status 409 Conflict")
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

// TestCreateImprovProfile tests creating an improv profile using the dedicated endpoint
func (s *ProfileIntegrationTestSuite) TestCreateImprovProfile() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Data for creating an improv profile with the specific endpoint
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Test improv profile via specific endpoint",
			// ActivityType is intentionally omitted as it should be set by the server
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: true,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "Should return status 201 Created")

	// Check response content
	var createdProfile profile.ImprovProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)

	// Verify profile fields
	assert.NotZero(t, createdProfile.ProfileID, "Profile ID should not be zero")
	assert.Equal(t, userID, createdProfile.UserID, "User ID should match")
	assert.Equal(t, "Test improv profile via specific endpoint", createdProfile.Description, "Description should match")
	assert.Equal(t, profile.ActivityTypeImprov, createdProfile.ActivityType, "Activity type should be improv")
	assert.Equal(t, "Hobby", createdProfile.Goal, "Goal should match")
	assert.ElementsMatch(t, []string{"Short Form"}, createdProfile.Styles, "Styles should match")
	assert.True(t, createdProfile.LookingForTeam, "LookingForTeam should be true")
}

// TestCreateMusicProfile tests creating a music profile using the dedicated endpoint
func (s *ProfileIntegrationTestSuite) TestCreateMusicProfile() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Data for creating a music profile with the specific endpoint
	createData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Test music profile via specific endpoint",
			// ActivityType is intentionally omitted as it should be set by the server
		},
		Genres:      []string{"rock", "jazz"},
		Instruments: []string{"electric_guitar", "piano"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/music", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusCreated, createResp.StatusCode, "Should return status 201 Created")

	// Check response content
	var createdProfile profile.MusicProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)

	// Verify profile fields
	assert.NotZero(t, createdProfile.ProfileID, "Profile ID should not be zero")
	assert.Equal(t, userID, createdProfile.UserID, "User ID should match")
	assert.Equal(t, "Test music profile via specific endpoint", createdProfile.Description, "Description should match")
	assert.Equal(t, profile.ActivityTypeMusic, createdProfile.ActivityType, "Activity type should be music")
	assert.ElementsMatch(t, []string{"rock", "jazz"}, createdProfile.Genres, "Genres should match")
	assert.ElementsMatch(t, []string{"electric_guitar", "piano"}, createdProfile.Instruments, "Instruments should match")
}

// TestUpdateImprovProfileWithSpecificEndpoint tests updating an improv profile using the dedicated endpoint
func (s *ProfileIntegrationTestSuite) TestUpdateImprovProfileWithSpecificEndpoint() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile first
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Initial improv profile for specific update",
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.ImprovProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Now update using the specific endpoint
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description: "Updated via specific improv endpoint",
			// ActivityType is intentionally omitted as it should be set by the server
		},
		Goal:           "Career",
		Styles:         []string{"Long Form", "Short Form"},
		LookingForTeam: true,
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d/improv", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update status
	assert.Equal(t, http.StatusOK, updateResp.StatusCode, "Should return status 200 OK")

	// Check updated profile data
	var updatedProfile profile.ImprovProfile
	err = json.NewDecoder(updateResp.Body).Decode(&updatedProfile)
	assert.NoError(t, err)

	// Verify the fields were updated correctly
	assert.Equal(t, profileID, updatedProfile.ProfileID)
	assert.Equal(t, "Updated via specific improv endpoint", updatedProfile.Description)
	assert.Equal(t, "Career", updatedProfile.Goal)
	assert.ElementsMatch(t, []string{"Long Form", "Short Form"}, updatedProfile.Styles)
	assert.True(t, updatedProfile.LookingForTeam)
}

// TestUpdateMusicProfileWithSpecificEndpoint tests updating a music profile using the dedicated endpoint
func (s *ProfileIntegrationTestSuite) TestUpdateMusicProfileWithSpecificEndpoint() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create a music profile first
	createData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Initial music profile for specific update",
		},
		Genres:      []string{"rock"},
		Instruments: []string{"electric_guitar"},
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/music", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.MusicProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Now update using the specific endpoint
	updateData := profile.UpdateMusicProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description: "Updated via specific music endpoint",
			// ActivityType is intentionally omitted as it should be set by the server
		},
		Genres:      []string{"rock", "jazz", "pop"},
		Instruments: []string{"electric_guitar", "piano", "drums"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d/music", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update status
	assert.Equal(t, http.StatusOK, updateResp.StatusCode, "Should return status 200 OK")

	// Check updated profile data
	var updatedProfile profile.MusicProfile
	err = json.NewDecoder(updateResp.Body).Decode(&updatedProfile)
	assert.NoError(t, err)

	// Verify the fields were updated correctly
	assert.Equal(t, profileID, updatedProfile.ProfileID)
	assert.Equal(t, "Updated via specific music endpoint", updatedProfile.Description)
	assert.ElementsMatch(t, []string{"rock", "jazz", "pop"}, updatedProfile.Genres)
	assert.ElementsMatch(t, []string{"electric_guitar", "piano", "drums"}, updatedProfile.Instruments)
}

// TestUpdateProfileUnauthorized tests updating a profile without authorization
func (s *ProfileIntegrationTestSuite) TestUpdateProfileUnauthorized() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create a profile first
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Profile for unauthorized update test",
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.ImprovProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Attempt to update without authorization
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description: "Unauthorized update attempt",
		},
		Goal:           "Career",
		Styles:         []string{"Long Form"},
		LookingForTeam: true,
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d/improv", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	// Intentionally omit authorization header

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update is rejected with unauthorized status
	assert.Equal(t, http.StatusUnauthorized, updateResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestUpdateProfileWrongType tests updating a profile with the wrong endpoint
func (s *ProfileIntegrationTestSuite) TestUpdateProfileWrongType() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Improv profile for wrong type update test",
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.ImprovProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Try to update an improv profile with the music endpoint
	updateData := profile.UpdateMusicProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description: "Wrong type update attempt",
		},
		Genres:      []string{"rock"},
		Instruments: []string{"electric_guitar"},
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d/music", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token)

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update is rejected with bad request status
	assert.Equal(t, http.StatusBadRequest, updateResp.StatusCode, "Should return status 400 Bad Request for wrong profile type")
}

// TestUpdateProfileOtherUsersProfile tests attempting to update another user's profile
func (s *ProfileIntegrationTestSuite) TestUpdateProfileOtherUsersProfile() {
	t := s.T()

	// Create two users
	userID1, token1, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create first test user")

	_, token2, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create second test user")

	// Create profile for first user
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID1,
			Description: "Profile for access control test",
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token1)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	assert.NoError(t, err)
	defer createResp.Body.Close()

	// Verify profile was created
	assert.Equal(t, http.StatusCreated, createResp.StatusCode)

	// Get the profile ID
	var createdProfile profile.ImprovProfile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	assert.NoError(t, err)
	profileID := createdProfile.ProfileID

	// Second user attempts to update first user's profile
	updateData := profile.UpdateImprovProfileRequest{
		UpdateProfileRequest: profile.UpdateProfileRequest{
			Description: "Unauthorized user update attempt",
		},
		Goal:           "Career",
		Styles:         []string{"Long Form"},
		LookingForTeam: true,
	}

	updateJSON, _ := json.Marshal(updateData)
	updateReq, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/profiles/%d/improv", s.appUrl, profileID), bytes.NewBuffer(updateJSON))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+token2) // Using second user's token

	updateResp, err := client.Do(updateReq)
	assert.NoError(t, err)
	defer updateResp.Body.Close()

	// Verify update is rejected with forbidden status
	assert.Equal(t, http.StatusForbidden, updateResp.StatusCode, "Should return status 403 Forbidden")
}

// TestGetUserProfiles tests the retrieval of all profiles for a user
func (s *ProfileIntegrationTestSuite) TestGetUserProfiles() {
	t := s.T()

	// Create a new user for this test
	userID, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile first
	improvData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Test improv profile for profiles listing",
		},
		Goal:           "Hobby",
		Styles:         []string{"Short Form"},
		LookingForTeam: false,
	}

	improvJSON, _ := json.Marshal(improvData)
	improvReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/improv", bytes.NewBuffer(improvJSON))
	improvReq.Header.Set("Content-Type", "application/json")
	improvReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	improvResp, err := client.Do(improvReq)
	assert.NoError(t, err)
	defer improvResp.Body.Close()

	// Verify improv profile was created
	assert.Equal(t, http.StatusCreated, improvResp.StatusCode)

	// Get the improv profile ID
	var createdImprovProfile profile.ImprovProfile
	err = json.NewDecoder(improvResp.Body).Decode(&createdImprovProfile)
	assert.NoError(t, err)
	improvProfileID := createdImprovProfile.ProfileID

	// Now create a music profile for the same user
	musicData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:      userID,
			Description: "Test music profile for profiles listing",
		},
		Genres:      []string{"rock"},
		Instruments: []string{"electric_guitar"},
	}

	musicJSON, _ := json.Marshal(musicData)
	musicReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles/music", bytes.NewBuffer(musicJSON))
	musicReq.Header.Set("Content-Type", "application/json")
	musicReq.Header.Set("Authorization", "Bearer "+token)

	musicResp, err := client.Do(musicReq)
	assert.NoError(t, err)
	defer musicResp.Body.Close()

	// Verify music profile was created
	assert.Equal(t, http.StatusCreated, musicResp.StatusCode)

	// Get the music profile ID
	var createdMusicProfile profile.MusicProfile
	err = json.NewDecoder(musicResp.Body).Decode(&createdMusicProfile)
	assert.NoError(t, err)
	musicProfileID := createdMusicProfile.ProfileID

	// Test that we can get all the user's profiles
	profilesReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID), nil)
	profilesReq.Header.Set("Authorization", "Bearer "+token)

	profilesResp, err := client.Do(profilesReq)
	assert.NoError(t, err)
	defer profilesResp.Body.Close()

	// Verify we can get profiles
	assert.Equal(t, http.StatusOK, profilesResp.StatusCode)

	// Parse the response
	var userProfiles profile.UserProfilesResponse
	err = json.NewDecoder(profilesResp.Body).Decode(&userProfiles)
	assert.NoError(t, err)

	// Verify we got both profiles
	assert.Contains(t, userProfiles.Profiles, profile.ActivityTypeImprov)
	assert.Equal(t, improvProfileID, userProfiles.Profiles[profile.ActivityTypeImprov])

	assert.Contains(t, userProfiles.Profiles, profile.ActivityTypeMusic)
	assert.Equal(t, musicProfileID, userProfiles.Profiles[profile.ActivityTypeMusic])

	// Verify we have exactly two profiles (one of each type)
	assert.Equal(t, 2, len(userProfiles.Profiles), "User should have exactly two profiles")

	// Test unauthorized access
	otherUserReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID+1), nil)
	otherUserReq.Header.Set("Authorization", "Bearer "+token)

	otherUserResp, err := client.Do(otherUserReq)
	assert.NoError(t, err)
	defer otherUserResp.Body.Close()

	// Verify forbidden access to another user's profiles
	assert.Equal(t, http.StatusForbidden, otherUserResp.StatusCode)

	// Test unauthorized request
	noAuthReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID), nil)

	noAuthResp, err := client.Do(noAuthReq)
	assert.NoError(t, err)
	defer noAuthResp.Body.Close()

	// Verify unauthorized request
	assert.Equal(t, http.StatusUnauthorized, noAuthResp.StatusCode)
}

// TestGetUserProfilesInvalidID tests error handling for invalid user IDs
func (s *ProfileIntegrationTestSuite) TestGetUserProfilesInvalidID() {
	t := s.T()

	// Create a user for authentication
	_, token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Test with non-numeric user ID
	invalidReq, _ := http.NewRequest("GET", s.appUrl+"/api/users/invalid/profiles", nil)
	invalidReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	invalidResp, err := client.Do(invalidReq)
	assert.NoError(t, err)
	defer invalidResp.Body.Close()

	// Verify bad request response
	assert.Equal(t, http.StatusBadRequest, invalidResp.StatusCode)

	// Test with user ID = 0 (invalid)
	zeroIDReq, _ := http.NewRequest("GET", s.appUrl+"/api/users/0/profiles", nil)
	zeroIDReq.Header.Set("Authorization", "Bearer "+token)

	zeroIDResp, err := client.Do(zeroIDReq)
	assert.NoError(t, err)
	defer zeroIDResp.Body.Close()

	// Verify bad request response
	assert.Equal(t, http.StatusBadRequest, zeroIDResp.StatusCode)
}

// TestProfileIntegration запускает набор интеграционных тестов для профилей
func TestProfileIntegration(t *testing.T) {
	// Пропускаем тесты, если задана переменная окружения SKIP_INTEGRATION_TESTS
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(ProfileIntegrationTestSuite))
}
