package push

import (
	"encoding/json"
	"log"
	"net/http"

	pushservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/push"
)

// RegisterTokenRequest represents a push token registration request
type RegisterTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
	DeviceID string `json:"device_id,omitempty"`
}

// RegisterTokenRequest represents a push token registration request
type UnregisterTokenRequest struct {
	Token string `json:"token"`
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
// @Param token body RegisterTokenRequest true "Push Token Information"
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

	var req RegisterTokenRequest
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

	log.Printf("User %d registered token %s for platform %s", userID, req.Token, req.Platform)

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// UnregisterToken godoc
// @Summary Unregister a push notification token
// @Description Unregister a device push notification token
// @Tags push
// @Accept json
// @Produce json
// @Param token body UnregisterTokenRequest true "Push Token Information"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/push/unregister [delete]
func (h *Handler) UnregisterToken(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req UnregisterTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteToken(r.Context(), userID, req.Token); err != nil {
		if err == pushservice.ErrTokenNotFound {
			http.Error(w, "Token does not exist", http.StatusBadRequest)
			return
		}

		http.Error(w, "Failed to delete token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("User %d unregistered token %s", userID, req.Token)

	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Helper function to send JSON responses
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
