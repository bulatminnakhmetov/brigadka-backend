package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite определяет набор интеграционных тестов для аутентификации
type AuthIntegrationTestSuite struct {
	suite.Suite
	appUrl string
}

// SetupSuite подготавливает окружение перед запуском всех тестов
func (s *AuthIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Значение по умолчанию для локального тестирования
	}
}

// Вспомогательная функция для генерации уникального email
func generateUniqueEmail() string {
	return fmt.Sprintf("test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// Вспомогательная функция для регистрации тестового пользователя
func (s *AuthIntegrationTestSuite) registerTestUser(email, password, fullName, gender string, age, cityID int) (*auth.AuthResponse, error) {
	registerData := auth.RegisterRequest{
		Email:    email,
		Password: password,
		FullName: fullName,
		Gender:   gender,
		Age:      age,
		CityID:   cityID,
	}

	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	if err != nil {
		return nil, err
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registerResp.Body)
		return nil, fmt.Errorf("failed to register user. Status code: %d, Body: %s", registerResp.StatusCode, string(body))
	}

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	if err != nil {
		return nil, err
	}

	return &registerResult, nil
}

// TestRegisterAndLogin тестирует полный цикл регистрации и входа в систему
func (s *AuthIntegrationTestSuite) TestRegisterAndLogin() {
	t := s.T()

	// Генерируем уникальный email для этого теста
	email := generateUniqueEmail()
	password := "TestPassword123"

	// Шаг 1: Регистрация пользователя
	registerData := auth.RegisterRequest{
		Email:    email,
		Password: password,
		FullName: "Test User",
		Gender:   "male",
		Age:      30,
		CityID:   1,
	}

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
	assert.NotEmpty(t, registerResult.RefreshToken) // Проверяем наличие refresh токена
	assert.Equal(t, email, registerResult.User.Email)
	assert.Equal(t, "Test User", registerResult.User.FullName)
	assert.Equal(t, "male", registerResult.User.Gender)
	assert.Equal(t, 30, registerResult.User.Age)
	assert.Empty(t, registerResult.User.PasswordHash) // Хеш пароля не должен передаваться клиенту

	// Шаг 2: Вход с зарегистрированными данными
	loginData := auth.LoginRequest{
		Email:    email,
		Password: password,
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	assert.Equal(t, http.StatusOK, loginResp.StatusCode)

	// Читаем тело ответа для логирования
	bodyBytes, err := io.ReadAll(loginResp.Body)
	assert.NoError(t, err)

	// Декодируем тело ответа
	var loginResult auth.AuthResponse
	err = json.Unmarshal(bodyBytes, &loginResult)
	assert.NoError(t, err)

	// Проверка данных пользователя в ответе
	assert.NotEmpty(t, loginResult.Token)
	assert.NotEmpty(t, loginResult.RefreshToken) // Проверяем наличие refresh токена
	assert.Equal(t, email, loginResult.User.Email)
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

	protectedBodyBytes, err := io.ReadAll(protectedResp.Body)
	assert.NoError(t, err)
	protectedResult := string(protectedBodyBytes)

	assert.Equal(t, fmt.Sprintf("Protected resource. User ID: %d, Email: %s", loginResult.User.ID, loginResult.User.Email), protectedResult)
}

// TestRefreshToken тестирует обновление токена с использованием refresh токена
func (s *AuthIntegrationTestSuite) TestRefreshToken() {
	t := s.T()

	// Генерируем уникальный email для этого теста
	email := generateUniqueEmail()
	password := "TestPassword123"

	// Шаг 1: Регистрация пользователя для получения токенов
	authResponse, err := s.registerTestUser(email, password, "Refresh Test User", "male", 30, 1)
	assert.NoError(t, err, "Failed to register user")
	assert.NotEmpty(t, authResponse.RefreshToken, "Refresh token should not be empty")

	// Шаг 2: Используем refresh токен для получения новых токенов
	refreshData := auth.RefreshRequest{
		RefreshToken: authResponse.RefreshToken,
	}

	refreshJSON, _ := json.Marshal(refreshData)
	refreshReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/refresh", bytes.NewBuffer(refreshJSON))
	refreshReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	refreshResp, err := client.Do(refreshReq)
	assert.NoError(t, err)
	defer refreshResp.Body.Close()

	assert.Equal(t, http.StatusOK, refreshResp.StatusCode)

	var refreshResult auth.AuthResponse
	err = json.NewDecoder(refreshResp.Body).Decode(&refreshResult)
	assert.NoError(t, err)

	// Проверка результата обновления токена
	assert.NotEmpty(t, refreshResult.Token)
	assert.NotEmpty(t, refreshResult.RefreshToken)
	assert.NotEqual(t, authResponse.Token, refreshResult.Token, "New access token should be different")
	assert.NotEqual(t, authResponse.RefreshToken, refreshResult.RefreshToken, "New refresh token should be different")

	// Данные пользователя должны остаться теми же
	assert.Equal(t, authResponse.User.ID, refreshResult.User.ID)
	assert.Equal(t, email, refreshResult.User.Email)
	assert.Equal(t, "Refresh Test User", refreshResult.User.FullName)

	// Шаг 3: Проверяем, что новый токен действительно работает
	protectedReq, _ := http.NewRequest("GET", s.appUrl+"/api/protected", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+refreshResult.Token)

	protectedResp, err := client.Do(protectedReq)
	assert.NoError(t, err)
	defer protectedResp.Body.Close()

	assert.Equal(t, http.StatusOK, protectedResp.StatusCode)
}

// TestInvalidRefreshToken тестирует попытку обновления токена с невалидным refresh токеном
func (s *AuthIntegrationTestSuite) TestInvalidRefreshToken() {
	t := s.T()

	// Тестируем с невалидным refresh токеном
	refreshData := auth.RefreshRequest{
		RefreshToken: "invalid-refresh-token",
	}

	refreshJSON, _ := json.Marshal(refreshData)
	refreshReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/refresh", bytes.NewBuffer(refreshJSON))
	refreshReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	refreshResp, err := client.Do(refreshReq)
	assert.NoError(t, err)
	defer refreshResp.Body.Close()

	// Должен вернуть статус Unauthorized
	assert.Equal(t, http.StatusUnauthorized, refreshResp.StatusCode)
}

// TestRegisterWithExistingEmail проверяет обработку ошибки при регистрации с существующим email
func (s *AuthIntegrationTestSuite) TestRegisterWithExistingEmail() {
	t := s.T()

	// Генерируем уникальный email для этого теста
	email := generateUniqueEmail()

	// Сначала регистрируем пользователя
	_, err := s.registerTestUser(email, "FirstPassword123", "Original User", "male", 30, 1)
	assert.NoError(t, err, "Failed to register initial user")

	// Пытаемся зарегистрировать еще одного пользователя с тем же email
	registerData := auth.RegisterRequest{
		Email:    email, // Используем тот же email
		Password: "SecondPassword456",
		FullName: "Another User",
		Gender:   "female",
		Age:      25,
		CityID:   2,
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

	// Генерируем уникальный email для этого теста
	email := generateUniqueEmail()
	password := "CorrectPassword123"

	// Сначала регистрируем пользователя
	_, err := s.registerTestUser(email, password, "Test User", "male", 30, 1)
	assert.NoError(t, err, "Failed to register user")

	// Проверяем вход с неправильным паролем
	loginData := auth.LoginRequest{
		Email:    email,
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

	// Проверяем вход с несуществующим email
	loginData = auth.LoginRequest{
		Email:    "nonexistent_" + generateUniqueEmail(),
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

	// Генерируем уникальный email для этого теста
	email := generateUniqueEmail()
	password := "TestPassword123"

	// Сначала регистрируем пользователя
	auth, err := s.registerTestUser(email, password, "Test User", "male", 30, 1)
	assert.NoError(t, err, "Failed to register user")

	// Получаем токен из успешной регистрации
	token := auth.Token

	// Проверяем верификацию токена
	verifyReq, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
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
