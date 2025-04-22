package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SearchService defines the interface for search functionality
type SearchService interface {
	SearchProfiles(ctx context.Context, req ProfileSearchRequest) (*ProfileSearchResponse, error)
}

// SearchServiceImpl implements the SearchService interface
type SearchServiceImpl struct {
	db *sql.DB
}

// NewSearchService creates a new search service
func NewSearchService(db *sql.DB) SearchService {
	return &SearchServiceImpl{
		db: db,
	}
}

// SearchProfiles searches for profiles based on the provided criteria
func (s *SearchServiceImpl) SearchProfiles(ctx context.Context, req ProfileSearchRequest) (*ProfileSearchResponse, error) {
	// Build the SQL query based on search parameters
	queryBuilder := newQueryBuilder()

	// Base table joins
	queryBuilder.baseJoin()

	// Apply filters
	// General filters
	if req.FullName != "" {
		queryBuilder.addFilter("u.full_name ILIKE $%d", "%"+req.FullName+"%")
	}

	if req.CityID != nil {
		queryBuilder.addFilter("u.city_id = $%d", *req.CityID)
	}

	if req.Gender != "" {
		queryBuilder.addFilter("u.gender = $%d", req.Gender)
	}

	if req.AgeMin != nil {
		queryBuilder.addFilter("u.age >= $%d", *req.AgeMin)
	}

	if req.AgeMax != nil {
		queryBuilder.addFilter("u.age <= $%d", *req.AgeMax)
	}

	if req.ActivityType != "" {
		queryBuilder.addFilter("p.activity_type = $%d", req.ActivityType)
	}

	// Improv-specific filters
	if len(req.ImprovStyles) > 0 {
		// Join the improv_profile_styles table
		queryBuilder.addJoin("LEFT JOIN improv_profile_styles ips ON p.id = ips.profile_id")

		// Build an IN clause for styles
		placeholders := make([]string, len(req.ImprovStyles))
		for i := range req.ImprovStyles {
			placeholders[i] = fmt.Sprintf("$%d", queryBuilder.nextParamIndex())
			queryBuilder.params = append(queryBuilder.params, req.ImprovStyles[i])
		}

		queryBuilder.addPlainFilter(fmt.Sprintf("p.activity_type = 'improv' AND ips.style IN (%s)", strings.Join(placeholders, ", ")))
	}

	if req.ImprovGoal != "" {
		queryBuilder.addJoin("LEFT JOIN improv_profiles ip ON p.id = ip.profile_id")
		queryBuilder.addFilter("p.activity_type = 'improv' AND ip.goal = $%d", req.ImprovGoal)
	}

	if req.ImprovLookingForTeam != nil {
		queryBuilder.addJoin("LEFT JOIN improv_profiles ip ON p.id = ip.profile_id")
		queryBuilder.addFilter("p.activity_type = 'improv' AND ip.looking_for_team = $%d", *req.ImprovLookingForTeam)
	}

	// Music-specific filters
	if len(req.MusicGenres) > 0 {
		// Join the music_profile_genres table
		queryBuilder.addJoin("LEFT JOIN music_profile_genres mpg ON p.id = mpg.profile_id")

		// Build an IN clause for genres
		placeholders := make([]string, len(req.MusicGenres))
		for i := range req.MusicGenres {
			placeholders[i] = fmt.Sprintf("$%d", queryBuilder.nextParamIndex())
			queryBuilder.params = append(queryBuilder.params, req.MusicGenres[i])
		}

		queryBuilder.addPlainFilter(fmt.Sprintf("p.activity_type = 'music' AND mpg.genre_code IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(req.MusicInstruments) > 0 {
		// Join the music_profile_instruments table
		queryBuilder.addJoin("LEFT JOIN music_profile_instruments mpi ON p.id = mpi.profile_id")

		// Build an IN clause for instruments
		placeholders := make([]string, len(req.MusicInstruments))
		for i := range req.MusicInstruments {
			placeholders[i] = fmt.Sprintf("$%d", queryBuilder.nextParamIndex())
			queryBuilder.params = append(queryBuilder.params, req.MusicInstruments[i])
		}

		queryBuilder.addPlainFilter(fmt.Sprintf("p.activity_type = 'music' AND mpi.instrument_code IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Build the count query to get total results
	countQuery := queryBuilder.buildCountQuery()

	// Execute count query
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, queryBuilder.params...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count profiles: %w", err)
	}

	// Calculate pagination
	pageSize := req.Limit
	totalPages := (totalCount + pageSize - 1) / pageSize // Ceiling division
	currentPage := (req.Offset / pageSize) + 1

	// Add pagination to the query
	queryBuilder.addPagination(req.Limit, req.Offset)

	// Build the final query
	query := queryBuilder.buildQuery()

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, queryBuilder.params...)
	if err != nil {
		return nil, fmt.Errorf("failed to search profiles: %w", err)
	}
	defer rows.Close()

	// Process results
	results := make([]ProfileSearchResult, 0)

	for rows.Next() {
		var result ProfileSearchResult
		var city sql.NullString
		var gender sql.NullString
		var age sql.NullInt64

		// Scan base fields
		err := rows.Scan(
			&result.ProfileID,
			&result.UserID,
			&result.FullName,
			&city,
			&gender,
			&age,
			&result.ActivityType,
			&result.Description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan profile row: %w", err)
		}

		// Set nullable fields
		if city.Valid {
			result.City = city.String
		}
		if gender.Valid {
			result.Gender = gender.String
		}
		if age.Valid {
			ageInt := int(age.Int64)
			result.Age = &ageInt
		}

		// Fetch activity-specific data based on the profile type
		switch result.ActivityType {
		case "improv":
			improvProfile, err := s.getImprovProfileData(ctx, result.ProfileID)
			if err != nil {
				return nil, fmt.Errorf("failed to get improv data for profile %d: %w", result.ProfileID, err)
			}
			result.ImprovGoal = improvProfile.Goal
			result.ImprovStyles = improvProfile.Styles
			result.ImprovLookingForTeam = improvProfile.LookingForTeam

		case "music":
			musicProfile, err := s.getMusicProfileData(ctx, result.ProfileID)
			if err != nil {
				return nil, fmt.Errorf("failed to get music data for profile %d: %w", result.ProfileID, err)
			}
			result.MusicGenres = musicProfile.Genres
			result.MusicInstruments = musicProfile.Instruments
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating profile rows: %w", err)
	}

	return &ProfileSearchResponse{
		Results:     results,
		TotalCount:  totalCount,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		PageSize:    pageSize,
	}, nil
}

// Helper struct to store improv profile data
type improvProfileData struct {
	Goal           string
	Styles         []string
	LookingForTeam *bool
}

// Helper struct to store music profile data
type musicProfileData struct {
	Genres      []string
	Instruments []string
}

// getImprovProfileData fetches improv-specific data for a profile
func (s *SearchServiceImpl) getImprovProfileData(ctx context.Context, profileID int) (*improvProfileData, error) {
	// Get the basic improv profile info
	var goal string
	var lookingForTeam bool

	err := s.db.QueryRowContext(ctx, `
        SELECT goal, looking_for_team 
        FROM improv_profiles 
        WHERE profile_id = $1
    `, profileID).Scan(&goal, &lookingForTeam)

	if err != nil {
		if err == sql.ErrNoRows {
			return &improvProfileData{}, nil
		}
		return nil, err
	}

	// Get the improv styles
	rows, err := s.db.QueryContext(ctx, `
        SELECT style 
        FROM improv_profile_styles 
        WHERE profile_id = $1
    `, profileID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var styles []string
	for rows.Next() {
		var style string
		if err := rows.Scan(&style); err != nil {
			return nil, err
		}
		styles = append(styles, style)
	}

	return &improvProfileData{
		Goal:           goal,
		Styles:         styles,
		LookingForTeam: &lookingForTeam,
	}, nil
}

// getMusicProfileData fetches music-specific data for a profile
func (s *SearchServiceImpl) getMusicProfileData(ctx context.Context, profileID int) (*musicProfileData, error) {
	// Get the music genres
	genreRows, err := s.db.QueryContext(ctx, `
        SELECT genre_code 
        FROM music_profile_genres 
        WHERE profile_id = $1
    `, profileID)

	if err != nil {
		return nil, err
	}
	defer genreRows.Close()

	var genres []string
	for genreRows.Next() {
		var genre string
		if err := genreRows.Scan(&genre); err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}

	// Get the music instruments
	instrumentRows, err := s.db.QueryContext(ctx, `
        SELECT instrument_code 
        FROM music_profile_instruments 
        WHERE profile_id = $1
    `, profileID)

	if err != nil {
		return nil, err
	}
	defer instrumentRows.Close()

	var instruments []string
	for instrumentRows.Next() {
		var instrument string
		if err := instrumentRows.Scan(&instrument); err != nil {
			return nil, err
		}
		instruments = append(instruments, instrument)
	}

	return &musicProfileData{
		Genres:      genres,
		Instruments: instruments,
	}, nil
}

// QueryBuilder helper to build SQL queries
type queryBuilder struct {
	joins      []string
	filters    []string
	params     []interface{}
	paramIndex int
	limit      int
	offset     int
}

// Create a new query builder
func newQueryBuilder() *queryBuilder {
	return &queryBuilder{
		joins:      make([]string, 0),
		filters:    make([]string, 0),
		params:     make([]interface{}, 0),
		paramIndex: 0,
	}
}

// Add the base join needed for all queries
func (qb *queryBuilder) baseJoin() {
	qb.joins = append(qb.joins, `
        FROM profiles p
        JOIN users u ON p.user_id = u.id
        LEFT JOIN cities c ON u.city_id = c.city_id
    `)
}

// Add a join to the query
func (qb *queryBuilder) addJoin(join string) {
	// Check if this join already exists
	for _, existingJoin := range qb.joins {
		if strings.Contains(existingJoin, join) {
			return
		}
	}
	qb.joins = append(qb.joins, join)
}

// Get the next parameter index
func (qb *queryBuilder) nextParamIndex() int {
	qb.paramIndex++
	return qb.paramIndex
}

// Add a filter with a parameter
func (qb *queryBuilder) addFilter(filter string, value interface{}) {
	paramIndex := qb.nextParamIndex()
	qb.filters = append(qb.filters, fmt.Sprintf(filter, paramIndex))
	qb.params = append(qb.params, value)
}

// Add a filter without parameter placeholders (for complex filters)
func (qb *queryBuilder) addPlainFilter(filter string) {
	qb.filters = append(qb.filters, filter)
}

// Add pagination
func (qb *queryBuilder) addPagination(limit, offset int) {
	qb.limit = limit
	qb.offset = offset
}

// Build the complete query
func (qb *queryBuilder) buildQuery() string {
	query := `
        SELECT 
            p.id, 
            u.id, 
            u.full_name, 
            c.name as city, 
            u.gender, 
            u.age, 
            p.activity_type, 
            p.description
    `

	// Add joins
	query += strings.Join(qb.joins, " ")

	// Add WHERE clause if there are filters
	if len(qb.filters) > 0 {
		query += " WHERE " + strings.Join(qb.filters, " AND ")
	}

	// Add ORDER BY
	query += " ORDER BY p.created_at DESC"

	// Add LIMIT and OFFSET
	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", qb.limit, qb.offset)
	}

	return query
}

// Build a query to count total results
func (qb *queryBuilder) buildCountQuery() string {
	query := "SELECT COUNT(DISTINCT p.id) "

	// Add joins
	query += strings.Join(qb.joins, " ")

	// Add WHERE clause if there are filters
	if len(qb.filters) > 0 {
		query += " WHERE " + strings.Join(qb.filters, " AND ")
	}

	return query
}
