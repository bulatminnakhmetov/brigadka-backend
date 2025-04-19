package media

import (
	"time"
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

// UploadRequest представляет запрос на загрузку медиа
type UploadRequest struct {
	ProfileID int    `json:"profile_id"`
	MediaRole string `json:"role"`
	// Файл передается отдельно в multipart/form-data
}

// MediaResponse представляет ответ на загрузку медиа
type MediaResponse struct {
	Media *Media `json:"media"`
}

// MediaListResponse представляет ответ со списком медиа
type MediaListResponse struct {
	Media []Media `json:"media"`
}

// MediaService определяет интерфейс для работы с медиа
type MediaService interface {
	UploadMedia(profileID int, mediaRole string, fileHeader UploadedFile) (*Media, error)
	GetMedia(mediaID int) (*Media, error)
	GetMediaByProfile(profileID int, mediaRole string) ([]Media, error)
	DeleteMedia(mediaID int) error
}
