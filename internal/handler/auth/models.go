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

type AuthResponse struct {
	UserID       int    `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

func ToAuthResponse(serviceResponse *serviceAuth.AuthResponse) AuthResponse {
	return AuthResponse{
		UserID:       serviceResponse.User.ID,
		Token:        serviceResponse.Token,
		RefreshToken: serviceResponse.RefreshToken,
	}
}
