package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Создаем мок для ProfileService
type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) CreateImprovProfile(userID int, description string, goal string, styles []string, lookingForTeam bool) (*ImprovProfile, error) {
	args := m.Called(userID, description, goal, styles, lookingForTeam)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImprovProfile), args.Error(1)
}

func (m *MockProfileService) CreateMusicProfile(userID int, description string, genres []string, instruments []string) (*MusicProfile, error) {
	args := m.Called(userID, description, genres, instruments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MusicProfile), args.Error(1)
}

func (m *MockProfileService) GetProfile(profileID int) (*ProfileResponse, error) {
	args := m.Called(profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProfileResponse), args.Error(1)
}

func (m *MockProfileService) GetActivityTypes(lang string) (ActivityTypeCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ActivityTypeCatalog), args.Error(1)
}

func (m *MockProfileService) GetImprovStyles(lang string) (ImprovStyleCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ImprovStyleCatalog), args.Error(1)
}

func (m *MockProfileService) GetImprovGoals(lang string) (ImprovGoalCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ImprovGoalCatalog), args.Error(1)
}

func (m *MockProfileService) GetMusicGenres(lang string) (MusicGenreCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(MusicGenreCatalog), args.Error(1)
}

func (m *MockProfileService) GetMusicInstruments(lang string) (MusicInstrumentCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(MusicInstrumentCatalog), args.Error(1)
}

func (m *MockProfileService) UpdateImprovProfile(profileID int, description string, goal string, styles []string, lookingForTeam bool) (*ImprovProfile, error) {
	args := m.Called(profileID, description, goal, styles, lookingForTeam)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImprovProfile), args.Error(1)
}

func (m *MockProfileService) UpdateMusicProfile(profileID int, description string, genres []string, instruments []string) (*MusicProfile, error) {
	args := m.Called(profileID, description, genres, instruments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MusicProfile), args.Error(1)
}

func TestCreateProfileHandler(t *testing.T) {
	t.Run("Invalid request body", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создание запроса с невалидным JSON
		request, err := http.NewRequest(http.MethodPost, "/api/profiles", bytes.NewBuffer([]byte("invalid json")))
		if err != nil {
			t.Fatal(err)
		}

		// Создание ResponseRecorder для записи ответа
		recorder := httptest.NewRecorder()

		// Вызов обработчика
		handler.CreateProfile(recorder, request)

		// Проверка ответа
		assert.Equal(t, http.StatusBadRequest, recorder.Code)

		// Мок не должен вызываться
		mockService.AssertNotCalled(t, "CreateImprovProfile")
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Missing activity type", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Подготовка тестовых данных с пустым activity_type
		req := CreateProfileRequest{
			UserID:       1,
			Description:  "Test description",
			ActivityType: "", // Пустой тип активности
		}

		// Создание запроса
		reqBody, _ := json.Marshal(req)
		request, err := http.NewRequest(http.MethodPost, "/api/profiles", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatal(err)
		}

		// Создание ResponseRecorder для записи ответа
		recorder := httptest.NewRecorder()

		// Вызов обработчика
		handler.CreateProfile(recorder, request)

		// Проверка ответа
		assert.Equal(t, http.StatusBadRequest, recorder.Code)

		// Мок не должен вызываться
		mockService.AssertNotCalled(t, "CreateImprovProfile")
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Method not allowed", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создание GET запроса вместо POST
		request, err := http.NewRequest(http.MethodGet, "/api/profiles", nil)
		if err != nil {
			t.Fatal(err)
		}

		// Создание ResponseRecorder для записи ответа
		recorder := httptest.NewRecorder()

		// Вызов обработчика
		handler.CreateProfile(recorder, request)

		// Проверка ответа
		assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)

		// Мок не должен вызываться
		mockService.AssertNotCalled(t, "CreateImprovProfile")
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	// Тест на создание профиля импровизации
	t.Run("CreateImprovProfile", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый профиль
		testProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		// Настраиваем поведение мока с looking_for_team как последним параметром
		mockService.On("CreateImprovProfile", 1, "Test Description", "Hobby", []string{"Short Form"}, true).Return(testProfile, nil)

		// Создаем тестовый запрос
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true, // Устанавливаем флаг
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)

		// Проверяем тело ответа
		var response ImprovProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, testProfile.ActivityType, response.ActivityType)
		assert.Equal(t, testProfile.Goal, response.Goal)
		assert.Equal(t, testProfile.Styles, response.Styles)
		assert.Equal(t, testProfile.LookingForTeam, response.LookingForTeam)
	})

	// Тест на создание музыкального профиля
	t.Run("CreateMusicProfile", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый профиль
		testProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}

		// Настраиваем поведение мока
		mockService.On("CreateMusicProfile", 1, "Test Description", []string{"rock"}, []string{"guitar"}).Return(testProfile, nil)

		// Создаем тестовый запрос
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeMusic,
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)

		// Проверяем тело ответа
		var response MusicProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, testProfile.ActivityType, response.ActivityType)
		assert.Equal(t, testProfile.Genres, response.Genres)
		assert.Equal(t, testProfile.Instruments, response.Instruments)
	})

	// Тест на попытку создания профиля с неподдерживаемым типом активности
	t.Run("UnsupportedActivityType", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый запрос с неподдерживаемым типом активности
		reqData := CreateProfileRequest{
			UserID:       1,
			Description:  "Test Description",
			ActivityType: "unsupported",
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа - должен быть BadRequest
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем сообщение об ошибке
		assert.Contains(t, rr.Body.String(), "Unsupported activity type")
	})

	// Тест на обработку ошибки от сервиса
	t.Run("ServiceError", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Настраиваем мок на возврат ошибки
		mockService.On("CreateImprovProfile", 1, "Test Description", "Hobby", []string{"Short Form"}, true).Return(nil, ErrUserNotFound)

		// Создаем тестовый запрос
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true, // Устанавливаем флаг
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа - должен соответствовать ошибке
		assert.Equal(t, http.StatusNotFound, rr.Code)

		// Проверяем сообщение об ошибке
		assert.Contains(t, rr.Body.String(), "User not found")
	})

	// Проверка валидации для импровизации (пустая цель)
	t.Run("EmptyImprovGoal", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый запрос с пустой целью
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
			},
			Goal:   "", // Пустая цель
			Styles: []string{"Short Form"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем сообщение
		assert.Contains(t, rr.Body.String(), "Improv goal is required")
	})

	// Проверка валидации для импровизации (пустые стили)
	t.Run("EmptyImprovStyles", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый запрос с пустым списком стилей
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
			},
			Goal:   "Hobby",
			Styles: []string{}, // Пустой список стилей
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем сообщение
		assert.Contains(t, rr.Body.String(), "At least one improv style is required")
	})

	// Проверка валидации для музыкального профиля (пустые инструменты)
	t.Run("EmptyMusicInstruments", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый запрос с пустым списком инструментов
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeMusic,
			},
			Genres:      []string{"rock"},
			Instruments: []string{}, // Пустой список инструментов
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.CreateProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Проверяем сообщение
		assert.Contains(t, rr.Body.String(), "At least one instrument is required")
	})
}

func TestGetProfileHandler(t *testing.T) {
	// Тест успешного получения профиля импровизации
	t.Run("Success - Improv Profile", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый профиль импровизации
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		// Создаем тестовый ответ
		testResponse := &ProfileResponse{
			ImprovProfile: improvProfile,
		}

		// Настраиваем поведение мока
		mockService.On("GetProfile", 1).Return(testResponse, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/1", nil)

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)

		// Проверяем тело ответа
		var response ProfileResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.ImprovProfile)
		assert.Equal(t, improvProfile.ProfileID, response.ImprovProfile.ProfileID)
		assert.Equal(t, improvProfile.Goal, response.ImprovProfile.Goal)
		assert.Equal(t, improvProfile.Styles, response.ImprovProfile.Styles)
		assert.Equal(t, improvProfile.LookingForTeam, response.ImprovProfile.LookingForTeam)
		assert.Nil(t, response.MusicProfile)
	})

	// Тест успешного получения музыкального профиля
	t.Run("Success - Music Profile", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый музыкальный профиль
		musicProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       2,
				Description:  "Music Profile",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock", "jazz"},
			Instruments: []string{"guitar", "piano"},
		}

		// Создаем тестовый ответ
		testResponse := &ProfileResponse{
			MusicProfile: musicProfile,
		}

		// Настраиваем поведение мока
		mockService.On("GetProfile", 2).Return(testResponse, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/2", nil)

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)

		// Проверяем тело ответа
		var response ProfileResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response.MusicProfile)
		assert.Equal(t, musicProfile.ProfileID, response.MusicProfile.ProfileID)
		assert.Equal(t, musicProfile.Genres, response.MusicProfile.Genres)
		assert.Equal(t, musicProfile.Instruments, response.MusicProfile.Instruments)
		assert.Nil(t, response.ImprovProfile)
	})

	// Тест обработки ошибки "профиль не найден"
	t.Run("ProfileNotFound", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Настраиваем мок на возврат ошибки
		mockService.On("GetProfile", 999).Return(nil, ErrProfileNotFound)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/999", nil)

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetProfile(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusNotFound, rr.Code)

		// Проверяем сообщение об ошибке
		assert.Contains(t, rr.Body.String(), "Profile not found")
	})
}

func TestUpdateProfileHandler(t *testing.T) {
	t.Run("Method not allowed", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create a GET request instead of PUT
		req, _ := http.NewRequest("GET", "/api/profiles/1", nil)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 405 Method Not Allowed
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

		// The mock should not be called
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Invalid URL", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create a request with invalid URL
		req, _ := http.NewRequest("PUT", "/api/profiles", nil)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// The mock should not be called
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Invalid profile ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create a request with non-numeric profile ID
		req, _ := http.NewRequest("PUT", "/api/profiles/invalid", nil)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// The mock should not be called
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create a request with valid profile ID but no user ID in context
		req, _ := http.NewRequest("PUT", "/api/profiles/1", strings.NewReader(`{"activity_type": "improv"}`))
		req.Header.Set("Content-Type", "application/json")

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 401 Unauthorized
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		// The mock should not be called
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Profile not found", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return profile not found error
		mockService.On("GetProfile", 999).Return(nil, ErrProfileNotFound)

		// Create a request with valid data but non-existent profile
		req, _ := http.NewRequest("PUT", "/api/profiles/999", strings.NewReader(`{"activity_type": "improv"}`))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 404 Not Found
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile not found")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Forbidden - different user", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create a profile that belongs to user 2
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       2, // Different from the requesting user
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		// Setup mock to return the profile
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: improvProfile}, nil)

		// Create a request with valid data
		req, _ := http.NewRequest("PUT", "/api/profiles/1", strings.NewReader(`{"activity_type": "improv"}`))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context (different from profile owner)
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 403 Forbidden
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "Forbidden")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return a valid profile
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: improvProfile}, nil)

		// Create a request with invalid JSON
		req, _ := http.NewRequest("PUT", "/api/profiles/1", strings.NewReader(`invalid json`))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request body")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Missing activity type", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return a valid profile
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: improvProfile}, nil)

		// Create a request with missing activity_type
		req, _ := http.NewRequest("PUT", "/api/profiles/1", strings.NewReader(`{"description": "New description"}`))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Activity type is required")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Successful improv profile update", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: originalProfile}, nil)

		// Setup updated profile
		updatedProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Updated description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Career",
			Styles:         []string{"Long Form", "Short Form"},
			LookingForTeam: false,
		}
		mockService.On("UpdateImprovProfile", 1, "Updated description", "Career", []string{"Long Form", "Short Form"}, false).Return(updatedProfile, nil)

		// Create request body
		reqBody := map[string]interface{}{
			"description":      "Updated description",
			"activity_type":    ActivityTypeImprov,
			"goal":             "Career",
			"styles":           []string{"Long Form", "Short Form"},
			"looking_for_team": false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Parse response
		var response ProfileResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response data
		assert.NotNil(t, response.ImprovProfile)
		assert.Equal(t, updatedProfile.ProfileID, response.ImprovProfile.ProfileID)
		assert.Equal(t, updatedProfile.UserID, response.ImprovProfile.UserID)
		assert.Equal(t, updatedProfile.Description, response.ImprovProfile.Description)
		assert.Equal(t, updatedProfile.ActivityType, response.ImprovProfile.ActivityType)
		assert.Equal(t, updatedProfile.Goal, response.ImprovProfile.Goal)
		assert.Equal(t, updatedProfile.Styles, response.ImprovProfile.Styles)
		assert.Equal(t, updatedProfile.LookingForTeam, response.ImprovProfile.LookingForTeam)
	})

	t.Run("Successful music profile update", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       1,
				Description:  "Original music description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetProfile", 2).Return(&ProfileResponse{MusicProfile: originalProfile}, nil)

		// Setup updated profile
		updatedProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       1,
				Description:  "Updated music description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock", "jazz", "blues"},
			Instruments: []string{"guitar", "piano"},
		}
		mockService.On("UpdateMusicProfile", 2, "Updated music description", []string{"rock", "jazz", "blues"}, []string{"guitar", "piano"}).Return(updatedProfile, nil)

		// Create request body
		reqBody := map[string]interface{}{
			"description":   "Updated music description",
			"activity_type": ActivityTypeMusic,
			"genres":        []string{"rock", "jazz", "blues"},
			"instruments":   []string{"guitar", "piano"},
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/2", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 200 OK
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Parse response
		var response ProfileResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response data
		assert.NotNil(t, response.MusicProfile)
		assert.Equal(t, updatedProfile.ProfileID, response.MusicProfile.ProfileID)
		assert.Equal(t, updatedProfile.UserID, response.MusicProfile.UserID)
		assert.Equal(t, updatedProfile.Description, response.MusicProfile.Description)
		assert.Equal(t, updatedProfile.ActivityType, response.MusicProfile.ActivityType)
		assert.ElementsMatch(t, updatedProfile.Genres, response.MusicProfile.Genres)
		assert.ElementsMatch(t, updatedProfile.Instruments, response.MusicProfile.Instruments)
	})

	t.Run("Validation error - empty improv goal", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: originalProfile}, nil)

		// Create request with empty goal
		reqBody := map[string]interface{}{
			"description":      "Updated description",
			"activity_type":    ActivityTypeImprov,
			"goal":             "", // Empty goal
			"styles":           []string{"Long Form"},
			"looking_for_team": false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Improv goal is required")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Validation error - empty improv styles", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: originalProfile}, nil)

		// Create request with empty styles
		reqBody := map[string]interface{}{
			"description":      "Updated description",
			"activity_type":    ActivityTypeImprov,
			"goal":             "Career",
			"styles":           []string{}, // Empty styles
			"looking_for_team": false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one improv style is required")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Validation error - empty music instruments", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       1,
				Description:  "Original music description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetProfile", 2).Return(&ProfileResponse{MusicProfile: originalProfile}, nil)

		// Create request with empty instruments
		reqBody := map[string]interface{}{
			"description":   "Updated music description",
			"activity_type": ActivityTypeMusic,
			"genres":        []string{"rock", "jazz"},
			"instruments":   []string{}, // Empty instruments
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/2", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one instrument is required")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Service error", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: originalProfile}, nil)

		// Setup service to return an error
		mockService.On("UpdateImprovProfile", 1, "Updated description", "Career", []string{"Long Form"}, false).Return(nil, ErrInvalidImprovGoal)

		// Create request body
		reqBody := map[string]interface{}{
			"description":      "Updated description",
			"activity_type":    ActivityTypeImprov,
			"goal":             "Career",
			"styles":           []string{"Long Form"},
			"looking_for_team": false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is appropriate for the error
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid improv goal")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Unsupported activity type", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetProfile", 1).Return(&ProfileResponse{ImprovProfile: originalProfile}, nil)

		// Create request with unsupported activity type
		reqBody := map[string]interface{}{
			"description":   "Updated description",
			"activity_type": "unsupported_type",
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder to record the response
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateProfile(rr, req)

		// Check the status code is 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Unsupported activity type")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}
