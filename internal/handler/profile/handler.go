package profile

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/profile"
)

// ProfileService интерфейс для работы с профилями
type ProfileService interface {
	CreateImprovProfile(userID int, description string, goal string, styles []string, lookingForTeam bool) (*profile.ImprovProfile, error)
	CreateMusicProfile(userID int, description string, genres []string, instruments []string) (*profile.MusicProfile, error)
	GetImprovProfile(profileID int) (*profile.ImprovProfile, error)
	GetMusicProfile(profileID int) (*profile.MusicProfile, error)

	// Methods for updating profiles
	UpdateImprovProfile(profileID int, description string, goal string, styles []string, lookingForTeam bool, fullName string, gender string, age int, cityID int) (*profile.ImprovProfile, error)
	UpdateMusicProfile(profileID int, description string, genres []string, instruments []string, fullName string, gender string, age int, cityID int) (*profile.MusicProfile, error)

	GetActivityTypes(lang string) ([]profile.TranslatedItem, error)
	GetImprovStyles(lang string) ([]profile.TranslatedItem, error)
	GetImprovGoals(lang string) ([]profile.TranslatedItem, error)
	GetMusicGenres(lang string) ([]profile.TranslatedItem, error)
	GetMusicInstruments(lang string) ([]profile.TranslatedItem, error)

	GetUserProfiles(userID int) (map[string]int, error)
}

// MediaService определяет интерфейс для работы с медиа
type MediaService interface {
	GetProfileMedia(profileID int) (*media.ProfileMedia, error)
}

// ProfileHandler handles requests related to profiles
type ProfileHandler struct {
	profileService ProfileService
	mediaService   MediaService
}

// NewProfileHandler creates a new instance of ProfileHandler
func NewProfileHandler(profileService ProfileService, mediaService MediaService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		mediaService:   mediaService,
	}
}

// handleError handles errors and returns appropriate HTTP status
func handleError(w http.ResponseWriter, err error) {
	// Return different HTTP status codes based on error type
	switch {
	case errors.Is(err, profile.ErrUserNotFound):
		http.Error(w, "User not found", http.StatusNotFound)
	case errors.Is(err, profile.ErrInvalidActivityType):
		http.Error(w, "Invalid activity type", http.StatusBadRequest)
	case errors.Is(err, profile.ErrProfileAlreadyExists):
		http.Error(w, "Profile already exists for this user", http.StatusConflict)
	case errors.Is(err, profile.ErrInvalidImprovGoal):
		http.Error(w, "Invalid improv goal", http.StatusBadRequest)
	case errors.Is(err, profile.ErrInvalidImprovStyle):
		http.Error(w, "Invalid improv style", http.StatusBadRequest)
	case errors.Is(err, profile.ErrInvalidMusicGenre):
		http.Error(w, "Invalid music genre", http.StatusBadRequest)
	case errors.Is(err, profile.ErrInvalidInstrument):
		http.Error(w, "Invalid instrument", http.StatusBadRequest)
	case errors.Is(err, profile.ErrEmptyInstruments):
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
// @Success      200  {object}  ImprovProfileResponse
// @Failure      400  {string}  string  "Invalid ID or profile type"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/profiles/improv/{id} [get]
func (h *ProfileHandler) GetImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract profile ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Get improv profile directly
	improvProfile, err := h.profileService.GetImprovProfile(profileID)
	if err != nil {
		switch {
		case errors.Is(err, profile.ErrProfileNotFound):
			http.Error(w, "Profile not found", http.StatusNotFound)
		case errors.Is(err, profile.ErrInvalidActivityType):
			http.Error(w, "Profile is not an improv profile", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	profileMedia, err := h.mediaService.GetProfileMedia(improvProfile.UserID)
	if err != nil {
		log.Printf("Failed to get profile media: %v", err)
	}

	// Convert to API response model
	response := ToImprovProfileResponse(improvProfile, profileMedia)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Get music profile
// @Description  Gets a music profile by ID
// @Tags         profiles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Profile ID"
// @Success      200  {object}  MusicProfileResponse
// @Failure      400  {string}  string  "Invalid ID or profile type"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Internal server error"
// @Router       /api/profiles/music/{id} [get]
func (h *ProfileHandler) GetMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract profile ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Get music profile directly
	musicProfile, err := h.profileService.GetMusicProfile(profileID)
	if err != nil {
		switch {
		case errors.Is(err, profile.ErrProfileNotFound):
			http.Error(w, "Profile not found", http.StatusNotFound)
		case errors.Is(err, profile.ErrInvalidActivityType):
			http.Error(w, "Profile is not a music profile", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	profileMedia, err := h.mediaService.GetProfileMedia(musicProfile.UserID)
	if err != nil {
		log.Printf("Failed to get profile media: %v", err)
	}

	// Convert to API response model
	response := ToMusicProfileResponse(musicProfile, profileMedia)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Create improv profile
// @Description  Creates a new improv profile for a user
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateImprovProfileRequest  true  "Improv profile data"
// @Success      201      {object}  ImprovProfileDTO
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      409      {string}  string  "Profile already exists"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/improv [post]
func (h *ProfileHandler) CreateImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CreateImprovProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Field validation
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

	// Convert to API response model
	response := ToImprovProfileDTO(profile)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary      Create music profile
// @Description  Creates a new music profile for a user
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body  CreateMusicProfileRequest  true  "Music profile data"
// @Success      201      {object}  MusicProfileDTO
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      409      {string}  string  "Profile already exists"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/music [post]
func (h *ProfileHandler) CreateMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req CreateMusicProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Field validation
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

	// Convert to API response model
	response := ToMusicProfileDTO(profile)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary      Update improv profile
// @Description  Updates an existing improv profile
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Profile ID"
// @Param        request  body      UpdateImprovProfileRequest  true  "Updated improv profile data"
// @Success      200      {object}  ImprovProfileResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      403      {string}  string  "Forbidden"
// @Failure      404      {string}  string  "Profile not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/improv/{id} [put]
func (h *ProfileHandler) UpdateImprovProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract profile ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Get user ID from context for permission check
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDValue.(int)

	// Check if the profile belongs to the current user
	// Get the current profile
	currentProfile, err := h.profileService.GetImprovProfile(profileID)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check permissions
	if currentProfile.UserID != userID {
		http.Error(w, "Forbidden: you can only update your own profile", http.StatusForbidden)
		return
	}

	// Parse request body
	var req UpdateImprovProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Field validation
	if req.Goal == "" {
		http.Error(w, "Improv goal is required", http.StatusBadRequest)
		return
	}

	if len(req.Styles) == 0 {
		http.Error(w, "At least one improv style is required", http.StatusBadRequest)
		return
	}

	updatedProfile, err := h.profileService.UpdateImprovProfile(
		profileID,
		req.Description,
		req.Goal,
		req.Styles,
		req.LookingForTeam,
		req.FullName,
		req.Gender,
		req.Age,
		req.CityID,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	profileMedia, err := h.mediaService.GetProfileMedia(updatedProfile.UserID)
	if err != nil {
		log.Printf("Failed to get profile media: %v", err)
	}

	// Convert to API response model
	response := ToImprovProfileResponse(updatedProfile, profileMedia)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Update music profile
// @Description  Updates an existing music profile
// @Tags         profiles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int  true  "Profile ID"
// @Param        request  body      UpdateMusicProfileRequest  true  "Updated music profile data"
// @Success      200      {object}  MusicProfileResponse
// @Failure      400      {string}  string  "Invalid data"
// @Failure      401      {string}  string  "Unauthorized"
// @Failure      403      {string}  string  "Forbidden"
// @Failure      404      {string}  string  "Profile not found"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/profiles/music/{id} [put]
func (h *ProfileHandler) UpdateMusicProfile(w http.ResponseWriter, r *http.Request) {
	// Check request method
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract profile ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	profileIDStr := pathParts[len(pathParts)-1]
	profileID, err := strconv.Atoi(profileIDStr)
	if err != nil {
		http.Error(w, "Invalid profile ID", http.StatusBadRequest)
		return
	}

	// Get user ID from context for permission check
	userIDValue := r.Context().Value("user_id")
	if userIDValue == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID := userIDValue.(int)

	// Check if the profile belongs to the current user
	// Get the current profile
	currentProfile, err := h.profileService.GetMusicProfile(profileID)
	if err != nil {
		if errors.Is(err, profile.ErrProfileNotFound) {
			http.Error(w, "Profile not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get profile: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Check permissions
	if currentProfile.UserID != userID {
		http.Error(w, "Forbidden: you can only update your own profile", http.StatusForbidden)
		return
	}

	// Parse request body
	var req UpdateMusicProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Field validation
	if len(req.Instruments) == 0 {
		http.Error(w, "At least one instrument is required", http.StatusBadRequest)
		return
	}

	updatedProfile, err := h.profileService.UpdateMusicProfile(
		profileID,
		req.Description,
		req.Genres,
		req.Instruments,
		req.FullName,
		req.Gender,
		req.Age,
		req.CityID,
	)

	if err != nil {
		handleError(w, err)
		return
	}

	profileMedia, err := h.mediaService.GetProfileMedia(updatedProfile.UserID)
	if err != nil {
		log.Printf("Failed to get profile media: %v", err)
	}

	// Convert to API response model
	response := ToMusicProfileResponse(updatedProfile, profileMedia)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
// @Router       /api/profiles/user/{user_id} [get]
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
