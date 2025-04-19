package media

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// MediaHandler обрабатывает запросы для работы с медиа
type MediaHandler struct {
	service MediaService
}

// NewMediaHandler создает новый экземпляр MediaHandler
func NewMediaHandler(service MediaService) *MediaHandler {
	return &MediaHandler{
		service: service,
	}
}

// @Summary      Загрузка медиа файла
// @Description  Загружает медиа файл для профиля
// @Tags         media
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        profile_id  formData  int     true  "ID профиля"
// @Param        role  formData  string  true  "Роль медиа (avatar, gallery, cover)"
// @Param        file        formData  file    true  "Файл для загрузки"
// @Success      201         {object}  MediaResponse
// @Failure      400         {string}  string  "Невалидные данные"
// @Failure      401         {string}  string  "Не авторизован"
// @Failure      413         {string}  string  "Файл слишком большой"
// @Failure      415         {string}  string  "Неподдерживаемый тип файла"
// @Failure      500         {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/media/upload [post]
func (h *MediaHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Ограничиваем размер загружаемого файла (15 МБ)
	r.Body = http.MaxBytesReader(w, r.Body, 15*1024*1024)

	// Парсим multipart form
	err := r.ParseMultipartForm(15 * 1024 * 1024)
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Получаем ID профиля
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

	// Получаем роль медиа
	mediaRole := r.FormValue("role")
	if mediaRole == "" {
		http.Error(w, "Media role is required", http.StatusBadRequest)
		return
	}

	// Валидация роли медиа
	validRoles := map[string]bool{
		"avatar":  true,
		"gallery": true,
		"cover":   true,
	}

	if !validRoles[mediaRole] {
		http.Error(w, "Invalid media role", http.StatusBadRequest)
		return
	}

	// Получаем файл
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Загружаем медиа через сервис
	media, err := h.service.UploadMedia(profileID, mediaRole, &FileHeaderWrapper{fileHeader})
	if err != nil {
		switch err {
		case ErrFileTooBig:
			http.Error(w, "File too big", http.StatusRequestEntityTooLarge)
		case ErrInvalidFileType:
			http.Error(w, "Unsupported file type", http.StatusUnsupportedMediaType)
		default:
			http.Error(w, "Failed to upload media: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ
	response := MediaResponse{
		Media: media,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary      Получение медиа по ID
// @Description  Возвращает информацию о медиа по ID
// @Tags         media
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "ID медиа"
// @Success      200  {object}  MediaResponse
// @Failure      400  {string}  string  "Невалидный ID"
// @Failure      401  {string}  string  "Не авторизован"
// @Failure      404  {string}  string  "Медиа не найдено"
// @Failure      500  {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/media/{id} [get]
func (h *MediaHandler) GetMedia(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID медиа из URL
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

	// Получаем медиа
	media, err := h.service.GetMedia(mediaID)
	if err != nil {
		if err == ErrMediaNotFound {
			http.Error(w, "Media not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Формируем ответ
	response := MediaResponse{
		Media: media,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Получение медиа для профиля
// @Description  Возвращает список медиа файлов для профиля
// @Tags         media
// @Produce      json
// @Security     BearerAuth
// @Param        profile_id   path      int     true   "ID профиля"
// @Param        role   query     string  false  "Роль медиа (фильтр)"
// @Success      200          {object}  MediaListResponse
// @Failure      400          {string}  string  "Невалидный ID профиля"
// @Failure      401          {string}  string  "Не авторизован"
// @Failure      500          {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/profiles/{profile_id}/media [get]
func (h *MediaHandler) GetMediaByProfile(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID профиля из URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// URL: /api/profiles/{profile_id}/media
	profileIDStr := pathParts[len(pathParts)-2]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Получаем опциональный параметр role для фильтрации
	mediaRole := r.URL.Query().Get("role")

	// Получаем медиа для профиля
	mediaList, err := h.service.GetMediaByProfile(profileID, mediaRole)
	if err != nil {
		http.Error(w, "Failed to get media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Формируем ответ
	response := MediaListResponse{
		Media: mediaList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Удаление медиа
// @Description  Удаляет медиа файл
// @Tags         media
// @Security     BearerAuth
// @Param        id   path      int  true  "ID медиа"
// @Success      204  {string}  string  ""
// @Failure      400  {string}  string  "Невалидный ID"
// @Failure      401  {string}  string  "Не авторизован"
// @Failure      404  {string}  string  "Медиа не найдено"
// @Failure      500  {string}  string  "Внутренняя ошибка сервера"
// @Router       /api/media/{id} [delete]
func (h *MediaHandler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID медиа из URL
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

	// Удаляем медиа
	err = h.service.DeleteMedia(mediaID)
	if err != nil {
		if err == ErrMediaNotFound {
			http.Error(w, "Media not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete media: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Возвращаем пустой ответ с кодом 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
