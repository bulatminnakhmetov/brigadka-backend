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

// Search searches for profiles with the given filters and sorts results by improv style matches
func (s *ProfileServiceImpl) Search(userID int, filter SearchFilter) (*SearchResult, error) {
	// Set defaults for pagination
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	// Convert ages to birthdate bounds if provided
	var birthDateMin, birthDateMax *time.Time
	if filter.AgeMin != nil {
		date := time.Now().AddDate(-*filter.AgeMin, 0, 0)
		birthDateMax = &date
	}
	if filter.AgeMax != nil {
		date := time.Now().AddDate(-*filter.AgeMax-1, 0, 0).AddDate(0, 0, 1)
		birthDateMin = &date
	}

	// Call repository to search profiles with style matches
	profiles, totalCount, err := s.profileRepo.SearchProfiles(
		userID,
		filter.FullName,
		filter.LookingForTeam,
		filter.Goals,
		filter.ImprovStyles,
		birthDateMin,
		birthDateMax,
		filter.Genders,
		filter.CityID,
		filter.HasAvatar,
		filter.HasVideo,
		filter.CreatedAfter,
		filter.Page,
		filter.PageSize,
	)

	if err != nil {
		return nil, err
	}

	// Convert repository profiles to service profiles
	result := &SearchResult{
		Profiles:   make([]Profile, 0, len(profiles)),
		TotalCount: totalCount,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
	}

	for _, p := range profiles {
		expanded, err := s.ExpandProfile(p)
		if err != nil {
			log.Printf("Error expanding profile %d: %v", p.UserID, err)
			continue
		}
		result.Profiles = append(result.Profiles, *expanded)
	}

	return result, nil
}
