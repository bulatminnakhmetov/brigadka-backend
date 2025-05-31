package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	userrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"
)

type User = userrepo.User

type UserRepository interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int) (*User, error)
	CreateUser(user *User) error
	UpdateEmailVerificationStatus(userID int, verified bool) error
	UpdateUser(user *User) error
}

type EmailVerificationService interface {
	SendVerificationEmail(userID int, userEmail string) error
	VerifyEmail(token string) error
	ResendVerificationEmail(userID int, ignoreCooldown bool) error
	IsTokenExpiredForEmail(email string) (bool, error)
}

type AuthService struct {
	userRepository UserRepository
	emailService   EmailVerificationService
	jwtSecret      []byte
	tokenExpiry    time.Duration
	refreshExpiry  time.Duration
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

func NewAuthService(userRepo UserRepository, emailService EmailVerificationService, jwtSecret string) *AuthService {
	return &AuthService{
		userRepository: userRepo,
		emailService:   emailService,
		jwtSecret:      []byte(jwtSecret),
		tokenExpiry:    time.Hour * 1,      // Token valid for 1 hour
		refreshExpiry:  time.Hour * 24 * 7, // Refresh token valid for 7 days
	}
}

func (s *AuthService) Login(email, password string) (*AuthResponse, error) {
	user, err := s.userRepository.GetUserByEmail(email)

	if err != nil && err != userrepo.ErrUserNotFound {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	// Clear sensitive data
	userCopy := *user
	userCopy.PasswordHash = ""

	return &AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         &userCopy,
	}, nil
}

func (s *AuthService) Register(email, password string) (*AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepository.GetUserByEmail(email)

	if err != nil && err != userrepo.ErrUserNotFound {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to process request")
	}

	var user *User

	// Special handling for unverified emails
	if existingUser != nil {
		// If the user exists and is already verified, registration is rejected
		if existingUser.EmailVerified {
			return nil, errors.New("email already registered")
		}

		// If token is expired, update the existing user instead of creating a new one
		existingUser.PasswordHash = string(hashedPassword)
		existingUser.EmailVerified = false

		if err := s.userRepository.UpdateUser(existingUser); err != nil {
			return nil, errors.New("failed to update user")
		}

		user = existingUser
	} else {
		// Create new user if no existing user was found
		newUser := &User{
			Email:         email,
			PasswordHash:  string(hashedPassword),
			EmailVerified: false, // New users start unverified
		}

		// Save user to DB
		if err := s.userRepository.CreateUser(newUser); err != nil {
			return nil, errors.New("failed to create user")
		}

		user = newUser
	}

	// Send verification email
	if err := s.emailService.SendVerificationEmail(user.ID, user.Email); err != nil {
		fmt.Printf("Failed to send verification email: %v\n", err)
	}

	// Generate JWT token
	token, err := s.generateToken(user)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) VerifyEmail(token string) error {
	return s.emailService.VerifyEmail(token)
}

func (s *AuthService) RefreshToken(refreshToken string) (*AuthResponse, error) {
	// Parse refresh token
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	// Verify that it's a refresh token
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Get user from database
	userID := int(claims["user_id"].(float64))
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate new tokens
	newToken, err := s.generateToken(user)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	// Clear sensitive data
	user.PasswordHash = ""

	return &AuthResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		User:         user,
	}, nil
}

func (s *AuthService) generateToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":        user.ID,
		"email":          user.Email,
		"email_verified": user.EmailVerified, // Include verification status in token
		"exp":            time.Now().Add(s.tokenExpiry).UnixNano(),
		"type":           "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) generateRefreshToken(user *User) (string, error) {
	var expireAt time.Time
	if user.EmailVerified {
		expireAt = time.Now().Add(s.refreshExpiry)
	} else {
		expireAt = time.Now().Add(s.tokenExpiry) // Use shorter expiry for unverified users
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     expireAt.UnixNano(),
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// GetUserInfoFromToken extracts user information from JWT token
func (s *AuthService) GetUserInfoFromToken(tokenString string) (*User, error) {
	// Extract JWT token
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	userID := int(claims["user_id"].(float64))
	email := claims["email"].(string)
	emailVerified := claims["email_verified"].(bool)

	return &userrepo.User{ID: userID, Email: email, EmailVerified: emailVerified}, nil
}

// IsUserVerified checks if a user's email is verified
func (s *AuthService) IsUserVerified(userID int) (bool, error) {
	user, err := s.userRepository.GetUserByID(userID)
	if err != nil {
		return false, err
	}
	return user.EmailVerified, nil
}

// ResendVerificationEmail sends a new verification email
func (s *AuthService) ResendVerificationEmail(userID int, ignoreCooldown bool) error {
	return s.emailService.ResendVerificationEmail(userID, ignoreCooldown)
}

// GetUserByEmail returns user information by email
func (s *AuthService) GetUserByEmail(email string) (*User, error) {
	return s.userRepository.GetUserByEmail(email)
}
