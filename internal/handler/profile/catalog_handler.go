package profile

import (
	"encoding/json"
	"net/http"
)

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

	items, err := h.profileService.GetActivityTypes(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CatalogResponse{Items: items})
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

	items, err := h.profileService.GetImprovGoals(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CatalogResponse{Items: items})
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

	items, err := h.profileService.GetMusicGenres(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CatalogResponse{Items: items})
}

// @Summary      Получение музыкальных инструментов
// @Description  Возвращает список доступных музыкальных инструментов
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        lang  query     string  false  "Код языка (по умолчанию 'ru')"
// @Success      200   CatalogResponse
// @Failure      500   {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/catalog/music-instruments [get]
func (h *ProfileHandler) GetMusicInstruments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lang := r.URL.Query().Get("lang")

	items, err := h.profileService.GetMusicInstruments(lang)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CatalogResponse{Items: items})
}
