package profile

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Общие типы профилей
const (
	ActivityTypeImprov = "improv"
	ActivityTypeMusic  = "music"
)

// Profile представляет базовый профиль пользователя
type Profile struct {
	ProfileID    int       `json:"profile_id"`
	UserID       int       `json:"user_id"`
	Description  string    `json:"description"`
	ActivityType string    `json:"activity_type"`
	CreatedAt    time.Time `json:"created_at"`
}

// ImprovProfile представляет профиль пользователя для импровизации
type ImprovProfile struct {
	Profile
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// MusicProfile представляет профиль пользователя для музыки
type MusicProfile struct {
	Profile
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// CreateProfileRequest представляет базовый запрос на создание профиля
type CreateProfileRequest struct {
	UserID      int    `json:"user_id"`
	Description string `json:"description"`
}

// CreateImprovProfileRequest представляет запрос на создание профиля импровизации
type CreateImprovProfileRequest struct {
	CreateProfileRequest
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// CreateMusicProfileRequest представляет запрос на создание музыкального профиля
type CreateMusicProfileRequest struct {
	CreateProfileRequest
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// UpdateProfileRequest представляет базовый запрос на обновление профиля
type UpdateProfileRequest struct {
	Description string `json:"description"`
}

// UpdateImprovProfileRequest представляет запрос на обновление профиля импровизации
type UpdateImprovProfileRequest struct {
	UpdateProfileRequest
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// UpdateMusicProfileRequest представляет запрос на обновление музыкального профиля
type UpdateMusicProfileRequest struct {
	UpdateProfileRequest
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// UserProfilesResponse represents the response format for user profiles
type UserProfilesResponse struct {
	Profiles map[string]int `json:"profiles"` // activity_type -> profile_id
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

// handleError обрабатывает ошибки и возвращает соответствующий HTTP-статус
func handleError(w http.ResponseWriter, err error) {
	// Возвращаем различные коды состояния HTTP в зависимости от типа ошибки
	switch {
	case errors.Is(err, ErrUserNotFound):
		http.Error(w, "User not found", http.StatusNotFound)
	case errors.Is(err, ErrInvalidActivityType):
		http.Error(w, "Invalid activity type", http.StatusBadRequest)
	case errors.Is(err, ErrProfileAlreadyExists):
		http.Error(w, "Profile already exists for this user", http.StatusConflict)
	case errors.Is(err, ErrInvalidImprovGoal):
		http.Error(w, "Invalid improv goal", http.StatusBadRequest)
	case errors.Is(err, ErrInvalidImprovStyle):
		http.Error(w, "Invalid improv style", http.StatusBadRequest)
	case errors.Is(err, ErrInvalidMusicGenre):
		http.Error(w, "Invalid music genre", http.StatusBadRequest)
	case errors.Is(err, ErrInvalidInstrument):
		http.Error(w, "Invalid instrument", http.StatusBadRequest)
	case errors.Is(err, ErrEmptyInstruments):
		http.Error(w, "At least one instrument is required", http.StatusBadRequest)
	default:
		http.Error(w, "Failed to create profile: "+err.Error(), http.StatusInternalServerError)
	}
}

// @Summary      Get improv profile
// @Description  Gets an improv profile by ID
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Profile ID"
// @Success      200  {object}  ImprovProfile
// @Failure      400  {string}  string  "Invalid ID or profile type"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/profiles/{id}/improv [get]
func (h *ProfileHandler) GetImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-2]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем профиль импровизации напрямую
	profile, err := h.profileService.GetImprovProfile(profileID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			http.Error(w, "Profile not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidActivityType):
			http.Error(w, "Profile is not an improv profile", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Get music profile
// @Description  Gets a music profile by ID
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Profile ID"
// @Success      200  {object}  MusicProfile
// @Failure      400  {string}  string  "Invalid ID or profile type"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/profiles/{id}/music [get]
func (h *ProfileHandler) GetMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-2]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем музыкальный профиль напрямую
	profile, err := h.profileService.GetMusicProfile(profileID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			http.Error(w, "Profile not found", http.StatusNotFound)
		case errors.Is(err, ErrInvalidActivityType):
			http.Error(w, "Profile is not a music profile", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Получение типов активности
// @Description  Возвращает список доступных типов активности профиля
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   {array}   TranslatedItem
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/activity-types [get]
func (h *ProfileHandler) GetActivityTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	catalog, err := h.profileService.GetActivityTypes(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// @Summary      Получение стилей импровизации
// @Description  Возвращает список доступных стилей импровизации
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   {array}   TranslatedItem
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/improv-styles [get]
func (h *ProfileHandler) GetImprovStyles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	catalog, err := h.profileService.GetImprovStyles(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// @Summary      Получение целей импровизации
// @Description  Возвращает список доступных целей для занятий импровизацией
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   {array}   TranslatedItem
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/improv-goals [get]
func (h *ProfileHandler) GetImprovGoals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	catalog, err := h.profileService.GetImprovGoals(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// @Summary      Получение музыкальных жанров
// @Description  Возвращает список доступных музыкальных жанров
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   {array}   TranslatedItem
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/music-genres [get]
func (h *ProfileHandler) GetMusicGenres(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	catalog, err := h.profileService.GetMusicGenres(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// @Summary      Получение музыкальных инструментов
// @Description  Возвращает список доступных музыкальных инструментов
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   {array}   TranslatedItem
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/music-instruments [get]
func (h *ProfileHandler) GetMusicInstruments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	catalog, err := h.profileService.GetMusicInstruments(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(catalog)
}

// Add these methods to the ProfileHandler struct

// @Summary      Create improv profile
// @Description  Creates a new improv profile for a user
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateImprovProfileRequest  true  "Improv profile data"
// @Success      201      {object}  ImprovProfile
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      409      {string}  string  "Profile already exists"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/improv [post]
func (h *ProfileHandler) CreateImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req CreateImprovProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if req.UserID <= 0 {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	if req.Goal == "" {
		http.Error(w, "Improv goal is required", http.StatusBadRequest)
		return
	}

	if len(req.Styles) == 0 {
		http.Error(w, "At least one improv style is required", http.StatusBadRequest)
		return
	}

	profile, err := h.profileService.CreateImprovProfile(
		req.UserID,
		req.Description,
		req.Goal,
		req.Styles,
		req.LookingForTeam,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Create music profile
// @Description  Creates a new music profile for a user
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateMusicProfileRequest  true  "Music profile data"
// @Success      201      {object}  MusicProfile
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      409      {string}  string  "Profile already exists"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/music [post]
func (h *ProfileHandler) CreateMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var req CreateMusicProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if req.UserID <= 0 {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	if len(req.Instruments) == 0 {
		http.Error(w, "At least one instrument is required", http.StatusBadRequest)
		return
	}

	profile, err := h.profileService.CreateMusicProfile(
		req.UserID,
		req.Description,
		req.Genres,
		req.Instruments,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Update improv profile
// @Description  Updates an existing improv profile
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Profile ID"
// @Param        request  body      UpdateImprovProfileRequest  true  "Updated improv profile data"
// @Success      200      {object}  ImprovProfile
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      403      {string}  string  "Forbidden"
// @Failure      404      {string}  string  "Profile not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/{id}/improv [put]
func (h *ProfileHandler) UpdateImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-2] // Изменено для нового формата URL
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем ID пользователя из контекста для проверки прав
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDValue.(int)

	// Проверяем, принадлежит ли профиль текущему пользователю
	// Получаем текущий профиль
	currentProfile, err := h.profileService.GetImprovProfile(profileID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Проверяем права доступа
	if currentProfile.UserID != userID {
		http.Error(w, "Forbidden: you can only update your own profile", http.StatusForbidden)
		return
	}

	// Читаем тело запроса
	var req UpdateImprovProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if req.Goal == "" {
		http.Error(w, "Improv goal is required", http.StatusBadRequest)
		return
	}

	if len(req.Styles) == 0 {
		http.Error(w, "At least one improv style is required", http.StatusBadRequest)
		return
	}

	profile, err := h.profileService.UpdateImprovProfile(
		profileID,
		req.Description,
		req.Goal,
		req.Styles,
		req.LookingForTeam,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Update music profile
// @Description  Updates an existing music profile
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Profile ID"
// @Param        request  body      UpdateMusicProfileRequest  true  "Updated music profile data"
// @Success      200      {object}  MusicProfile
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      403      {string}  string  "Forbidden"
// @Failure      404      {string}  string  "Profile not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/{id}/music [put]
func (h *ProfileHandler) UpdateMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-2] // Изменено для нового формата URL
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем ID пользователя из контекста для проверки прав
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDValue.(int)

	// Проверяем, принадлежит ли профиль текущему пользователю
	// Получаем текущий профиль
	currentProfile, err := h.profileService.GetMusicProfile(profileID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Проверяем права доступа
	if currentProfile.UserID != userID {
		http.Error(w, "Forbidden: you can only update your own profile", http.StatusForbidden)
		return
	}

	// Читаем тело запроса
	var req UpdateMusicProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if len(req.Instruments) == 0 {
		http.Error(w, "At least one instrument is required", http.StatusBadRequest)
		return
	}

	profile, err := h.profileService.UpdateMusicProfile(
		profileID,
		req.Description,
		req.Genres,
		req.Instruments,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// @Summary      Get user profiles
// @Description  Gets all profiles for a specific user
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        user_id  path  int  true  "User ID"
// @Success      200      {object}  UserProfilesResponse
// @Failure      400      {string}  string  "Invalid user ID"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      403      {string}  string  "Forbidden"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/{user_id} [get]
func (h *ProfileHandler) GetUserProfiles(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 1 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	userIDStr := pathParts[len(pathParts)-1]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check authentication
	currentUserIDValue := r.Context().Value("user_id")
	if currentUserIDValue == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if currentUserID := currentUserIDValue.(int); userID != currentUserID {
		http.Error(w, "Forbidden: you can only view your own profiles", http.StatusForbidden)
		return
	}

	// Get all profiles for the user
	profiles, err := h.profileService.GetUserProfiles(userID)
	if err != nil {
		http.Error(w, "Failed to get user profiles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := UserProfilesResponse{
		Profiles: profiles,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
