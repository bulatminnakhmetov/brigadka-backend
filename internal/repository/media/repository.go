package media

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

var (
	ErrMediaNotFound = errors.New("media not found")
)

// Media представляет запись о медиафайле
type Media struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Role         string    `json:"role"`
	URL          string    `json:"url"`
	ThumbnailURL string    `json:"thumbnail_url"`
	UploadedAt   time.Time `json:"uploaded_at"`
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
func (r *RepositoryImpl) CreateMedia(userID int, mediaType, mediaURL, thumbnailURL string) (int, error) {
	var mediaID int
	err := r.db.QueryRow(
		"INSERT INTO media (owner_id, type, url, thumbnail_url) VALUES ($1, $2, $3, $4) RETURNING id",
		userID, mediaType, mediaURL, thumbnailURL,
	).Scan(&mediaID)

	if err != nil {
		return 0, fmt.Errorf("failed to save media info: %w", err)
	}

	return mediaID, nil
}

// DeleteMediaByID deletes media by its ID
func (r *RepositoryImpl) DeleteMedia(userID, mediaID int) error {
	_, err := r.db.Exec("DELETE FROM media WHERE id = $1 AND owner_id = $2", mediaID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete media from DB: %w", err)
	}
	return nil
}

// GetMediaByID retrieves media by its ID
func (r *RepositoryImpl) GetMediaByID(userID int, mediaID int) (*Media, error) {
	var m Media
	err := r.db.QueryRow(
		"SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media WHERE id = $1 AND owner_id = $2",
		mediaID, userID,
	).Scan(&m.ID, &m.UserID, &m.Role, &m.URL, &m.ThumbnailURL, &m.UploadedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMediaNotFound
		}
		return nil, fmt.Errorf("failed to get media from DB: %w", err)
	}

	return &m, nil
}

// GetMediaByID retrieves media by its ID
func (r *RepositoryImpl) GetMediaByIDs(userID int, mediaIDs []int) ([]Media, error) {
	if len(mediaIDs) == 0 {
		return nil, nil
	}
	result := make([]Media, 0, len(mediaIDs))
	for _, id := range mediaIDs {
		media, err := r.GetMediaByID(userID, id)
		if err != nil {
			log.Printf("failed to get media by ID %d: %v", id, err)
			continue
		}
		result = append(result, *media)
	}
	return result, nil
}
