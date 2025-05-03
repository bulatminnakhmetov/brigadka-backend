package auth

import (
	"time"

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
	FullName string `json:"full_name"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty"`
	CityID   int    `json:"city_id,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Response models
type UserDTO struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty"`
	CityID   int    `json:"city_id,omitempty"`
	// Add any additional user fields for API responses
}

type AuthResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserDTO   `json:"user"`
}

// Conversion functions
func ToUserDTO(serviceUser *serviceAuth.User) UserDTO {
	return UserDTO{
		ID:       serviceUser.ID,
		Email:    serviceUser.Email,
		FullName: serviceUser.FullName,
		Gender:   serviceUser.Gender,
		Age:      serviceUser.Age,
		CityID:   serviceUser.CityID,
	}
}

func ToAuthResponse(serviceResponse *serviceAuth.AuthResponse) AuthResponse {
	// Calculate token expiry time (1 hour from now, matching service configuration)
	expiresAt := time.Now().Add(time.Hour)

	return AuthResponse{
		Token:        serviceResponse.Token,
		RefreshToken: serviceResponse.RefreshToken,
		ExpiresAt:    expiresAt,
		User:         ToUserDTO(serviceResponse.User),
	}
}
