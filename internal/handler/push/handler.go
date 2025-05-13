package push

import (
	"encoding/json"
	"net/http"

	pushservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/push"
)

// TokenRequest represents a push token registration request
type TokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
	DeviceID string `json:"device_id,omitempty"`
}

// Handler handles push notification endpoints
type Handler struct {
	service pushservice.PushService
}

// NewHandler creates a new push notification handler
func NewHandler(service pushservice.PushService) *Handler {
	return &Handler{
		service: service,
	}
}

// RegisterToken godoc
// @Summary Register a push notification token
// @Description Register a device push notification token for the current user
// @Tags push
// @Accept json
// @Produce json
// @Param token body TokenRequest true "Push Token Information"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /api/push/register [post]
func (h *Handler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	if req.Platform == "" {
		http.Error(w, "Platform is required", http.StatusBadRequest)
		return
	}

	if err := h.service.SaveToken(r.Context(), userID, req.Token, req.Platform, req.DeviceID); err != nil {
		http.Error(w, "Failed to save token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// UnregisterToken godoc
// @Summary Unregister a push notification token
// @Description Unregister a device push notification token
// @Tags push
// @Accept json
// @Produce json
// @Param token body TokenRequest true "Push Token Information"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/push/unregister [delete]
func (h *Handler) UnregisterToken(w http.ResponseWriter, r *http.Request) {
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteToken(r.Context(), req.Token); err != nil {
		http.Error(w, "Failed to delete token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Helper function to send JSON responses
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
