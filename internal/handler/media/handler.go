package media

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	mediaservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
)

// MediaService определяет интерфейс для работы с медиа
type MediaService interface {
	UploadMedia(profileID int, mediaRole string, fileHeader mediaservice.UploadedFile) (*int, error)
	DeleteMedia(mediaID int) error
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

// @Summary      Upload media file
// @Description  Uploads a media file for a profile
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        profile_id  formData  int     true  "Profile ID"
// @Param        role        formData  string  true  "Media role (avatar, gallery, cover)"
// @Param        file        formData  file    true  "File to upload"
// @Success      201         {object}  MediaResponse
// @Failure      400         {string}  string  "Invalid data"
// @Failure      401         {string}  string  "Unauthorized"
// @Failure      413         {string}  string  "File too big"
// @Failure      415         {string}  string  "Unsupported file type"
// @Failure      500         {string}  string  "Internal server error"
// @Router       /api/media/upload [post]
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit uploaded file size (15 MB)
	r.Body = http.MaxBytesReader(w, r.Body, 15*1024*1024)

	// Parse multipart form
	err := r.ParseMultipartForm(15 * 1024 * 1024)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get profile ID
	profileIDStr := r.FormValue("profile_id")
	if profileIDStr == "" {
		http.Error(w, "Profile ID is required", http.StatusBadRequest)
		return
	}

	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Get media role
	mediaRole := r.FormValue("role")
	if mediaRole == "" {
		http.Error(w, "Media role is required", http.StatusBadRequest)
		return
	}

	// Validate media role
	validRoles := map[string]bool{
		"avatar":  true,
		"gallery": true,
		"cover":   true,
	}

	if !validRoles[mediaRole] {
		http.Error(w, "Invalid media role", http.StatusBadRequest)
		return
	}

	// Get file
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Upload media through service
	id, err := h.service.UploadMedia(profileID, mediaRole, &mediaservice.FileHeaderWrapper{FileHeader: fileHeader})
	if err != nil {
		switch err {
		case mediaservice.ErrFileTooBig:
			http.Error(w, "File too big", http.StatusRequestEntityTooLarge)
		case mediaservice.ErrInvalidFileType:
			http.Error(w, "Unsupported file type", http.StatusUnsupportedMediaType)
		default:
			http.Error(w, "Failed to upload media: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(UploadMediaResponse{Id: *id})
}

// @Summary      Delete media
// @Description  Deletes a media file
// @Tags         media
// @Security     BearerAuth
// @Param        id   path      int  true  "Media ID"
// @Success      204  {string}  string  ""
// @Failure      400  {string}  string  "Invalid ID"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Media not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/media/{id} [delete]
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract media ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	mediaIDStr := pathParts[len(pathParts)-1]
	mediaID, err := strconv.Atoi(mediaIDStr)
	if err != nil {
		http.Error(w, "Invalid media ID", http.StatusBadRequest)
		return
	}

	// Delete media
	err = h.service.DeleteMedia(mediaID)
	if err != nil {
		if err == mediaservice.ErrMediaNotFound {
			http.Error(w, "Media not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete media: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return empty response with 204 No Content status
	w.WriteHeader(http.StatusNoContent)
}
