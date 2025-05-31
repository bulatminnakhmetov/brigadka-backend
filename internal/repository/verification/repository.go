package verification

import (
	"database/sql"
	"errors"
	"time"
)

// TokenType defines the type of verification token
type TokenType string

const (
	// EmailVerification token type for email verification
	EmailVerification TokenType = "email_verification"

	// PasswordReset token type for password reset
	PasswordReset TokenType = "password_reset"
)

// VerificationToken represents a token used for account verification
type VerificationToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"token"`
	Type      TokenType `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Repository errors
var (
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)

// Repository interface defines methods for managing verification tokens
type Repository interface {
	CreateToken(token *VerificationToken) error
	GetTokenByValue(tokenValue string, tokenType TokenType) (*VerificationToken, error)
	GetTokenByUserID(userID int, tokenType TokenType) (*VerificationToken, error)
	DeleteToken(tokenID int) error
	DeleteExpiredTokens() error
	DeleteTokensByUserID(userID int, tokenType TokenType) error
}

// PostgresRepository implements the verification token repository
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL verification repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateToken saves a new verification token to the database
func (r *PostgresRepository) CreateToken(token *VerificationToken) error {
	// First, delete any existing tokens of the same type for this user
	err := r.DeleteTokensByUserID(token.UserID, token.Type)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO verification_tokens (user_id, token, type, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err = r.db.QueryRow(
		query,
		token.UserID,
		token.Token,
		token.Type,
		token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)

	return err
}

// GetTokenByValue retrieves a token by its value and type
func (r *PostgresRepository) GetTokenByValue(tokenValue string, tokenType TokenType) (*VerificationToken, error) {
	query := `
		SELECT id, user_id, token, type, expires_at, created_at
		FROM verification_tokens
		WHERE token = $1 AND type = $2
	`

	var token VerificationToken
	err := r.db.QueryRow(query, tokenValue, tokenType).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.Type,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		return &token, ErrTokenExpired
	}

	return &token, nil
}

// GetTokenByUserID retrieves a token by user ID and type
func (r *PostgresRepository) GetTokenByUserID(userID int, tokenType TokenType) (*VerificationToken, error) {
	query := `
		SELECT id, user_id, token, type, expires_at, created_at
		FROM verification_tokens
		WHERE user_id = $1 AND type = $2
	`

	var token VerificationToken
	err := r.db.QueryRow(query, userID, tokenType).Scan(
		&token.ID,
		&token.UserID,
		&token.Token,
		&token.Type,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	return &token, nil
}

// DeleteToken deletes a token by ID
func (r *PostgresRepository) DeleteToken(tokenID int) error {
	query := `
		DELETE FROM verification_tokens
		WHERE id = $1
	`

	_, err := r.db.Exec(query, tokenID)
	return err
}

// DeleteExpiredTokens removes all expired tokens from the database
func (r *PostgresRepository) DeleteExpiredTokens() error {
	query := `
		DELETE FROM verification_tokens
		WHERE expires_at < NOW()
	`

	_, err := r.db.Exec(query)
	return err
}

// DeleteTokensByUserID deletes all tokens of a specific type for a user
func (r *PostgresRepository) DeleteTokensByUserID(userID int, tokenType TokenType) error {
	query := `
		DELETE FROM verification_tokens
		WHERE user_id = $1 AND type = $2
	`

	_, err := r.db.Exec(query, userID, tokenType)
	return err
}
