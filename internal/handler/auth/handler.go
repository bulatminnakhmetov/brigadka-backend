package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	authservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/verification"
)

type AuthHandler struct {
	authService *authservice.AuthService
}

func NewAuthHandler(authService *authservice.AuthService) *AuthHandler {
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
		if strings.Contains(err.Error(), "email already registered but not verified") {
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

// @Summary      Email verification
// @Description  Verify a user's email address using a verification token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  VerifyEmailRequest  true  "Verification token"
// @Success      200      {object}  VerificationResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Invalid verification token"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.authService.VerifyEmail(req.Token)
	if err != nil {
		if strings.Contains(err.Error(), "invalid verification token") ||
			strings.Contains(err.Error(), "verification token has expired") {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := VerificationResponse{
		Success: true,
		Message: "Email verified successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Resend verification email
// @Description  Resend verification email to a user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body  ResendVerificationRequest  true  "Email for verification"
// @Success      200      {object}  VerificationResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      404      {string}  string  "User not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /auth/resend-verification [post]
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from email
	user, err := h.authService.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Check if already verified
	if user.EmailVerified {
		response := VerificationResponse{
			Success: true,
			Message: "Email is already verified",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Resend verification
	err = h.authService.ResendVerificationEmail(user.ID, req.IgnoreCooldown)
	if err != nil {
		if errors.Is(err, verification.ErrEmailRecentlySent) {
			http.Error(w, "Verification email was sent recently", http.StatusTooManyRequests)
			return
		}
		if errors.Is(err, verification.ErrEmailAlreadyVerified) {
			http.Error(w, "Email is already verified", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := VerificationResponse{
		Success: true,
		Message: "Verification email sent",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Get user verification status
// @Description  Check if user's email is verified
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  VerificationStatusResponse
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /auth/verification-status [get]
func (h *AuthHandler) GetVerificationStatus(w http.ResponseWriter, r *http.Request) {
	// Get userID from context set by the modified AuthMiddleware
	userID := r.Context().Value("user_id").(int)

	// Get verification status from auth service
	isVerified, err := h.authService.IsUserVerified(userID)
	if err != nil {
		http.Error(w, "Error checking verification status", http.StatusInternalServerError)
		return
	}

	response := VerificationStatusResponse{
		Verified: isVerified,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// Middleware for authentication
func (h *AuthHandler) AuthMiddleware(requireEmailVerification bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			user, err := h.authService.GetUserInfoFromToken(tokenString)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if requireEmailVerification && !user.EmailVerified {
				http.Error(w, "Email not verified", http.StatusForbidden)
				return
			}

			// Add user data to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "user_id", user.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
