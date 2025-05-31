package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	authService "github.com/bulatminnakhmetov/brigadka-backend/internal/service/auth"
)

type AuthHandler struct {
	authService *authService.AuthService
}

func NewAuthHandler(authService *authService.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// @Summary      User login
// @Description  Authenticate user by email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  LoginRequest  true  "Login data"
// @Success      200      {object}  AuthResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Invalid credentials"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	serviceResponse, err := h.authService.Login(req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert service response to API response
	response := ToAuthResponse(serviceResponse)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      User registration
// @Description  Create a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  RegisterRequest  true  "Registration data"
// @Success      201      {object}  AuthResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      409      {string}  string  "Email already registered"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	serviceResponse, err := h.authService.Register(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email already registered" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert service response to API response
	response := ToAuthResponse(serviceResponse)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary      Token refresh
// @Description  Get a new token using a refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  RefreshRequest  true  "Token refresh data"
// @Success      200      {object}  AuthResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Invalid refresh token"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	serviceResponse, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Convert service response to API response
	response := ToAuthResponse(serviceResponse)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware for authentication
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		userID, email, err := h.authService.GetUserInfoFromToken(tokenString)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Add user data to request context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user_id", userID)
		ctx = context.WithValue(ctx, "email", email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function to extract token from request
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Format should be: "Bearer {token}"
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}

	return authHeader[7:]
}
