package verification

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/config"
	userrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"
	verificationRepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/verification"
)

const (
	// TokenLength is the length of the verification token
	TokenLength = 32

	// TokenExpirationHours is how long a token is valid for
	TokenExpirationHours = 24

	// TestToken is a predefined token used in test environments
	TestToken = "test-verification-token-%d"
)

// Error constants
var (
	ErrGenerateToken             = errors.New("failed to generate token")
	ErrSaveToken                 = errors.New("failed to save token")
	ErrGenerateVerificationToken = errors.New("failed to generate verification token")
	ErrInvalidToken              = errors.New("invalid verification token")
	ErrTokenExpired              = errors.New("verification token has expired")
	ErrVerifyToken               = errors.New("failed to verify token")
	ErrUpdateVerificationStatus  = errors.New("failed to update user verification status")
	ErrGetUser                   = errors.New("failed to get user")
	ErrEmailAlreadyVerified      = errors.New("email already verified")
	ErrEmailRecentlySent         = errors.New("verification email already sent recently, please wait before resending")
)

type EmailProviderClient interface {
	// SendVerificationEmail sends an email with the verification link
	SendVerificationEmail(to string, subject string, body string) error
}

// EmailVerificationService handles email verification
type EmailVerificationService struct {
	emailProviderClient EmailProviderClient
	userRepo            UserRepository
	verificationRepo    VerificationRepository
	frontendURL         string
	environment         string
}

// UserRepository defines methods needed from the user repository
type UserRepository interface {
	GetUserByEmail(email string) (*userrepo.User, error)
	GetUserByID(id int) (*userrepo.User, error)
	UpdateEmailVerificationStatus(userID int, verified bool) error
}

// VerificationRepository defines methods needed from the verification repository
type VerificationRepository interface {
	CreateToken(token *verificationRepo.VerificationToken) error
	GetTokenByValue(tokenValue string, tokenType verificationRepo.TokenType) (*verificationRepo.VerificationToken, error)
	GetTokenByUserID(userID int, tokenType verificationRepo.TokenType) (*verificationRepo.VerificationToken, error)
	DeleteToken(tokenID int) error
	DeleteExpiredTokens() error
}

// NewEmailVerificationService creates a new email verification service
func NewEmailVerificationService(
	userRepo UserRepository,
	emailProviderClient EmailProviderClient,
	verificationRepo VerificationRepository,
	frontendURL string,
	env string,
) *EmailVerificationService {
	return &EmailVerificationService{
		userRepo:            userRepo,
		emailProviderClient: emailProviderClient,
		verificationRepo:    verificationRepo,
		frontendURL:         frontendURL,
		environment:         env,
	}
}

// GenerateVerificationToken creates a new token for email verification
func (s *EmailVerificationService) generateVerificationToken(userID int) (string, error) {
	// If in test environment, use predefined token
	if s.environment == config.EnvTypeTest {
		return fmt.Sprintf(TestToken, userID), nil
	}

	// Generate random token
	tokenBytes := make([]byte, TokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("%w: %v", ErrGenerateToken, err)
	}

	token := hex.EncodeToString(tokenBytes)

	return token, nil
}

// SendVerificationEmail sends an email with a verification link
func (s *EmailVerificationService) SendVerificationEmail(userID int, userEmail string) error {
	token, err := s.generateVerificationToken(userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGenerateVerificationToken, err)
	}

	// Create and save token to database
	verificationToken := &verificationRepo.VerificationToken{
		UserID:    userID,
		Token:     token,
		Type:      verificationRepo.EmailVerification,
		ExpiresAt: time.Now().Add(TokenExpirationHours * time.Hour),
	}

	if err := s.verificationRepo.CreateToken(verificationToken); err != nil {
		return fmt.Errorf("%w: %v", ErrSaveToken, err)
	}

	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", s.frontendURL, token)

	s.emailProviderClient.SendVerificationEmail(userEmail, "Brigadka: Email Verification", verificationLink)

	return nil
}

// VerifyEmail validates the token and marks the user's email as verified
func (s *EmailVerificationService) VerifyEmail(token string) error {
	// Get token from database
	verificationToken, err := s.verificationRepo.GetTokenByValue(token, verificationRepo.EmailVerification)
	if err != nil {
		if err == verificationRepo.ErrTokenNotFound {
			return ErrInvalidToken
		}
		if err == verificationRepo.ErrTokenExpired {
			return ErrTokenExpired
		}
		return fmt.Errorf("%w: %v", ErrVerifyToken, err)
	}

	// Mark user as verified
	if err := s.userRepo.UpdateEmailVerificationStatus(verificationToken.UserID, true); err != nil {
		return fmt.Errorf("%w: %v", ErrUpdateVerificationStatus, err)
	}

	// Delete the token as it's been used
	if err := s.verificationRepo.DeleteToken(verificationToken.ID); err != nil {
		// Just log this error, don't fail the verification
		fmt.Printf("Failed to delete used token: %v\n", err)
	}

	return nil
}

// ResendVerificationEmail generates a new token and sends a new verification email
func (s *EmailVerificationService) ResendVerificationEmail(userID int, ignoreCooldown bool) error {
	// Get user details
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrGetUser, err)
	}

	// If already verified, don't send again
	if user.EmailVerified {
		return ErrEmailAlreadyVerified
	}

	token, err := s.verificationRepo.GetTokenByUserID(user.ID, verificationRepo.EmailVerification)
	if err == nil {
		if s.environment == config.EnvTypeTest && !ignoreCooldown &&
			time.Now().Before(token.CreatedAt.Add(50*time.Second)) {
			// If token exists and was created less than 50 seconds ago, don't generate a new one
			return ErrEmailRecentlySent
		}
	}

	// Send verification email
	return s.SendVerificationEmail(user.ID, user.Email)
}

// IsTokenExpiredForEmail checks if a user has an expired token
func (s *EmailVerificationService) IsTokenExpiredForEmail(email string) (bool, error) {
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		if err == userrepo.ErrUserNotFound {
			return false, nil
		}
		return false, err
	}

	// User is verified, token doesn't matter
	if user.EmailVerified {
		return false, nil
	}

	// Check if user has a token
	token, err := s.verificationRepo.GetTokenByUserID(user.ID, verificationRepo.EmailVerification)
	if err != nil {
		if err == verificationRepo.ErrTokenNotFound {
			// No token means it's expired or never existed
			return true, nil
		}
		return false, err
	}

	// Check if token is expired
	return time.Now().After(token.ExpiresAt), nil
}
