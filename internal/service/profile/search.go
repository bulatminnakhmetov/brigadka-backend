package profile

import (
	"log"
	"time"
)

// SearchFilter defines the filters for profile searches
type SearchFilter struct {
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

// SearchResult represents the search results including pagination details
type SearchResult struct {
	Profiles   []Profile `json:"profiles"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// Search searches for profiles based on the provided filters
func (s *ProfileServiceImpl) Search(filter SearchFilter) (*SearchResult, error) {
	// Set defaults if not provided
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 10
	} else if filter.PageSize > 100 {
		filter.PageSize = 100 // Maximum page size to prevent excessive queries
	}

	// Convert age min/max to birthday range if provided
	var birthDateMax, birthDateMin *time.Time
	if filter.AgeMin != nil {
		date := time.Now().AddDate(-*filter.AgeMin, 0, 0)
		birthDateMax = &date
	}
	if filter.AgeMax != nil {
		date := time.Now().AddDate(-*filter.AgeMax-1, 0, 0).AddDate(0, 0, 1) // Add a day to get inclusive range
		birthDateMin = &date
	}

	// Call repository to get results
	results, totalCount, err := s.profileRepo.SearchProfiles(filter.FullName, filter.LookingForTeam,
		filter.Goals, filter.ImprovStyles, birthDateMin, birthDateMax,
		filter.Genders, filter.CityID, filter.HasAvatar, filter.HasVideo,
		filter.CreatedAfter,
		filter.Page, filter.PageSize)
	if err != nil {
		return nil, err
	}

	// Convert repository models to service models
	profiles := make([]Profile, 0, len(results))
	for _, result := range results {
		profile, err := s.ExpandProfile(result)
		if err != nil {
			log.Printf("failed to expand profile: %v", err)
			continue
		}
		profiles = append(profiles, *profile)
	}

	return &SearchResult{
		Profiles:   profiles,
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}, nil
}
