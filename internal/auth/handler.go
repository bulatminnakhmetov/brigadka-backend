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
}

type UserRepository interface {
	GetUserByEmail(email string) (*User, error)
	CreateUser(user *User) error
}

type User struct {
	UserID       int    `json:"user_id"`
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

type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

func NewAuthHandler(userRepo UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userRepository: userRepo,
		jwtSecret:      []byte(jwtSecret),
		tokenExpiry:    time.Hour * 24, // Токен действителен 24 часа
	}
}

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

	// Очистка чувствительных данных
	user.PasswordHash = ""

	resp := AuthResponse{
		Token: token,
		User:  user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

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

	// Очистка чувствительных данных
	newUser.PasswordHash = ""

	resp := AuthResponse{
		Token: token,
		User:  newUser,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

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
		"user_id": user.UserID,
		"email":   user.Email,
		"exp":     time.Now().Add(h.tokenExpiry).Unix(),
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
