package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func expectEmailCheck(mock sqlmock.Sqlmock, email string, exists bool) {
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM users WHERE email = \\$1\\)").
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}

func expectCityCheck(mock sqlmock.Sqlmock, cityID int, exists bool) {
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM cities WHERE city_id = \\$1\\)").
		WithArgs(cityID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}

func expectGenderCheck(mock sqlmock.Sqlmock, gender string, exists bool) {
	mock.ExpectQuery("SELECT EXISTS\\(SELECT 1 FROM gender_catalog WHERE gender_code = \\$1\\)").
		WithArgs(gender).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(exists))
}

func expectUserInsert(mock sqlmock.Sqlmock, email, fullName string, cityID int, gender string, age int) {
	mock.ExpectExec("INSERT INTO users").
		WithArgs(email, sqlmock.AnyArg(), fullName, cityID, gender, age).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func setupTestRouter(t *testing.T) (*gin.Engine, *AuthController, sqlmock.Sqlmock) {
	// Создаем мок базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	// Создаем контроллер с моком базы данных
	controller := &AuthController{
		DB:     db,
		JWTKey: []byte("test_key"),
	}

	// Настраиваем Gin для тестов
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.POST("/register", controller.Register)
	router.POST("/login", controller.Login)

	return router, controller, mock
}

func TestRegister_Success(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	expectEmailCheck(mock, "test@example.com", false)
	expectCityCheck(mock, 1, true)
	expectGenderCheck(mock, "male", true)
	expectUserInsert(mock, "test@example.com", "Test User", 1, "male", 25)

	body := `{
        "email": "test@example.com",
        "password": "password123",
        "full_name": "Test User",
        "city_id": 1,
        "gender": "male",
        "age": 25
    }`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.JSONEq(t, `{"message": "User registered"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_DuplicateEmail(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	expectEmailCheck(mock, "test@example.com", true)

	body := `{
        "email": "test@example.com",
        "password": "password123",
        "full_name": "Test User",
        "city_id": 1,
        "gender": "male",
        "age": 25
    }`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Email already registered"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_InvalidInput(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "Missing required fields",
			body: `{"email": "test@example.com", "password": "password123"}`,
		},
		{
			name: "Invalid email",
			body: `{
		        "email": "invalid-email",
		        "password": "password123",
		        "full_name": "Test User",
		        "city_id": 1,
		        "gender": "male",
		        "age": 25
		    }`,
		},
		{
			name: "Negative age",
			body: `{
                "email": "test@example.com",
                "password": "password123",
                "full_name": "Test User",
                "city_id": 1,
                "gender": "male",
                "age": -1
            }`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestRegister_InvalidCity(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	expectEmailCheck(mock, "test@example.com", false)
	expectCityCheck(mock, 999, false)

	body := `{
        "email": "test@example.com",
        "password": "password123",
        "full_name": "Test User",
        "city_id": 999,
        "gender": "male",
        "age": 25
    }`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Invalid city ID"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRegister_InvalidGender(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	expectEmailCheck(mock, "test@example.com", false)
	expectCityCheck(mock, 1, true)
	expectGenderCheck(mock, "invalid_gender", false)

	body := `{
        "email": "test@example.com",
        "password": "password123",
        "full_name": "Test User",
        "city_id": 1,
        "gender": "invalid_gender",
        "age": 25
    }`
	req, _ := http.NewRequest("POST", "/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Invalid gender code"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_Success(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	// Генерируем реальный хеш пароля
	password := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Настраиваем мок для успешного входа
	rows := sqlmock.NewRows([]string{"user_id", "password_hash"}).
		AddRow(1, string(hashedPassword))
	mock.ExpectQuery("SELECT user_id, password_hash FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	body := `{"email": "test@example.com", "password": "password123"}`
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"token":`)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_UserNotFound(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	// Настраиваем мок для случая, когда пользователь не найден
	mock.ExpectQuery("SELECT user_id, password_hash FROM users WHERE email = \\$1").
		WithArgs("nonexistent@example.com").
		WillReturnError(sql.ErrNoRows)

	body := `{"email": "nonexistent@example.com", "password": "password123"}`
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.JSONEq(t, `{"error": "User not found"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_InvalidPassword(t *testing.T) {
	router, _, mock := setupTestRouter(t)

	// Настраиваем мок для случая с неверным паролем
	rows := sqlmock.NewRows([]string{"user_id", "password_hash"}).
		AddRow(1, "$2a$10$hashedpassword")
	mock.ExpectQuery("SELECT user_id, password_hash FROM users WHERE email = \\$1").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	body := `{"email": "test@example.com", "password": "wrongpassword"}`
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.JSONEq(t, `{"error": "Invalid email or password"}`, w.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_InvalidInput(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Тест с невалидным email
	body := `{"email": "invalid-email", "password": "password123"}`
	req, _ := http.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Invalid input"}`, w.Body.String())

	// Тест с отсутствующим паролем
	body = `{"email": "test@example.com"}`
	req, _ = http.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "Invalid input"}`, w.Body.String())
}
