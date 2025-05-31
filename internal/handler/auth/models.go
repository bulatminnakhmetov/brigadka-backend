package auth

import (
	serviceAuth "github.com/bulatminnakhmetov/brigadka-backend/internal/service/auth"
)

// Request models
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

type ResendVerificationRequest struct {
	IgnoreCooldown bool `json:"ignore_cooldown,omitempty"`
}

// Response models
type AuthResponse struct {
	UserID        int    `json:"user_id"`
	Token         string `json:"token"`
	RefreshToken  string `json:"refresh_token"`
	EmailVerified bool   `json:"email_verified"`
}

type VerificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VerificationStatusResponse represents the response from the verification status endpoint
type VerificationStatusResponse struct {
	Verified bool `json:"verified"`
}

func ToAuthResponse(serviceResponse *serviceAuth.AuthResponse) AuthResponse {
	return AuthResponse{
		UserID:        serviceResponse.User.ID,
		Token:         serviceResponse.Token,
		RefreshToken:  serviceResponse.RefreshToken,
		EmailVerified: serviceResponse.User.EmailVerified,
	}
}
