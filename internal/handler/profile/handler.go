package profile

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/profile"
	"github.com/go-chi/chi/v5"
)

// Date is a custom type that handles JSON marshaling and unmarshaling of dates
type Date struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface for Date
func (d *Date) UnmarshalJSON(data []byte) error {
	var dateStr string
	if err := json.Unmarshal(data, &dateStr); err != nil {
		return err
	}

	// Handle empty string case
	if dateStr == "" {
		d.Time = time.Time{}
		return nil
	}

	// Parse the date string
	parsedTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return err
	}

	d.Time = parsedTime
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Date
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return json.Marshal("")
	}
	return json.Marshal(d.Time.Format("2006-01-02"))
}

// ProfileResponse represents profile data for response
type ProfileResponse struct {
	UserID         int             `json:"user_id"`
	FullName       string          `json:"full_name"`
	Birthday       Date            `json:"birthday,omitempty"`
	Gender         string          `json:"gender,omitempty"`
	CityID         int             `json:"city_id,omitempty"`
	Bio            string          `json:"bio,omitempty"`
	Goal           string          `json:"goal,omitempty"`
	LookingForTeam bool            `json:"looking_for_team"`
	ImprovStyles   []string        `json:"improv_styles,omitempty"`
	Avatar         *profile.Media  `json:"avatar,omitempty"`
	Videos         []profile.Media `json:"videos,omitempty"`
	CreatedAt      time.Time       `json:"created_at,omitempty"`
}

// ProfileCreateRequest represents data needed to create a profile
type ProfileCreateRequest struct {
	UserID         int      `json:"user_id" validate:"required"`
	FullName       string   `json:"full_name" validate:"required"`
	Birthday       Date     `json:"birthday" validate:"required"`
	Gender         string   `json:"gender" validate:"required"`
	CityID         int      `json:"city_id" validate:"required"`
	Bio            string   `json:"bio" validate:"required"`
	Goal           string   `json:"goal" validate:"required"`
	ImprovStyles   []string `json:"improv_styles" validate:"required"`
	LookingForTeam bool     `json:"looking_for_team"`
	Avatar         *int     `json:"avatar,omitempty"`
	Videos         []int    `json:"videos,omitempty"`
}

// ProfileUpdateRequest represents data needed to update a profile
type ProfileUpdateRequest struct {
	FullName       *string  `json:"full_name,omitempty"`
	Birthday       *Date    `json:"birthday,omitempty"`
	Gender         *string  `json:"gender,omitempty"`
	CityID         *int     `json:"city_id,omitempty"`
	Bio            *string  `json:"bio,omitempty"`
	Goal           *string  `json:"goal,omitempty"`
	ImprovStyles   []string `json:"improv_styles,omitempty"`
	LookingForTeam *bool    `json:"looking_for_team,omitempty"`
	Avatar         *int     `json:"avatar,omitempty"`
	Videos         []int    `json:"videos,omitempty"`
}

// SearchRequest represents the search query parameters
type SearchRequest struct {
	FullName       *string    `json:"full_name,omitempty"`
	LookingForTeam *bool      `json:"looking_for_team,omitempty"`
	Goals          []string   `json:"goals,omitempty"`
	ImprovStyles   []string   `json:"improv_styles,omitempty"`
	AgeMin         *int       `json:"age_min,omitempty"`
	AgeMax         *int       `json:"age_max,omitempty"`
	Genders        []string   `json:"genders,omitempty"`
	CityID         *int       `json:"city_id,omitempty"`
	HasAvatar      *bool      `json:"has_avatar,omitempty"`
	HasVideo       *bool      `json:"has_video,omitempty"`
	CreatedAfter   *time.Time `json:"created_after,omitempty"`
	Page           int        `json:"page"`
	PageSize       int        `json:"page_size"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Profiles   []ProfileResponse `json:"profiles"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

// TranslatedItem represents a catalog item with translations
// For swagger documentation
type TranslatedItem struct {
	Code        string
	Label       string
	Description string
}

// City represents a city
// For swagger documentation
type City struct {
	ID   int
	Name string
}

// ProfileService defines the interface for profile operations
type ProfileService interface {
	CreateProfile(req profile.ProfileCreateRequest) (*profile.Profile, error)
	GetProfile(userID int) (*profile.Profile, error)
	UpdateProfile(userID int, req profile.ProfileUpdateRequest) (*profile.Profile, error)
	GetImprovStyles(lang string) ([]profile.TranslatedItem, error)
	GetImprovGoals(lang string) ([]profile.TranslatedItem, error)
	GetGenders(lang string) ([]profile.TranslatedItem, error)
	GetCities() ([]profile.City, error)
	Search(userID int, filter profile.SearchFilter) (*profile.SearchResult, error)
}

// ProfileHandler handles requests related to profiles
type ProfileHandler struct {
	profileService ProfileService
}

// NewProfileHandler creates a new instance of ProfileHandler
func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

// handleError handles errors and returns appropriate HTTP status
func handleError(w http.ResponseWriter, err error) {
	// Return different HTTP status codes based on error type
	switch {
	case errors.Is(err, profile.ErrUserNotFound):
		http.Error(w, "User not found", http.StatusNotFound)
	case errors.Is(err, profile.ErrProfileAlreadyExists):
		http.Error(w, "Profile already exists for this user", http.StatusConflict)
	case errors.Is(err, profile.ErrInvalidImprovGoal):
		http.Error(w, "Invalid improv goal", http.StatusBadRequest)
	case errors.Is(err, profile.ErrInvalidImprovStyle):
		http.Error(w, "Invalid improv style", http.StatusBadRequest)
	case errors.Is(err, profile.ErrProfileNotFound):
		http.Error(w, "Profile not found", http.StatusNotFound)
	case errors.Is(err, profile.ErrInvalidGender):
		http.Error(w, "Invalid gender", http.StatusBadRequest)
	case errors.Is(err, profile.ErrInvalidCity):
		http.Error(w, "Invalid city", http.StatusBadRequest)
	default:
		http.Error(w, "Server error: "+err.Error(), http.StatusInternalServerError)
	}
}

func convertToProfileResponse(profile *profile.Profile) ProfileResponse {
	return ProfileResponse{
		UserID:         profile.UserID,
		FullName:       profile.FullName,
		Birthday:       Date{Time: profile.Birthday},
		Gender:         profile.Gender,
		CityID:         profile.CityID,
		Bio:            profile.Bio,
		Goal:           profile.Goal,
		ImprovStyles:   profile.ImprovStyles,
		LookingForTeam: profile.LookingForTeam,
		Avatar:         profile.Avatar,
		Videos:         profile.Videos,
		CreatedAt:      profile.CreatedAt,
	}
}

func convertToCreateProfileRequest(req ProfileCreateRequest) profile.ProfileCreateRequest {
	return profile.ProfileCreateRequest{
		UserID:         req.UserID,
		FullName:       req.FullName,
		Birthday:       req.Birthday.Time,
		Gender:         req.Gender,
		CityID:         req.CityID,
		Bio:            req.Bio,
		Goal:           req.Goal,
		ImprovStyles:   req.ImprovStyles,
		LookingForTeam: req.LookingForTeam,
		Avatar:         req.Avatar,
		Videos:         req.Videos,
	}
}

func convertToUpdateProfileRequest(req ProfileUpdateRequest) profile.ProfileUpdateRequest {
	var birthday *time.Time
	if req.Birthday != nil {
		birthday = &req.Birthday.Time
	}

	return profile.ProfileUpdateRequest{
		FullName:       req.FullName,
		Birthday:       birthday,
		Gender:         req.Gender,
		CityID:         req.CityID,
		Bio:            req.Bio,
		Goal:           req.Goal,
		ImprovStyles:   req.ImprovStyles,
		LookingForTeam: req.LookingForTeam,
		Avatar:         req.Avatar,
		Videos:         req.Videos,
	}
}

// @Summary      Create Profile
// @Description  Creates a new user profile
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        request  body  profile.ProfileCreateRequest  true  "Profile data"
// @Success      201  {object}  profile.Profile
// @Failure      400  {string}  string  "Invalid request body"
// @Failure      404  {string}  string  "User not found"
// @Failure      409  {string}  string  "Profile already exists for this user"
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles [post]
// @Security     BearerAuth
func (h *ProfileHandler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var req ProfileCreateRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call the service to create the profile
	createdProfile, err := h.profileService.CreateProfile(convertToCreateProfileRequest(req))
	if err != nil {
		handleError(w, err)
		return
	}

	response := convertToProfileResponse(createdProfile)

	// Return the created profile
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Update Profile
// @Description  Updates an existing user profile (partial update)
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        request  body  profile.ProfileUpdateRequest  true  "Profile update data"
// @Success      200  {object}  profile.Profile
// @Failure      400  {string}  string  "Invalid request body"
// @Failure      401  {string}  string  "Unauthorized"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles [patch]
// @Security     BearerAuth
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var updateReq ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call the service to update the profile
	prof, err := h.profileService.UpdateProfile(userID, convertToUpdateProfileRequest(updateReq))

	if err != nil {
		handleError(w, err)
		return
	}

	response := convertToProfileResponse(prof)

	// Return the updated profile
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Profile
// @Description  Retrieves a user profile by ID
// @Tags         profile
// @Produce      json
// @Param        userID  path  int  true  "User ID"
// @Success      200  {object}  ProfileResponse
// @Failure      400  {string}  string  "Invalid user ID"
// @Failure      404  {string}  string  "Profile not found"
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles/{userID} [get]
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Extract userID from URL path using Chi router
	userIDStr := chi.URLParam(r, "userID")
	if userIDStr == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Call the service to get the profile
	prof, err := h.profileService.GetProfile(userID)
	if err != nil {
		handleError(w, err)
		return
	}

	response := convertToProfileResponse(prof)

	// Return the profile
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Improv Styles
// @Description  Retrieves a catalog of improv styles with translations
// @Tags         catalog
// @Produce      json
// @Param        lang  query  string  false  "Language code (default: en)"
// @Success      200  {array}  profile.TranslatedItem
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles/catalog/improv-styles [get]
func (h *ProfileHandler) GetImprovStyles(w http.ResponseWriter, r *http.Request) {
	// Get language from query parameter or use default
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "ru" // Default language
	}

	// Call the service to get the styles
	styles, err := h.profileService.GetImprovStyles(lang)
	if err != nil {
		handleError(w, err)
		return
	}

	// Return the styles
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(styles); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Improv Goals
// @Description  Retrieves a catalog of improv goals with translations
// @Tags         catalog
// @Produce      json
// @Param        lang  query  string  false  "Language code (default: en)"
// @Success      200  {array}  profile.TranslatedItem
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles/catalog/improv-goals [get]
func (h *ProfileHandler) GetImprovGoals(w http.ResponseWriter, r *http.Request) {
	// Get language from query parameter or use default
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "ru" // Default language
	}

	// Call the service to get the goals
	goals, err := h.profileService.GetImprovGoals(lang)
	if err != nil {
		handleError(w, err)
		return
	}

	// Return the goals
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(goals); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Genders
// @Description  Retrieves a catalog of genders with translations
// @Tags         catalog
// @Produce      json
// @Param        lang  query  string  false  "Language code (default: en)"
// @Success      200  {array}  profile.TranslatedItem
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles/catalog/genders [get]
func (h *ProfileHandler) GetGenders(w http.ResponseWriter, r *http.Request) {
	// Get language from query parameter or use default
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "ru" // Default language
	}

	// Call the service to get the genders
	genders, err := h.profileService.GetGenders(lang)
	if err != nil {
		handleError(w, err)
		return
	}

	// Return the genders
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(genders); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Get Cities
// @Description  Retrieves a list of available cities
// @Tags         catalog
// @Produce      json
// @Success      200  {array}  profile.City
// @Failure      500  {string}  string  "Server error"
// @Router       /profiles/catalog/cities [get]
func (h *ProfileHandler) GetCities(w http.ResponseWriter, r *http.Request) {
	// Call the service to get the cities
	cities, err := h.profileService.GetCities()
	if err != nil {
		handleError(w, err)
		return
	}

	// Return the cities
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cities); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// @Summary      Search Profiles
// @Description  Search for profiles with various filters
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        request  body      SearchRequest  true  "Search filters"
// @Success      200      {object}  SearchResponse
// @Failure      400      {string}  string  "Invalid request"
// @Failure      500      {string}  string  "Server error"
// @Router       /profiles/search [post]
func (h *ProfileHandler) SearchProfiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SearchRequest

	// Parse the request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Convert request to service filter
	filter := profile.SearchFilter{
		FullName:       req.FullName,
		LookingForTeam: req.LookingForTeam,
		Goals:          req.Goals,
		ImprovStyles:   req.ImprovStyles,
		AgeMin:         req.AgeMin,
		AgeMax:         req.AgeMax,
		Genders:        req.Genders,
		CityID:         req.CityID,
		HasAvatar:      req.HasAvatar,
		HasVideo:       req.HasVideo,
		CreatedAfter:   req.CreatedAfter,
		Page:           req.Page,
		PageSize:       req.PageSize,
	}

	// Call the service to perform the search
	result, err := h.profileService.Search(userID, filter)
	if err != nil {
		handleError(w, err)
		return
	}

	// Convert service profiles to response profiles
	profiles := make([]ProfileResponse, 0, len(result.Profiles))
	for _, p := range result.Profiles {
		profiles = append(profiles, convertToProfileResponse(&p))
	}

	// Create the response
	response := SearchResponse{
		Profiles:   profiles,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		PageSize:   result.PageSize,
	}

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
