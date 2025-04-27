package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepository UserRepository
	jwtSecret      []byte
	tokenExpiry    time.Duration
	refreshExpiry  time.Duration // Added for refresh token
}

type UserRepository interface {
	GetUserByEmail(email string) (*User, error)
	CreateUser(user *User) error
	GetUserByID(id int) (*User, error)
}

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	PasswordHash string `json:"-"`
	Gender       string `json:"gender,omitempty"`
	Age          int    `json:"age,omitempty"`
	CityID       int    `json:"city_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty"`
	CityID   int    `json:"city_id,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

func NewAuthHandler(userRepo UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userRepository: userRepo,
		jwtSecret:      []byte(jwtSecret),
		tokenExpiry:    time.Hour * 1,      // Токен действителен 1 час
		refreshExpiry:  time.Hour * 24 * 7, // Refresh токен действителен 7 дней
	}
}

// @Summary      Вход пользователя
// @Description  Аутентификация пользователя по email и паролю
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  LoginRequest  true  "Данные для входа"
// @Success      200      {object}  AuthResponse
// @Failure      400      {string}  string  "Невалидные данные"
// @Failure      401      {string}  string  "Неверные учетные данные"
// @Failure      500      {string}  string  "Внутренняя ошибка сервера"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userRepository.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Проверка пароля
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Генерация JWT токена
	token, err := h.generateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Генерация refresh токена
	refreshToken, err := h.generateRefreshToken(user)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Очистка чувствительных данных
	user.PasswordHash = ""

	resp := AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// @Summary      Регистрация пользователя
// @Description  Создание нового пользователя
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  RegisterRequest  true  "Данные для регистрации"
// @Success      201      {object}  AuthResponse
// @Failure      400      {string}  string  "Невалидные данные"
// @Failure      409      {string}  string  "Email уже зарегистрирован"
// @Failure      500      {string}  string  "Внутренняя ошибка сервера"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверка, что пользователь с таким email не существует
	existingUser, _ := h.userRepository.GetUserByEmail(req.Email)
	if existingUser != nil {
		http.Error(w, "Email already registered", http.StatusConflict)
		return
	}

	// Хэширование пароля
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	newUser := &User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		Gender:       req.Gender,
		Age:          req.Age,
		CityID:       req.CityID,
	}

	// Сохранение пользователя в БД
	if err := h.userRepository.CreateUser(newUser); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Генерация JWT токена
	token, err := h.generateToken(newUser)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Генерация refresh токена
	refreshToken, err := h.generateRefreshToken(newUser)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Очистка чувствительных данных
	newUser.PasswordHash = ""

	resp := AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         newUser,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// @Summary      Обновление токена
// @Description  Получение нового токена с помощью refresh токена
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  RefreshRequest  true  "Данные для обновления токена"
// @Success      200      {object}  AuthResponse
// @Failure      400      {string}  string  "Невалидные данные"
// @Failure      401      {string}  string  "Невалидный refresh токен"
// @Failure      500      {string}  string  "Внутренняя ошибка сервера"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Парсинг refresh токена
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Проверка, что это именно refresh токен
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		http.Error(w, "Invalid token type", http.StatusUnauthorized)
		return
	}

	// Получение пользователя из базы данных
	userID := int(claims["user_id"].(float64))
	user, err := h.userRepository.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Генерация новых токенов
	newToken, err := h.generateToken(user)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	newRefreshToken, err := h.generateRefreshToken(user)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	// Очистка чувствительных данных
	user.PasswordHash = ""

	resp := AuthResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// @Summary      Проверка токена
// @Description  Проверка валидности JWT токена
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200      {string}  string  "Токен валиден"
// @Failure      401      {string}  string  "Невалидный токен"
// @Router       /api/auth/verify [get]
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Формат заголовка должен быть: "Bearer {token}"
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
		return
	}

	tokenString := authHeader[7:]
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Проверяем, что алгоритм подписи соответствует ожидаемому
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Токен валиден
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"valid"}`))
}

func (h *AuthHandler) generateToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(h.tokenExpiry).UnixNano(),
		"type":    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// Функция для генерации refresh токена
func (h *AuthHandler) generateRefreshToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(h.refreshExpiry).UnixNano(),
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.jwtSecret)
}

// Middleware для аутентификации
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Формат заголовка должен быть: "Bearer {token}"
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader[7:]
		claims := jwt.MapClaims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return h.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Добавляем данные пользователя в контекст запроса
		userID := int(claims["user_id"].(float64))
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", userID)
		ctx = context.WithValue(ctx, "email", claims["email"].(string))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
