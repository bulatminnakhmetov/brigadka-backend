package media

// API request models

// UploadRequest represents a media upload request
type UploadRequest struct {
	ProfileID int    `json:"profile_id"`
	MediaRole string `json:"role"`
}

// API response models

// MediaResponse represents the response for a single media
type UploadMediaResponse struct {
	Id int `json:"id"`
}
