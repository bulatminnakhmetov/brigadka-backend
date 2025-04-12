package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Создаем мок для ProfileService
type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) CreateProfile(userID int, description string, activityType string) (*Profile, error) {
	args := m.Called(userID, description, activityType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Profile), args.Error(1)
}

func TestCreateProfileHandler(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Подготовка тестовых данных
		req := CreateProfileRequest{
			UserID:       1,
			Description:  "Test description",
			ActivityType: "sports",
		}

		profile := &Profile{
			ProfileID:    1,
			UserID:       req.UserID,
			Description:  req.Description,
			ActivityType: req.ActivityType,
			CreatedAt:    time.Now(),
		}

		// Настройка ожидаемого поведения мока
		mockService.On("CreateProfile", req.UserID, req.Description, req.ActivityType).Return(profile, nil)

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
		assert.Equal(t, http.StatusCreated, recorder.Code)

		var responseProfile Profile
		err = json.Unmarshal(recorder.Body.Bytes(), &responseProfile)
		assert.NoError(t, err)
		assert.Equal(t, profile.ProfileID, responseProfile.ProfileID)
		assert.Equal(t, profile.UserID, responseProfile.UserID)
		assert.Equal(t, profile.Description, responseProfile.Description)
		assert.Equal(t, profile.ActivityType, responseProfile.ActivityType)

		// Проверка вызовов мока
		mockService.AssertExpectations(t)
	})

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
		mockService.AssertNotCalled(t, "CreateProfile")
	})

	t.Run("Invalid user_id", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Подготовка тестовых данных с невалидным user_id
		req := CreateProfileRequest{
			UserID:       0, // Невалидный ID
			Description:  "Test description",
			ActivityType: "sports",
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
		mockService.AssertNotCalled(t, "CreateProfile")
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
		mockService.AssertNotCalled(t, "CreateProfile")
	})

	t.Run("Service error", func(t *testing.T) {
		// Инициализация мока
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Подготовка тестовых данных
		req := CreateProfileRequest{
			UserID:       1,
			Description:  "Test description",
			ActivityType: "sports",
		}

		// Настройка ожидаемого поведения мока - возвращаем ошибку
		serviceError := errors.New("service error")
		mockService.On("CreateProfile", req.UserID, req.Description, req.ActivityType).Return(nil, serviceError)

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
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)

		// Проверка вызовов мока
		mockService.AssertExpectations(t)
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
		mockService.AssertNotCalled(t, "CreateProfile")
	})
}
