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
	Profile *Profile `json:"base_profile"`
	Goal    string   `json:"goal"`
	Styles  []string `json:"styles"`
}

// MusicProfile представляет профиль пользователя для музыки
type MusicProfile struct {
	Profile     *Profile `json:"base_profile"`
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
	Goal   string   `json:"goal"`
	Styles []string `json:"styles"`
}

// CreateMusicProfileRequest представляет запрос на создание музыкального профиля
type CreateMusicProfileRequest struct {
	CreateProfileRequest
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// ProfileResponse представляет универсальный ответ для различных типов профилей
type ProfileResponse struct {
	Profile      *Profile       `json:"base_profile"`
	ImprovDetail *ImprovProfile `json:"improv_detail,omitempty"`
	MusicDetail  *MusicProfile  `json:"music_detail,omitempty"`
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

	var profile *Profile
	var err error

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

		profile, err = h.profileService.CreateImprovProfile(req.UserID, req.Description, req.Goal, req.Styles)

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

		profile, err = h.profileService.CreateMusicProfile(req.UserID, req.Description, req.Genres, req.Instruments)

	default:
		// Возвращаем ошибку вместо создания базового профиля
		http.Error(w, "Unsupported activity type", http.StatusBadRequest)
		return
	}

	if err != nil {
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
		return
	}

	// Формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(profile)
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

// Вспомогательная функция для перекодирования JSON из map в структуру
func remarshalJSON(data map[string]interface{}, target interface{}) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, target)
}
