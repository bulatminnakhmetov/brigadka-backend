package auth

import (
	"errors"
	"fmt"
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
}

type AuthService struct {
	userRepository UserRepository
	jwtSecret      []byte
	tokenExpiry    time.Duration
	refreshExpiry  time.Duration
}

type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}

func NewAuthService(userRepo UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepository: userRepo,
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

	if existingUser != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to process request")
	}

	newUser := &User{
		Email:        email,
		PasswordHash: string(hashedPassword),
	}

	// Save user to DB
	if err := s.userRepository.CreateUser(newUser); err != nil {
		return nil, errors.New("failed to create user")
	}

	// Generate JWT token
	token, err := s.generateToken(newUser)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(newUser)
	if err != nil {
		return nil, errors.New("failed to generate refresh token")
	}

	// Clear sensitive data
	newUser.PasswordHash = ""

	return &AuthResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User:         newUser,
	}, nil
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

func (s *AuthService) VerifyToken(tokenString string) error {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return errors.New("invalid token")
	}

	return nil
}

func (s *AuthService) generateToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(s.tokenExpiry).UnixNano(),
		"type":    "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) generateRefreshToken(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(s.refreshExpiry).UnixNano(),
		"type":    "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) GetUserInfoFromToken(tokenString string) (int, string, error) {
	claims := jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, "", errors.New("invalid token")
	}

	userID := int(claims["user_id"].(float64))
	email := claims["email"].(string)

	return userID, email, nil
}
