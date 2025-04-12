package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// Мок для репозитория пользователей
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetUserByEmail(email string) (*User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepository) CreateUser(user *User) error {
	args := m.Called(user)
	return args.Error(0)
}

func TestLogin(t *testing.T) {
	// Создаем мок репозитория
	mockRepo := new(MockUserRepository)

	// Создаем тестовый JWT секрет
	jwtSecret := "test-secret-key"

	// Создаем хендлер с моком
	handler := NewAuthHandler(mockRepo, jwtSecret)

	t.Run("Success case", func(t *testing.T) {
		// Хэшируем тестовый пароль
		password := "password123"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		// Создаем тестового пользователя
		user := &User{
			UserID:       1,
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			FullName:     "Test User",
		}

		// Настраиваем ожидаемый вызов мока
		mockRepo.On("GetUserByEmail", user.Email).Return(user, nil).Once()

		// Создаем тестовый запрос
		loginReq := LoginRequest{
			Email:    user.Email,
			Password: password,
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		// Создаем ResponseRecorder для записи ответа
		rr := httptest.NewRecorder()

		// Вызываем тестируемый метод
		handler.Login(rr, req)

		// Проверяем код ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем структуру ответа
		var response AuthResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Проверяем, что токен присутствует
		assert.NotEmpty(t, response.Token)

		// Проверяем пользователя в ответе
		assert.Equal(t, user.UserID, response.User.UserID)
		assert.Equal(t, user.Email, response.User.Email)
		assert.Equal(t, user.FullName, response.User.FullName)
		assert.Empty(t, response.User.PasswordHash) // Пароль не должен быть в ответе

		// Проверяем, что мок был вызван по ожиданиям
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		// Создаем невалидный JSON в запросе
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockRepo.AssertNotCalled(t, "GetUserByEmail")
	})

	t.Run("User not found", func(t *testing.T) {
		email := "nonexistent@example.com"
		password := "password123"

		// Настройка мока - пользователь не найден
		mockRepo.On("GetUserByEmail", email).Return(nil, errors.New("user not found")).Once()

		loginReq := LoginRequest{
			Email:    email,
			Password: password,
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid password", func(t *testing.T) {
		// Хэшируем правильный пароль
		correctPassword := "password123"
		incorrectPassword := "wrongpassword"
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.DefaultCost)

		user := &User{
			UserID:       1,
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
			FullName:     "Test User",
		}

		mockRepo.On("GetUserByEmail", user.Email).Return(user, nil).Once()

		loginReq := LoginRequest{
			Email:    user.Email,
			Password: incorrectPassword, // Отправляем неверный пароль
		}
		body, _ := json.Marshal(loginReq)
		req, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Login(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestRegister(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSecret := "test-secret-key"
	handler := NewAuthHandler(mockRepo, jwtSecret)

	t.Run("Success case", func(t *testing.T) {
		registerReq := RegisterRequest{
			Email:    "new@example.com",
			Password: "newpassword",
			FullName: "New User",
			Gender:   "male",
			Age:      28,
			CityID:   1,
		}

		// Настройка мока - пользователь не существует
		mockRepo.On("GetUserByEmail", registerReq.Email).Return(nil, errors.New("user not found")).Once()

		// При создании пользователя в репозитории будет установлен ID
		mockRepo.On("CreateUser", mock.Anything).Run(func(args mock.Arguments) {
			user := args.Get(0).(*User)
			user.UserID = 1 // Симулируем установку ID после создания
		}).Return(nil).Once()

		body, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Register(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		var response AuthResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.NotEmpty(t, response.Token)
		assert.Equal(t, 1, response.User.UserID)
		assert.Equal(t, registerReq.Email, response.User.Email)
		assert.Equal(t, registerReq.FullName, response.User.FullName)
		assert.Empty(t, response.User.PasswordHash) // Пароль не должен быть в ответе

		mockRepo.AssertExpectations(t)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Register(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		mockRepo.AssertNotCalled(t, "GetUserByEmail")
	})

	t.Run("Email already exists", func(t *testing.T) {
		existingUser := &User{
			UserID:   1,
			Email:    "existing@example.com",
			FullName: "Existing User",
		}

		registerReq := RegisterRequest{
			Email:    existingUser.Email,
			Password: "password",
			FullName: "New Name",
		}

		mockRepo.On("GetUserByEmail", registerReq.Email).Return(existingUser, nil).Once()

		body, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Register(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		mockRepo.AssertNotCalled(t, "CreateUser")
		mockRepo.AssertExpectations(t)
	})

	t.Run("Error creating user", func(t *testing.T) {
		registerReq := RegisterRequest{
			Email:    "new@example.com",
			Password: "newpassword",
			FullName: "New User",
		}

		mockRepo.On("GetUserByEmail", registerReq.Email).Return(nil, errors.New("user not found")).Once()
		mockRepo.On("CreateUser", mock.Anything).Return(errors.New("database error")).Once()

		body, _ := json.Marshal(registerReq)
		req, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		handler.Register(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		mockRepo.AssertExpectations(t)
	})
}

func TestVerify(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSecret := "test-secret-key"
	handler := NewAuthHandler(mockRepo, jwtSecret)

	t.Run("Valid token", func(t *testing.T) {
		// Создаем токен для тестов
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": 1,
			"email":   "test@example.com",
			"exp":     time.Now().Add(time.Hour).Unix(),
		})

		// Подписываем токен
		tokenString, _ := token.SignedString([]byte(jwtSecret))

		// Создаем запрос с токеном в заголовке
		req, _ := http.NewRequest("GET", "/api/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		rr := httptest.NewRecorder()

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/verify", nil)

		rr := httptest.NewRecorder()

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid authorization format", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/verify", nil)
		req.Header.Set("Authorization", "Invalid-Format")

		rr := httptest.NewRecorder()

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")

		rr := httptest.NewRecorder()

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Expired token", func(t *testing.T) {
		// Создаем просроченный токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": 1,
			"email":   "test@example.com",
			"exp":     time.Now().Add(-time.Hour).Unix(), // Токен истек час назад
		})

		tokenString, _ := token.SignedString([]byte(jwtSecret))

		req, _ := http.NewRequest("GET", "/api/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		rr := httptest.NewRecorder()

		handler.Verify(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestAuthMiddleware(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSecret := "test-secret-key"
	handler := NewAuthHandler(mockRepo, jwtSecret)

	t.Run("Valid token", func(t *testing.T) {
		// Создаем токен для тестов
		userId := 1
		email := "test@example.com"
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": userId,
			"email":   email,
			"exp":     time.Now().Add(time.Hour).Unix(),
		})

		tokenString, _ := token.SignedString([]byte(jwtSecret))

		// Создаем тестовый хендлер, который будет вызван при успешной аутентификации
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем, что контекст содержит ожидаемые значения
			ctxUserId := r.Context().Value("user_id").(int)
			ctxEmail := r.Context().Value("email").(string)

			assert.Equal(t, userId, ctxUserId)
			assert.Equal(t, email, ctxEmail)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		// Оборачиваем тестовый хендлер нашим middleware
		middleware := handler.AuthMiddleware(nextHandler)

		// Создаем запрос
		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)

		rr := httptest.NewRecorder()

		// Вызываем middleware
		middleware.ServeHTTP(rr, req)

		// Проверяем, что все прошло успешно и был вызван следующий хендлер
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "success", rr.Body.String())
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Этот хендлер не должен быть вызван
			t.Fatal("Next handler should not be called")
		})

		middleware := handler.AuthMiddleware(nextHandler)

		req, _ := http.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Invalid token", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Этот хендлер не должен быть вызван
			t.Fatal("Next handler should not be called")
		})

		middleware := handler.AuthMiddleware(nextHandler)

		req, _ := http.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rr := httptest.NewRecorder()

		middleware.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestGenerateToken(t *testing.T) {
	mockRepo := new(MockUserRepository)
	jwtSecret := "test-secret-key"
	handler := NewAuthHandler(mockRepo, jwtSecret)

	user := &User{
		UserID: 1,
		Email:  "test@example.com",
	}

	// Вызываем приватный метод через reflection или временно сделав его публичным
	// Для простоты теста мы можем проверить, что токен генерируется и валидируется правильно
	token, err := handler.generateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Проверяем, что токен декодируется корректно
	claims := jwt.MapClaims{}
	_, err = jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})

	assert.NoError(t, err)
	assert.Equal(t, float64(user.UserID), claims["user_id"])
	assert.Equal(t, user.Email, claims["email"])
	assert.NotEmpty(t, claims["exp"])
}
