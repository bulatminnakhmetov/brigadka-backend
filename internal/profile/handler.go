package profile

import (
	"encoding/json"
	"net/http"
	"time"
)

// Profile представляет профиль пользователя
type Profile struct {
	ProfileID    int       `json:"profile_id"`
	UserID       int       `json:"user_id"`
	Description  string    `json:"description"`
	ActivityType string    `json:"activity_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateProfileRequest представляет запрос на создание профиля
type CreateProfileRequest struct {
	UserID       int    `json:"user_id"`
	Description  string `json:"description"`
	ActivityType string `json:"activity_type"`
}

// ProfileHandler обрабатывает запросы, связанные с профилями
type ProfileHandler struct {
	profileService ProfileService
}

// NewProfileHandler создает новый экземпляр ProfileHandler
func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// CreateProfile обрабатывает запрос на создание нового профиля
func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем тело запроса
	var req CreateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if req.UserID <= 0 {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	if req.ActivityType == "" {
		http.Error(w, "Activity type is required", http.StatusBadRequest)
		return
	}

	// Создаем профиль с помощью сервиса
	profile, err := h.profileService.CreateProfile(req.UserID, req.Description, req.ActivityType)
	if err != nil {
		http.Error(w, "Failed to create profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// ProfileService интерфейс для работы с профилями
type ProfileService interface {
	CreateProfile(userID int, description string, activityType string) (*Profile, error)
}
