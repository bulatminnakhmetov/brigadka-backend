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
	UserID       int    `json:"user_id"`
	Description  string `json:"description"`
	ActivityType string `json:"activity_type"`
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

// ProfileResponse представляет универсальный ответ для различных типов профилей
type ProfileResponse struct {
	ImprovProfile *ImprovProfile `json:"improv_profile,omitempty"`
	MusicProfile  *MusicProfile  `json:"music_profile,omitempty"`
}

// UpdateProfileRequest представляет базовый запрос на обновление профиля
type UpdateProfileRequest struct {
	Description  string `json:"description"`
	ActivityType string `json:"activity_type"`
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

// @Summary      Создание профиля
// @Description  Создает новый профиль для пользователя
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateProfileRequest  true  "Данные профиля"
// @Success      201      {object}  Profile
// @Failure      400      {string}  string  "Невалидные данные"
// @Failure      401      {string}  string  "Не авторизован"
// @Failure      500      {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles [post]
func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Извлекаем activity_type для определения типа профиля
	activityType, ok := body["activity_type"].(string)
	if !ok || activityType == "" {
		http.Error(w, "Activity type is required", http.StatusBadRequest)
		return
	}

	// В зависимости от типа активности, создаем соответствующий профиль
	switch activityType {
	case ActivityTypeImprov:
		var req CreateImprovProfileRequest
		if err := remarshalJSON(body, &req); err != nil {
			http.Error(w, "Invalid request body for improv profile", http.StatusBadRequest)
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

	case ActivityTypeMusic:
		var req CreateMusicProfileRequest
		if err := remarshalJSON(body, &req); err != nil {
			http.Error(w, "Invalid request body for music profile", http.StatusBadRequest)
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

		profile, err := h.profileService.CreateMusicProfile(req.UserID, req.Description, req.Genres, req.Instruments)

		if err != nil {
			handleError(w, err)
			return
		}

		// Формируем ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(profile)

	default:
		// Возвращаем ошибку вместо создания базового профиля
		http.Error(w, "Unsupported activity type", http.StatusBadRequest)
		return
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

// @Summary      Получение профиля
// @Description  Получает профиль пользователя по ID
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "ID профиля"
// @Success      200  {object}  ProfileResponse
// @Failure      400  {string}  string  "Невалидный ID"
// @Failure      401  {string}  string  "Не авторизован"
// @Failure      404  {string}  string  "Профиль не найден"
// @Failure      500  {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/{id} [get]
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем профиль
	profileResp, err := h.profileService.GetProfile(profileID)
	if err != nil {
		switch {
		case errors.Is(err, ErrProfileNotFound):
			http.Error(w, "Profile not found", http.StatusNotFound)
		default:
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResp)
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

// @Summary      Обновление профиля
// @Description  Обновляет существующий профиль пользователя
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "ID профиля"
// @Param        request  body      UpdateProfileRequest  true  "Данные для обновления профиля"
// @Success      200      {object}  ProfileResponse
// @Failure      400      {string}  string  "Невалидные данные"
// @Failure      401      {string}  string  "Не авторизован"
// @Failure      403      {string}  string  "Доступ запрещен"
// @Failure      404      {string}  string  "Профиль не найден"
// @Failure      500      {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/{id} [put]
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
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
	currentProfile, err := h.profileService.GetProfile(profileID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Определяем ID владельца профиля
	var ownerID int
	if currentProfile.ImprovProfile != nil {
		ownerID = currentProfile.ImprovProfile.UserID
	} else if currentProfile.MusicProfile != nil {
		ownerID = currentProfile.MusicProfile.UserID
	}

	// Проверяем права доступа
	if ownerID != userID {
		http.Error(w, "Forbidden: you can only update your own profile", http.StatusForbidden)
		return
	}

	// Читаем тело запроса
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Извлекаем activity_type для определения типа профиля
	activityType, ok := body["activity_type"].(string)
	if !ok || activityType == "" {
		http.Error(w, "Activity type is required", http.StatusBadRequest)
		return
	}

	// В зависимости от типа активности, обновляем соответствующий профиль
	switch activityType {
	case ActivityTypeImprov:
		var req UpdateImprovProfileRequest
		if err := remarshalJSON(body, &req); err != nil {
			http.Error(w, "Invalid request body for improv profile", http.StatusBadRequest)
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

		response := ProfileResponse{
			ImprovProfile: profile,
		}
		json.NewEncoder(w).Encode(response)

	case ActivityTypeMusic:
		var req UpdateMusicProfileRequest
		if err := remarshalJSON(body, &req); err != nil {
			http.Error(w, "Invalid request body for music profile", http.StatusBadRequest)
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

		response := ProfileResponse{
			MusicProfile: profile,
		}
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Unsupported activity type", http.StatusBadRequest)
		return
	}
}

// Вспомогательная функция для перекодирования JSON из map в структуру
func remarshalJSON(data map[string]interface{}, target interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
