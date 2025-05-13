package push

import (
	"context"
	"database/sql"
	"time"
)

// PushToken represents a device push notification token
type PushToken struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	Token      string    `json:"token"`
	Platform   string    `json:"platform"`
	DeviceID   string    `json:"device_id"`
	LastSeenAt time.Time `json:"last_seen_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Repository defines methods for push token storage
type Repository interface {
	SaveToken(ctx context.Context, token PushToken) (int, error)
	GetUserTokens(ctx context.Context, userID int) ([]PushToken, error)
	DeleteToken(ctx context.Context, token string) error
	UpdateLastSeen(ctx context.Context, token string) error
}

type postgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new push token repository
func NewPostgresRepository(db *sql.DB) Repository {
	return &postgresRepository{
		db: db,
	}
}

// SaveToken saves or updates a push token
func (r *postgresRepository) SaveToken(ctx context.Context, token PushToken) (int, error) {
	query := `
        INSERT INTO push_tokens (user_id, token, platform, device_id, last_seen_at)
        VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (token) DO UPDATE 
        SET user_id = $1, platform = $3, device_id = $4, last_seen_at = NOW(), updated_at = NOW()
        RETURNING id`

	var id int
	err := r.db.QueryRowContext(ctx, query, token.UserID, token.Token, token.Platform, token.DeviceID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserTokens retrieves all push tokens for a user
func (r *postgresRepository) GetUserTokens(ctx context.Context, userID int) ([]PushToken, error) {
	query := `
        SELECT id, user_id, token, platform, device_id, last_seen_at, created_at, updated_at
        FROM push_tokens
        WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []PushToken
	for rows.Next() {
		var token PushToken
		if err := rows.Scan(
			&token.ID,
			&token.UserID,
			&token.Token,
			&token.Platform,
			&token.DeviceID,
			&token.LastSeenAt,
			&token.CreatedAt,
			&token.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// DeleteToken removes a push token
func (r *postgresRepository) DeleteToken(ctx context.Context, token string) error {
	query := `DELETE FROM push_tokens WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// UpdateLastSeen updates the last_seen_at timestamp for a token
func (r *postgresRepository) UpdateLastSeen(ctx context.Context, token string) error {
	query := `
        UPDATE push_tokens 
        SET last_seen_at = NOW(), updated_at = NOW() 
        WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}
