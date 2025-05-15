package profile

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Repository errors
var (
	ErrUserNotExists    = errors.New("user does not exist")
	ErrProfileExists    = errors.New("profile already exists")
	ErrProfileNotExists = errors.New("profile does not exist")
	ErrInvalidGoal      = errors.New("invalid improv goal")
	ErrInvalidStyle     = errors.New("invalid improv style")
	ErrInvalidGender    = errors.New("invalid gender")
	ErrInvalidCity      = errors.New("invalid city")
	ErrInvalidMediaRole = errors.New("invalid media role")
)

var (
	roleVideo  = "video"
	roleAvatar = "avatar"
)

// ProfileModel represents the profile data
type ProfileModel struct {
	UserID         int
	FullName       string
	Birthday       time.Time
	Gender         string
	CityID         int
	Bio            string
	Goal           string
	LookingForTeam bool
	CreatedAt      time.Time
	Avatar         *int
	Videos         []int
}

// UpdateProfileModel represents the updated profile data
type UpdateProfileModel struct {
	UserID         int
	FullName       *string
	Birthday       *time.Time
	Gender         *string
	CityID         *int
	Bio            *string
	Goal           *string
	LookingForTeam *bool
	Avatar         *int
	Videos         []int
}

// TranslatedItem represents a catalog item with translations
type TranslatedItem struct {
	Code        string
	Label       string
	Description string
}

// PostgresRepository implements Repository interface
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// BeginTx starts a new transaction
func (r *PostgresRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// CheckUserExists checks if a user exists
func (r *PostgresRepository) CheckUserExists(userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	return exists, err
}

// CheckProfileExists checks if a profile exists for a user
func (r *PostgresRepository) CheckProfileExists(userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)", userID).Scan(&exists)
	return exists, err
}

// CreateProfile creates a new profile
func (r *PostgresRepository) CreateProfile(tx *sql.Tx, profile *ProfileModel) (time.Time, error) {
	var createdAt time.Time

	err := tx.QueryRow(`
        INSERT INTO profiles (
            user_id, full_name, birthday, gender, city_id, 
            bio, goal, looking_for_team
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
        RETURNING created_at
    `, profile.UserID, profile.FullName, profile.Birthday, profile.Gender,
		profile.CityID, profile.Bio, profile.Goal, profile.LookingForTeam).Scan(&profile.CreatedAt)

	return createdAt, err
}

// AddImprovStyles adds improv styles to a profile
func (r *PostgresRepository) AddImprovStyles(tx *sql.Tx, userID int, styles []string) error {
	for _, style := range styles {
		_, err := tx.Exec(`
            INSERT INTO improv_profile_styles (user_id, style)
            VALUES ($1, $2)
        `, userID, style)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetProfile retrieves a profile by user ID
func (r *PostgresRepository) GetProfile(userID int) (*ProfileModel, error) {
	profile := &ProfileModel{}
	err := r.db.QueryRow(`
        SELECT user_id, full_name, birthday, gender, city_id, 
               bio, goal, looking_for_team, created_at 
        FROM profiles WHERE user_id = $1
    `, userID).Scan(
		&profile.UserID, &profile.FullName, &profile.Birthday,
		&profile.Gender, &profile.CityID, &profile.Bio,
		&profile.Goal, &profile.LookingForTeam, &profile.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotExists
		}
		return nil, err
	}

	// Get avatar
	avatar, err := r.GetProfileAvatar(userID)
	if err == nil && avatar != nil {
		profile.Avatar = avatar
	}

	// Get videos
	videos, err := r.GetProfileVideos(userID)
	if err == nil {
		profile.Videos = videos
	}

	return profile, nil
}

// GetProfileByUserID is now redundant since GetProfile does the same thing
// but kept for backward compatibility
func (r *PostgresRepository) GetProfileByUserID(userID int) (*ProfileModel, error) {
	return r.GetProfile(userID)
}

// GetProfileAvatar retrieves the avatar for a profile
func (r *PostgresRepository) GetProfileAvatar(userID int) (*int, error) {
	var mediaID int
	err := r.db.QueryRow(`
        SELECT media_id FROM profile_media 
        WHERE user_id = $1 AND role = 'avatar'
        LIMIT 1
    `, userID).Scan(&mediaID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No avatar, not an error
		}
		return nil, err
	}
	return &mediaID, nil
}

// GetProfileVideos retrieves videos for a profile
func (r *PostgresRepository) GetProfileVideos(userID int) ([]int, error) {
	rows, err := r.db.Query(`
        SELECT media_id FROM profile_media 
        WHERE user_id = $1 AND role = 'video'
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []int
	for rows.Next() {
		var mediaID int
		if err := rows.Scan(&mediaID); err != nil {
			return nil, err
		}
		videos = append(videos, mediaID)
	}
	return videos, rows.Err()
}

// AddProfileMedia adds media to a profile with the specified role
func (r *PostgresRepository) SetProfileVideos(tx *sql.Tx, userID int, videos []int) error {
	// Remove existing videos
	r.RemoveProfileMediaByRole(tx, userID, roleVideo)
	// Add new videos
	for _, videoID := range videos {
		err := r.addProfileMedia(tx, userID, videoID, "video")
		if err != nil {
			return err
		}
	}
	return nil
}

// addProfileMedia adds media to a profile with the specified role
func (r *PostgresRepository) addProfileMedia(tx *sql.Tx, userID int, mediaID int, role string) error {
	_, err := tx.Exec(`
        INSERT INTO profile_media (user_id, media_id, role)
        VALUES ($1, $2, $3)
    `, userID, mediaID, role)
	return err
}

func (r *PostgresRepository) RemoveProfileMediaByRole(tx *sql.Tx, userID int, mediaRole string) error {
	_, err := tx.Exec(`
		DELETE FROM profile_media 
		WHERE user_id = $1 AND role = $2
	`, userID, mediaRole)
	return err
}

func (r *PostgresRepository) RemoveAvatar(tx *sql.Tx, userID int) error {
	// Remove existing avatar(s)
	return r.RemoveProfileMediaByRole(tx, userID, roleAvatar)
}

// SetProfileAvatar sets the avatar for a profile
// It removes any existing avatar and adds the new one
func (r *PostgresRepository) SetProfileAvatar(tx *sql.Tx, userID int, mediaID int) error {
	// Remove existing avatar(s)
	err := r.RemoveAvatar(tx, userID)
	if err != nil {
		return err
	}
	// Add new avatar
	return r.addProfileMedia(tx, userID, mediaID, "avatar")
}

// RemoveProfileMedia removes specific media from a profile
func (r *PostgresRepository) RemoveProfileMedia(tx *sql.Tx, userID int, mediaID int) error {
	_, err := tx.Exec(`
        DELETE FROM profile_media 
        WHERE user_id = $1 AND media_id = $2
    `, userID, mediaID)
	return err
}

// ValidateMediaRole checks if a media role is valid
func (r *PostgresRepository) ValidateMediaRole(role string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM media_role_catalog WHERE role = $1)", role).Scan(&exists)
	return exists, err
}

// GetImprovStyles retrieves improv styles for a profile
func (r *PostgresRepository) GetImprovStyles(userID int) ([]string, error) {
	rows, err := r.db.Query(`
        SELECT style FROM improv_profile_styles WHERE user_id = $1
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var styles []string
	for rows.Next() {
		var style string
		if err = rows.Scan(&style); err != nil {
			return nil, err
		}
		styles = append(styles, style)
	}
	return styles, rows.Err()
}

// UpdateProfile updates a profile, only changing fields that are not nil in the update model
func (r *PostgresRepository) UpdateProfile(tx *sql.Tx, profile *UpdateProfileModel) error {
	// Start with base query
	query := "UPDATE profiles SET "

	// Track parameters and their positions
	params := []interface{}{}
	paramCount := 0
	paramPositions := make([]string, 0)

	// For each field, check if it's not nil and add it to the query
	if profile.FullName != nil {
		paramCount++
		params = append(params, *profile.FullName)
		paramPositions = append(paramPositions, fmt.Sprintf("full_name = $%d", paramCount))
	}

	if profile.Birthday != nil {
		paramCount++
		params = append(params, *profile.Birthday)
		paramPositions = append(paramPositions, fmt.Sprintf("birthday = $%d", paramCount))
	}

	if profile.Gender != nil {
		paramCount++
		params = append(params, *profile.Gender)
		paramPositions = append(paramPositions, fmt.Sprintf("gender = $%d", paramCount))
	}

	if profile.CityID != nil {
		paramCount++
		params = append(params, *profile.CityID)
		paramPositions = append(paramPositions, fmt.Sprintf("city_id = $%d", paramCount))
	}

	if profile.Bio != nil {
		paramCount++
		params = append(params, *profile.Bio)
		paramPositions = append(paramPositions, fmt.Sprintf("bio = $%d", paramCount))
	}

	if profile.Goal != nil {
		paramCount++
		params = append(params, *profile.Goal)
		paramPositions = append(paramPositions, fmt.Sprintf("goal = $%d", paramCount))
	}

	if profile.LookingForTeam != nil {
		paramCount++
		params = append(params, *profile.LookingForTeam)
		paramPositions = append(paramPositions, fmt.Sprintf("looking_for_team = $%d", paramCount))
	}

	// If no parameters were provided, return without executing query
	if len(params) == 0 {
		return nil
	}

	// Add all parameters to the query
	query += strings.Join(paramPositions, ", ")

	// Add the WHERE clause with the user_id
	paramCount++
	query += fmt.Sprintf(" WHERE user_id = $%d", paramCount)
	params = append(params, profile.UserID)

	// Execute the query
	_, err := tx.Exec(query, params...)
	return err
}

// ClearImprovStyles removes all styles from a profile
func (r *PostgresRepository) ClearImprovStyles(tx *sql.Tx, userID int) error {
	_, err := tx.Exec(`DELETE FROM improv_profile_styles WHERE user_id = $1`, userID)
	return err
}

// ClearProfileMedia removes all media from a profile or all media of a specific role
func (r *PostgresRepository) ClearProfileMedia(tx *sql.Tx, userID int, role string) error {
	var err error
	if role == "" {
		_, err = tx.Exec(`DELETE FROM profile_media WHERE user_id = $1`, userID)
	} else {
		_, err = tx.Exec(`DELETE FROM profile_media WHERE user_id = $1 AND role = $2`, userID, role)
	}
	return err
}

// ValidateImprovGoal checks if an improv goal is valid
func (r *PostgresRepository) ValidateImprovGoal(goal string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_goals_catalog WHERE goal_id = $1)", goal).Scan(&exists)
	return exists, err
}

// ValidateImprovStyle checks if an improv style is valid
func (r *PostgresRepository) ValidateImprovStyle(style string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_style_catalog WHERE style_code = $1)", style).Scan(&exists)
	return exists, err
}

// ValidateGender checks if a gender code is valid
func (r *PostgresRepository) ValidateGender(gender string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM gender_catalog WHERE gender_code = $1)", gender).Scan(&exists)
	return exists, err
}

// ValidateCity checks if a city ID is valid
func (r *PostgresRepository) ValidateCity(cityID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM cities WHERE city_id = $1)", cityID).Scan(&exists)
	return exists, err
}

// GetImprovStylesCatalog retrieves improv styles catalog
func (r *PostgresRepository) GetImprovStylesCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT isc.style_code, ist.label
        FROM improv_style_catalog isc
        LEFT JOIN improv_style_translation ist ON isc.style_code = ist.style_code AND ist.lang = $1
    `, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TranslatedItem
	for rows.Next() {
		var item TranslatedItem
		if err := rows.Scan(&item.Code, &item.Label); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetImprovGoalsCatalog retrieves improv goals catalog
func (r *PostgresRepository) GetImprovGoalsCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT igc.goal_id, igt.label
        FROM improv_goals_catalog igc
        LEFT JOIN improv_goals_translation igt ON igc.goal_id = igt.goal_id AND igt.lang = $1
    `, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TranslatedItem
	for rows.Next() {
		var item TranslatedItem
		if err := rows.Scan(&item.Code, &item.Label); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetGendersCatalog retrieves genders catalog
func (r *PostgresRepository) GetGendersCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT gc.gender_code, gct.label
        FROM gender_catalog gc
        LEFT JOIN gender_catalog_translation gct ON gc.gender_code = gct.gender_code AND gct.lang = $1
    `, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TranslatedItem
	for rows.Next() {
		var item TranslatedItem
		if err := rows.Scan(&item.Code, &item.Label); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetCities retrieves available cities
func (r *PostgresRepository) GetCities() ([]struct {
	ID   int
	Name string
}, error) {
	rows, err := r.db.Query(`SELECT city_id, name FROM cities ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []struct {
		ID   int
		Name string
	}

	for rows.Next() {
		var city struct {
			ID   int
			Name string
		}
		if err := rows.Scan(&city.ID, &city.Name); err != nil {
			return nil, err
		}
		cities = append(cities, city)
	}

	return cities, rows.Err()
}

// SearchProfiles searches for profiles and sorts them based on matching improv styles
func (r *PostgresRepository) SearchProfiles(
	currentUserID int,
	fullName *string,
	lookingForTeam *bool,
	goals []string,
	improvStyles []string,
	birthDateMin *time.Time,
	birthDateMax *time.Time,
	genders []string,
	cityID *int,
	hasAvatar *bool,
	hasVideo *bool,
	createdAfter *time.Time,
	page int,
	pageSize int,
) ([]*ProfileModel, int, error) {
	// Start building the query
	baseQuery := `
        WITH current_user_styles AS (
            SELECT style FROM improv_profile_styles WHERE user_id = $1
        ),
        profile_matches AS (
            SELECT 
                p.user_id, 
                p.full_name, 
                p.birthday, 
                p.gender, 
                p.city_id, 
                p.bio, 
                p.goal, 
                p.looking_for_team, 
                p.created_at,
                (
                    SELECT COUNT(*) 
                    FROM improv_profile_styles ips
                    JOIN current_user_styles cus ON ips.style = cus.style
                    WHERE ips.user_id = p.user_id
                ) AS style_match_count
            FROM profiles p
    `

	countQuery := `
        WITH current_user_styles AS (
            SELECT style FROM improv_profile_styles WHERE user_id = $1
        ),
        profile_matches AS (
            SELECT 
                p.user_id,
                (
                    SELECT COUNT(*) 
                    FROM improv_profile_styles ips
                    JOIN current_user_styles cus ON ips.style = cus.style
                    WHERE ips.user_id = p.user_id
                ) AS style_match_count
            FROM profiles p
    `

	// Add joins if needed
	joins := []string{}

	// For improv styles filter - we need a more complex join for the ALL condition
	if len(improvStyles) > 0 {
		// Join once for each style to ensure ALL styles are present (AND logic)
		for i := range improvStyles {
			alias := fmt.Sprintf("ips%d", i)
			joins = append(joins, fmt.Sprintf("JOIN improv_profile_styles %s ON p.user_id = %s.user_id", alias, alias))
		}
	}

	// For has_avatar filter
	if hasAvatar != nil {
		if *hasAvatar {
			joins = append(joins, "JOIN profile_media pa ON p.user_id = pa.user_id AND pa.role = 'avatar'")
		} else {
			joins = append(joins, "LEFT JOIN profile_media pa ON p.user_id = pa.user_id AND pa.role = 'avatar'")
		}
	}

	// For has_video filter
	if hasVideo != nil {
		if *hasVideo {
			joins = append(joins, "JOIN profile_media pv ON p.user_id = pv.user_id AND pv.role = 'video'")
		} else {
			joins = append(joins, "LEFT JOIN profile_media pv ON p.user_id = pv.user_id AND pv.role = 'video'")
		}
	}

	// Add all joins to the queries
	for _, join := range joins {
		baseQuery += " " + join
		countQuery += " " + join
	}

	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{currentUserID} // First argument is current user ID
	argIndex := 2

	// Exclude current user from results
	conditions = append(conditions, "p.user_id <> $1")

	// Full name search (using ILIKE for case-insensitive search)
	if fullName != nil && *fullName != "" {
		conditions = append(conditions, fmt.Sprintf("p.full_name ILIKE $%d", argIndex))
		args = append(args, "%"+*fullName+"%")
		argIndex++
	}

	// Looking for team filter
	if lookingForTeam != nil {
		conditions = append(conditions, fmt.Sprintf("p.looking_for_team = $%d", argIndex))
		args = append(args, *lookingForTeam)
		argIndex++
	}

	// Goals filter - ANY of the specified values (OR logic)
	if len(goals) > 0 {
		placeholders := make([]string, len(goals))
		for i := range goals {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, goals[i])
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("p.goal IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Improv styles filter - ALL of the specified values (AND logic)
	// We already joined the table multiple times, now add the WHERE conditions
	if len(improvStyles) > 0 {
		for i, style := range improvStyles {
			alias := fmt.Sprintf("ips%d", i)
			conditions = append(conditions, fmt.Sprintf("%s.style = $%d", alias, argIndex))
			args = append(args, style)
			argIndex++
		}
	}

	// Age range filter (converted to birthday range)
	if birthDateMin != nil {
		conditions = append(conditions, fmt.Sprintf("p.birthday >= $%d", argIndex))
		args = append(args, *birthDateMin)
		argIndex++
	}
	if birthDateMax != nil {
		conditions = append(conditions, fmt.Sprintf("p.birthday <= $%d", argIndex))
		args = append(args, *birthDateMax)
		argIndex++
	}

	// Genders filter - ANY of the specified values (OR logic)
	if len(genders) > 0 {
		placeholders := make([]string, len(genders))
		for i := range genders {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, genders[i])
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("p.gender IN (%s)", strings.Join(placeholders, ", ")))
	}

	// City filter
	if cityID != nil {
		conditions = append(conditions, fmt.Sprintf("p.city_id = $%d", argIndex))
		args = append(args, *cityID)
		argIndex++
	}

	// Has avatar filter (if NOT included in joins)
	if hasAvatar != nil && !*hasAvatar {
		conditions = append(conditions, "pa.media_id IS NULL")
	}

	// Has video filter (if NOT included in joins)
	if hasVideo != nil && !*hasVideo {
		conditions = append(conditions, "pv.media_id IS NULL")
	}

	// Add createdAfter condition to the WHERE clause if provided
	if createdAfter != nil {
		conditions = append(conditions, fmt.Sprintf("p.created_at >= $%d", argIndex))
		args = append(args, *createdAfter)
		argIndex++
	}

	// Add WHERE clause if there are conditions
	if len(conditions) > 0 {
		whereClause := " WHERE " + strings.Join(conditions, " AND ")
		baseQuery += whereClause
		countQuery += whereClause
	}

	// Close the CTE and add ORDER BY for style matches
	baseQuery += `) SELECT * FROM profile_matches ORDER BY style_match_count DESC, created_at DESC`
	countQuery += `) SELECT COUNT(*) FROM profile_matches`

	// Get total count
	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination to the final query
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, pageSize, (page-1)*pageSize)

	// Execute the query
	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// Parse results
	profiles := []*ProfileModel{}
	for rows.Next() {
		profile := &ProfileModel{}
		var styleMatchCount int
		if err := rows.Scan(
			&profile.UserID, &profile.FullName, &profile.Birthday,
			&profile.Gender, &profile.CityID, &profile.Bio,
			&profile.Goal, &profile.LookingForTeam, &profile.CreatedAt,
			&styleMatchCount,
		); err != nil {
			return nil, 0, err
		}

		// Get avatar
		avatar, err := r.GetProfileAvatar(profile.UserID)
		if err == nil && avatar != nil {
			profile.Avatar = avatar
		}

		// Get videos
		videos, err := r.GetProfileVideos(profile.UserID)
		if err == nil {
			profile.Videos = videos
		}

		profiles = append(profiles, profile)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return profiles, totalCount, nil
}
