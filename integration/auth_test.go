package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/database"
	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite определяет набор интеграционных тестов для аутентификации
type AuthIntegrationTestSuite struct {
	suite.Suite
	db          *sql.DB
	router      *chi.Mux
	appUrl      string
	authHandler *auth.AuthHandler
	testEmail   string
	jwtSecret   string
}

// SetupSuite подготавливает окружение перед запуском всех тестов
func (s *AuthIntegrationTestSuite) SetupSuite() {
	// Используем переменные окружения для настройки тестового окружения
	s.jwtSecret = "test-jwt-secret"
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := 5432
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "brigadka_test"
	}

	// Подключение к тестовой базе данных
	dbConfig := &database.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		DBName:   dbName,
		SSLMode:  "disable",
	}

	s.appUrl = os.Getenv("TEST_APP_URL")

	// Подключаемся к базе данных
	var err error
	s.db, err = database.NewConnection(dbConfig)
	if err != nil {
		s.T().Fatalf("Failed to connect to test database: %v", err)
	}

	// Генерируем уникальный email для тестов
	s.testEmail = fmt.Sprintf("test_user_%d@example.com", os.Getpid())

	// Инициализация репозитория и обработчика авторизации
	userRepo := auth.NewPostgresUserRepository(s.db)
	s.authHandler = auth.NewAuthHandler(userRepo, s.jwtSecret)

	// Настройка маршрутизатора
	s.router = chi.NewRouter()
	s.router.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", s.authHandler.Register)
		r.Post("/login", s.authHandler.Login)
		r.Get("/verify", s.authHandler.Verify)
	})

	// Настройка защищенного маршрута для тестирования middleware
	s.router.Group(func(r chi.Router) {
		r.Use(s.authHandler.AuthMiddleware)
		r.Get("/api/protected", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value("user_id").(int)
			email := r.Context().Value("email").(string)
			response := map[string]interface{}{
				"user_id": userID,
				"email":   email,
				"message": "protected resource accessed",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		})
	})
}

// TestRegisterAndLogin тестирует полный цикл регистрации и входа в систему
func (s *AuthIntegrationTestSuite) TestRegisterAndLogin() {
	t := s.T()

	// Данные для регистрации
	registerData := auth.RegisterRequest{
		Email:    s.testEmail,
		Password: "TestPassword123",
		FullName: "Test User",
		Gender:   "male",
		Age:      30,
		CityID:   1,
	}

	// Шаг 1: Регистрация пользователя
	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	assert.NoError(t, err)
	defer registerResp.Body.Close()

	assert.Equal(t, http.StatusCreated, registerResp.StatusCode)

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	assert.NoError(t, err)

	// Проверка данных пользователя в ответе
	assert.NotEmpty(t, registerResult.Token)
	assert.Equal(t, s.testEmail, registerResult.User.Email)
	assert.Equal(t, "Test User", registerResult.User.FullName)
	assert.Equal(t, "male", registerResult.User.Gender)
	assert.Equal(t, 30, registerResult.User.Age)
	assert.Empty(t, registerResult.User.PasswordHash) // Хеш пароля не должен передаваться клиенту

	// Шаг 2: Вход с зарегистрированными данными
	loginData := auth.LoginRequest{
		Email:    s.testEmail,
		Password: "TestPassword123",
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	var loginResult auth.AuthResponse
	println("Login response:", loginResp.Body)
	err = json.NewDecoder(loginResp.Body).Decode(&loginResult)
	assert.NoError(t, err)

	// Проверка данных пользователя в ответе
	assert.NotEmpty(t, loginResult.Token)
	assert.Equal(t, s.testEmail, loginResult.User.Email)
	assert.Equal(t, "Test User", loginResult.User.FullName)
	assert.Equal(t, "male", loginResult.User.Gender)
	assert.Equal(t, 30, loginResult.User.Age)
	assert.Empty(t, loginResult.User.PasswordHash)

	// Шаг 3: Проверка защищенного ресурса с полученным токеном
	protectedReq, _ := http.NewRequest("GET", s.appUrl+"/api/protected", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+loginResult.Token)

	protectedResp, err := client.Do(protectedReq)
	assert.NoError(t, err)
	defer protectedResp.Body.Close()

	assert.Equal(t, http.StatusOK, protectedResp.StatusCode)

	var protectedResult map[string]interface{}
	err = json.NewDecoder(protectedResp.Body).Decode(&protectedResult)
	assert.NoError(t, err)
	assert.Equal(t, "protected resource accessed", protectedResult["message"])
	assert.Equal(t, s.testEmail, protectedResult["email"])
}

// TestRegisterWithExistingEmail проверяет обработку ошибки при регистрации с существующим email
func (s *AuthIntegrationTestSuite) TestRegisterWithExistingEmail() {
	t := s.T()

	// Используем тот же email, который был зарегистрирован в предыдущем тесте
	registerData := auth.RegisterRequest{
		Email:    s.testEmail,
		Password: "AnotherPassword123",
		FullName: "Another Test User",
		Gender:   "female",
		Age:      25,
		CityID:   1,
	}

	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	assert.NoError(t, err)
	defer registerResp.Body.Close()

	// Должен быть конфликт, так как email уже существует
	assert.Equal(t, http.StatusConflict, registerResp.StatusCode)
}

// TestLoginWithInvalidCredentials проверяет обработку неверных учетных данных
func (s *AuthIntegrationTestSuite) TestLoginWithInvalidCredentials() {
	t := s.T()

	// Правильный email, неправильный пароль
	loginData := auth.LoginRequest{
		Email:    s.testEmail,
		Password: "WrongPassword",
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	// Должен быть статус Unauthorized
	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)

	// Несуществующий email
	loginData = auth.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "AnyPassword",
	}

	loginJSON, _ = json.Marshal(loginData)
	loginReq, _ = http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err = client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	// Должен быть статус Unauthorized
	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode)
}

// TestProtectedResourceWithoutAuth проверяет доступ к защищенному ресурсу без токена
func (s *AuthIntegrationTestSuite) TestProtectedResourceWithoutAuth() {
	t := s.T()

	// Запрос к защищенному ресурсу без токена
	protectedReq, _ := http.NewRequest("GET", s.appUrl+"/api/protected", nil)

	client := &http.Client{}
	protectedResp, err := client.Do(protectedReq)
	assert.NoError(t, err)
	defer protectedResp.Body.Close()

	// Должен быть статус Unauthorized
	assert.Equal(t, http.StatusUnauthorized, protectedResp.StatusCode)
}

// TestVerifyToken проверяет работу эндпоинта верификации токена
func (s *AuthIntegrationTestSuite) TestVerifyToken() {
	t := s.T()

	// Сначала логинимся, чтобы получить валидный токен
	loginData := auth.LoginRequest{
		Email:    s.testEmail,
		Password: "TestPassword123",
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	var loginResult auth.AuthResponse
	err = json.NewDecoder(loginResp.Body).Decode(&loginResult)
	assert.NoError(t, err)

	// Проверяем верификацию токена
	verifyReq, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+loginResult.Token)

	verifyResp, err := client.Do(verifyReq)
	assert.NoError(t, err)
	defer verifyResp.Body.Close()

	// Должен быть статус OK
	assert.Equal(t, http.StatusOK, verifyResp.StatusCode)

	// Проверяем верификацию с невалидным токеном
	verifyReq, _ = http.NewRequest("GET", s.appUrl+"/api/auth/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer invalid-token")

	verifyResp, err = client.Do(verifyReq)
	assert.NoError(t, err)
	defer verifyResp.Body.Close()

	// Должен быть статус Unauthorized
	assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode)
}

// TestAuthIntegration запускает набор интеграционных тестов
func TestAuthIntegration(t *testing.T) {
	// Пропускаем тесты, если задана переменная окружения SKIP_INTEGRATION_TESTS
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(AuthIntegrationTestSuite))
}
