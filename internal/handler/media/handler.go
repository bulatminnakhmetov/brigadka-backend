package media

import (
	"encoding/json"
	"net/http"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
)

// MediaService определяет интерфейс для работы с медиа
type MediaService interface {
	UploadMedia(userID int, fileHeader, thumbnailHeader media.UploadedFile) (*media.Media, error)
}

// MediaHandler handles requests for media operations
type MediaHandler struct {
	service MediaService
}

// NewMediaHandler creates a new instance of MediaHandler
func NewMediaHandler(service MediaService) *MediaHandler {
	return &MediaHandler{
		service: service,
	}
}

// Response for media operations
type MediaResponse struct {
	ID           int    `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// @Summary      Upload media
// @Description  Upload media file (image or video) with optional thumbnail
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Param        file       formData  file  true   "File to upload"
// @Param        thumbnail  formData  file  false  "Thumbnail file"
// @Success      200   {object}  MediaResponse
// @Failure      400   {string}  string  "Invalid file"
// @Failure      401   {string}  string  "Unauthorized"
// @Failure      413   {string}  string  "File too large"
// @Failure      500   {string}  string  "Internal server error"
// @Router       /api/media [post]
// @Security     BearerAuth
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (assuming it's set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	// Get main file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Could not get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create wrapper for the main file header
	fileWrapper := &media.FileHeaderWrapper{FileHeader: header}

	// Check for thumbnail file (optional)
	var thumbnailWrapper *media.FileHeaderWrapper
	thumbnailFile, thumbnailHeader, err := r.FormFile("thumbnail")
	if err != nil {
		http.Error(w, "Could not get thumbnail", http.StatusBadRequest)
		return
	}

	defer thumbnailFile.Close()

	thumbnailWrapper = &media.FileHeaderWrapper{FileHeader: thumbnailHeader}

	// Upload media
	uploaded, err := h.service.UploadMedia(userID, fileWrapper, thumbnailWrapper)
	if err != nil {
		switch err {
		case media.ErrInvalidFileType:
			http.Error(w, "Invalid file type", http.StatusBadRequest)
		case media.ErrFileTooBig:
			http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MediaResponse{
		ID:           uploaded.ID,
		URL:          uploaded.URL,
		ThumbnailURL: uploaded.ThumbnailURL,
	})
}
