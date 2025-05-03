package media

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrMediaNotFound = errors.New("media not found")
)

// Media представляет запись о медиафайле
type Media struct {
	ID        int       `json:"id"`
	ProfileID int       `json:"profile_id"`
	Type      string    `json:"type"`
	Role      string    `json:"role"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"uploaded_at"`
}

// RepositoryImpl implements the Repository interface
type RepositoryImpl struct {
	db *sql.DB
}

// NewRepository creates a new MediaRepository
func NewRepository(db *sql.DB) *RepositoryImpl {
	return &RepositoryImpl{
		db: db,
	}
}

// CreateMedia saves media information in the database
func (r *RepositoryImpl) CreateMedia(profileID int, mediaType, mediaRole, mediaURL string) (int, error) {
	var mediaID int
	err := r.db.QueryRow(
		"INSERT INTO media (profile_id, type, role, url) VALUES ($1, $2, $3, $4) RETURNING id",
		profileID, mediaType, mediaRole, mediaURL,
	).Scan(&mediaID)

	if err != nil {
		return 0, fmt.Errorf("failed to save media info: %w", err)
	}

	return mediaID, nil
}

// GetMediaByProfileAndRole retrieves all media for a profile with optional role filter
func (r *RepositoryImpl) GetMediaByProfileAndRole(profileID int, mediaRole string) ([]Media, error) {
	var query string
	var args []interface{}

	if mediaRole == "" {
		// If role is not specified, return all media for the profile
		query = "SELECT id, profile_id, type, role, url, uploaded_at FROM media WHERE profile_id = $1"
		args = []interface{}{profileID}
	} else {
		// If role is specified, filter by it
		query = "SELECT id, profile_id, type, role, url, uploaded_at FROM media WHERE profile_id = $1 AND role = $2"
		args = []interface{}{profileID, mediaRole}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	defer rows.Close()

	var mediaList []Media
	for rows.Next() {
		var media Media
		err := rows.Scan(
			&media.ID,
			&media.ProfileID,
			&media.Type,
			&media.Role,
			&media.URL,
			&media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media: %w", err)
		}
		mediaList = append(mediaList, media)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media rows: %w", err)
	}

	return mediaList, nil
}

// DeleteMediaByID deletes media by its ID
func (r *RepositoryImpl) DeleteMediaByID(mediaID int) error {
	_, err := r.db.Exec("DELETE FROM media WHERE id = $1", mediaID)
	if err != nil {
		return fmt.Errorf("failed to delete media from DB: %w", err)
	}
	return nil
}

// DeleteMediaByProfileAndRole deletes media for a profile with the specified role
func (r *RepositoryImpl) DeleteMediaByProfileAndRole(profileID int, mediaRole string) error {
	_, err := r.db.Exec("DELETE FROM media WHERE profile_id = $1 AND role = $2", profileID, mediaRole)
	if err != nil {
		return fmt.Errorf("failed to delete old media: %w", err)
	}
	return nil
}
