package search

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Errors
var (
	ErrInvalidSearchParams = errors.New("invalid search parameters")
)

// ProfileSearchRequest represents a search query for profiles
type ProfileSearchRequest struct {
	// General search parameters
	FullName     string `json:"full_name,omitempty"`     // Full name (partial matching)
	CityID       *int   `json:"city_id,omitempty"`       // City ID
	Gender       string `json:"gender,omitempty"`        // Gender code
	AgeMin       *int   `json:"age_min,omitempty"`       // Minimum age
	AgeMax       *int   `json:"age_max,omitempty"`       // Maximum age
	ActivityType string `json:"activity_type,omitempty"` // Activity type (improv, music)

	// Improv profile parameters
	ImprovGoal           string   `json:"improv_goal,omitempty"`             // Goal code
	ImprovStyles         []string `json:"improv_styles,omitempty"`           // Array of style codes
	ImprovLookingForTeam *bool    `json:"improv_looking_for_team,omitempty"` // Looking for team flag

	// Music profile parameters
	MusicGenres      []string `json:"music_genres,omitempty"`      // Array of genre codes
	MusicInstruments []string `json:"music_instruments,omitempty"` // Array of instrument codes

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Default will be set by service
	Offset int `json:"offset,omitempty"` // Default: 0
}

// ProfileSearchResult represents a single profile in search results
type ProfileSearchResult struct {
	ProfileID    int    `json:"profile_id"`
	UserID       int    `json:"user_id"`
	FullName     string `json:"full_name"`
	City         string `json:"city,omitempty"`
	Gender       string `json:"gender,omitempty"`
	Age          *int   `json:"age,omitempty"`
	ActivityType string `json:"activity_type"`
	Description  string `json:"description"`

	// Improv-specific fields (will be null for music profiles)
	ImprovGoal           string   `json:"improv_goal,omitempty"`
	ImprovStyles         []string `json:"improv_styles,omitempty"`
	ImprovLookingForTeam *bool    `json:"improv_looking_for_team,omitempty"`

	// Music-specific fields (will be null for improv profiles)
	MusicGenres      []string `json:"music_genres,omitempty"`
	MusicInstruments []string `json:"music_instruments,omitempty"`
}

// ProfileSearchResponse represents the search response
type ProfileSearchResponse struct {
	Results     []ProfileSearchResult `json:"results"`
	TotalCount  int                   `json:"total_count"`
	CurrentPage int                   `json:"current_page"`
	TotalPages  int                   `json:"total_pages"`
	PageSize    int                   `json:"page_size"`
}

// SearchHandler handles search functionality
type SearchHandler struct {
	searchService SearchService
}

// NewSearchHandler creates a new SearchHandler
func NewSearchHandler(searchService SearchService) *SearchHandler {
	return &SearchHandler{
		searchService: searchService,
	}
}

// @Summary      Search profiles
// @Description  Search profiles by various criteria
// @Tags         search
// @Accept       json
// @Produce      json
// @Param        request  body      ProfileSearchRequest  false  "Search parameters"
// @Success      200      {object}  ProfileSearchResponse
// @Failure      400      {string}  string  "Invalid request"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/search/profiles [post]
func (h *SearchHandler) SearchProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req ProfileSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults for pagination if not specified
	if req.Limit <= 0 {
		req.Limit = 20 // Default limit
	}

	// Maximum limit to prevent performance issues
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Sanitize inputs
	if req.FullName != "" {
		req.FullName = strings.TrimSpace(req.FullName)
	}

	// Validate age range if provided
	if req.AgeMin != nil && req.AgeMax != nil && *req.AgeMin > *req.AgeMax {
		http.Error(w, "Invalid age range: minimum age cannot be greater than maximum age", http.StatusBadRequest)
		return
	}

	// Execute search
	results, err := h.searchService.SearchProfiles(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSearchParams):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Failed to search profiles: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// @Summary      Search profiles (GET version)
// @Description  Search profiles by query parameters (simplified version)
// @Tags         search
// @Produce      json
// @Param        full_name              query  string  false  "Full name to search"
// @Param        city_id                query  int     false  "City ID"
// @Param        activity_type          query  string  false  "Activity type (improv, music)"
// @Param        improv_looking_for_team query  bool    false  "Looking for team"
// @Param        improv_goal            query  string  false  "Improv goal code"
// @Param        improv_style           query  string  false  "Improv style code (can be used multiple times)"
// @Param        music_genre            query  string  false  "Music genre code (can be used multiple times)"
// @Param        music_instrument       query  string  false  "Music instrument code (can be used multiple times)"
// @Param        limit                  query  int     false  "Results per page (default 20, max 100)"
// @Param        offset                 query  int     false  "Offset for pagination"
// @Success      200      {object}  ProfileSearchResponse
// @Failure      400      {string}  string  "Invalid request"
// @Failure      500      {string}  string  "Internal server error"
// @Router       /api/search/profiles [get]
func (h *SearchHandler) SearchProfilesGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	// Build search request from query parameters
	req := ProfileSearchRequest{
		FullName:     query.Get("full_name"),
		ActivityType: query.Get("activity_type"),
		Offset:       parseIntParam(query.Get("offset"), 0),
		Limit:        parseIntParam(query.Get("limit"), 20),
	}

	// Set city ID if present
	if cityID := query.Get("city_id"); cityID != "" {
		id, err := strconv.Atoi(cityID)
		if err == nil && id > 0 {
			req.CityID = &id
		}
	}

	// Parse age range
	if ageMin := query.Get("age_min"); ageMin != "" {
		age, err := strconv.Atoi(ageMin)
		if err == nil && age > 0 {
			req.AgeMin = &age
		}
	}
	if ageMax := query.Get("age_max"); ageMax != "" {
		age, err := strconv.Atoi(ageMax)
		if err == nil && age > 0 {
			req.AgeMax = &age
		}
	}

	// Parse gender
	if gender := query.Get("gender"); gender != "" {
		req.Gender = gender
	}

	// Parse improv parameters
	if goal := query.Get("improv_goal"); goal != "" {
		req.ImprovGoal = goal
	}

	// Parse improv styles (can have multiple values)
	if styles := query["improv_style"]; len(styles) > 0 {
		req.ImprovStyles = styles
	}

	// Parse looking for team flag
	if lft := query.Get("improv_looking_for_team"); lft != "" {
		lookingForTeam, err := strconv.ParseBool(lft)
		if err == nil {
			req.ImprovLookingForTeam = &lookingForTeam
		}
	}

	// Parse music parameters
	if genres := query["music_genre"]; len(genres) > 0 {
		req.MusicGenres = genres
	}

	if instruments := query["music_instrument"]; len(instruments) > 0 {
		req.MusicInstruments = instruments
	}

	// Maximum limit to prevent performance issues
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Execute search
	results, err := h.searchService.SearchProfiles(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSearchParams):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Failed to search profiles: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Helper function to parse integer parameters with default value
func parseIntParam(param string, defaultValue int) int {
	if param == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(param)
	if err != nil || value < 0 {
		return defaultValue
	}

	return value
}
