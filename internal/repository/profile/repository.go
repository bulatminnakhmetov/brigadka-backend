package profile

import (
	"database/sql"
	"errors"
	"time"
)

// Repository errors
var (
	ErrUserNotExists     = errors.New("user does not exist")
	ErrProfileExists     = errors.New("profile already exists")
	ErrProfileNotExists  = errors.New("profile does not exist")
	ErrInvalidGoal       = errors.New("invalid improv goal")
	ErrInvalidStyle      = errors.New("invalid improv style")
	ErrInvalidGenre      = errors.New("invalid music genre")
	ErrInvalidInstrument = errors.New("invalid music instrument")
)

// ProfileModel represents the base profile data
type ProfileModel struct {
	ID           int
	UserID       int
	Description  string
	ActivityType string
	CreatedAt    time.Time
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
func (r *PostgresRepository) CheckProfileExists(userID int, activityType string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1 AND activity_type = $2)",
		userID, activityType).Scan(&exists)
	return exists, err
}

// CreateBaseProfile creates a base profile record
func (r *PostgresRepository) CreateBaseProfile(tx *sql.Tx, userID int, description, activityType string) (int, time.Time, error) {
	var profileID int
	var createdAt time.Time
	err := tx.QueryRow(`
        INSERT INTO profiles (user_id, description, activity_type) 
        VALUES ($1, $2, $3) 
        RETURNING id, created_at
    `, userID, description, activityType).Scan(&profileID, &createdAt)
	return profileID, createdAt, err
}

// CreateImprovProfile creates an improv profile
func (r *PostgresRepository) CreateImprovProfile(tx *sql.Tx, profileID int, goal string, lookingForTeam bool) error {
	_, err := tx.Exec(`
        INSERT INTO improv_profiles (profile_id, goal, looking_for_team)
        VALUES ($1, $2, $3)
    `, profileID, goal, lookingForTeam)
	return err
}

// AddImprovStyles adds improv styles to a profile
func (r *PostgresRepository) AddImprovStyles(tx *sql.Tx, profileID int, styles []string) error {
	for _, style := range styles {
		_, err := tx.Exec(`
            INSERT INTO improv_profile_styles (profile_id, style)
            VALUES ($1, $2)
        `, profileID, style)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddMusicGenres adds music genres to a profile
func (r *PostgresRepository) AddMusicGenres(tx *sql.Tx, profileID int, genres []string) error {
	for _, genre := range genres {
		_, err := tx.Exec(`
            INSERT INTO music_profile_genres (profile_id, genre_code)
            VALUES ($1, $2)
        `, profileID, genre)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddMusicInstruments adds instruments to a music profile
func (r *PostgresRepository) AddMusicInstruments(tx *sql.Tx, profileID int, instruments []string) error {
	for _, instrument := range instruments {
		_, err := tx.Exec(`
            INSERT INTO music_profile_instruments (profile_id, instrument_code)
            VALUES ($1, $2)
        `, profileID, instrument)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetProfile retrieves a profile by ID
func (r *PostgresRepository) GetProfile(profileID int) (*ProfileModel, error) {
	profile := &ProfileModel{}
	err := r.db.QueryRow(`
        SELECT id, user_id, description, activity_type, created_at 
        FROM profiles WHERE id = $1
    `, profileID).Scan(&profile.ID, &profile.UserID, &profile.Description,
		&profile.ActivityType, &profile.CreatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotExists
		}
		return nil, err
	}
	return profile, nil
}

// GetImprovProfileDetails retrieves details for improv profile
func (r *PostgresRepository) GetImprovProfileDetails(profileID int) (string, bool, error) {
	var goal string
	var lookingForTeam bool
	err := r.db.QueryRow(`
        SELECT goal, looking_for_team FROM improv_profiles WHERE profile_id = $1
    `, profileID).Scan(&goal, &lookingForTeam)
	return goal, lookingForTeam, err
}

// GetImprovStyles retrieves improv styles for a profile
func (r *PostgresRepository) GetImprovStyles(profileID int) ([]string, error) {
	rows, err := r.db.Query(`
        SELECT style FROM improv_profile_styles WHERE profile_id = $1
    `, profileID)
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

// GetMusicGenres retrieves music genres for a profile
func (r *PostgresRepository) GetMusicGenres(profileID int) ([]string, error) {
	rows, err := r.db.Query(`
        SELECT genre_code FROM music_profile_genres WHERE profile_id = $1
    `, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var genres []string
	for rows.Next() {
		var genre string
		if err = rows.Scan(&genre); err != nil {
			return nil, err
		}
		genres = append(genres, genre)
	}
	return genres, rows.Err()
}

// GetMusicInstruments retrieves instruments for a music profile
func (r *PostgresRepository) GetMusicInstruments(profileID int) ([]string, error) {
	rows, err := r.db.Query(`
        SELECT instrument_code FROM music_profile_instruments WHERE profile_id = $1
    `, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []string
	for rows.Next() {
		var instrument string
		if err = rows.Scan(&instrument); err != nil {
			return nil, err
		}
		instruments = append(instruments, instrument)
	}
	return instruments, rows.Err()
}

// UpdateProfileDescription updates a profile's description
func (r *PostgresRepository) UpdateProfileDescription(tx *sql.Tx, profileID int, description string) error {
	_, err := tx.Exec(`
        UPDATE profiles SET description = $1 WHERE id = $2
    `, description, profileID)
	return err
}

// UpdateImprovProfile updates improv profile details
func (r *PostgresRepository) UpdateImprovProfile(tx *sql.Tx, profileID int, goal string, lookingForTeam bool) error {
	_, err := tx.Exec(`
        UPDATE improv_profiles SET goal = $1, looking_for_team = $2 WHERE profile_id = $3
    `, goal, lookingForTeam, profileID)
	return err
}

// ClearImprovStyles removes all styles from an improv profile
func (r *PostgresRepository) ClearImprovStyles(tx *sql.Tx, profileID int) error {
	_, err := tx.Exec(`DELETE FROM improv_profile_styles WHERE profile_id = $1`, profileID)
	return err
}

// ClearMusicGenres removes all genres from a music profile
func (r *PostgresRepository) ClearMusicGenres(tx *sql.Tx, profileID int) error {
	_, err := tx.Exec(`DELETE FROM music_profile_genres WHERE profile_id = $1`, profileID)
	return err
}

// ClearMusicInstruments removes all instruments from a music profile
func (r *PostgresRepository) ClearMusicInstruments(tx *sql.Tx, profileID int) error {
	_, err := tx.Exec(`DELETE FROM music_profile_instruments WHERE profile_id = $1`, profileID)
	return err
}

// ValidateImprovGoal checks if an improv goal is valid
func (r *PostgresRepository) ValidateImprovGoal(goal string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_goals_catalog WHERE goal_code = $1)", goal).Scan(&exists)
	return exists, err
}

// ValidateImprovStyle checks if an improv style is valid
func (r *PostgresRepository) ValidateImprovStyle(style string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_style_catalog WHERE style_code = $1)", style).Scan(&exists)
	return exists, err
}

// ValidateMusicGenre checks if a music genre is valid
func (r *PostgresRepository) ValidateMusicGenre(genre string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM music_genre_catalog WHERE genre_code = $1)", genre).Scan(&exists)
	return exists, err
}

// ValidateMusicInstrument checks if an instrument is valid
func (r *PostgresRepository) ValidateMusicInstrument(instrument string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM music_instrument_catalog WHERE instrument_code = $1)", instrument).Scan(&exists)
	return exists, err
}

// GetUserProfiles retrieves all profiles for a user
func (r *PostgresRepository) GetUserProfiles(userID int) (map[string]int, error) {
	rows, err := r.db.Query(`
        SELECT id, activity_type 
        FROM profiles 
        WHERE user_id = $1
    `, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	profiles := make(map[string]int)
	for rows.Next() {
		var profileID int
		var activityType string
		if err := rows.Scan(&profileID, &activityType); err != nil {
			return nil, err
		}
		profiles[activityType] = profileID
	}
	return profiles, rows.Err()
}

// GetActivityTypes retrieves activity types catalog
func (r *PostgresRepository) GetActivityTypesCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT activity_type, description 
        FROM activity_type_catalog
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TranslatedItem
	for rows.Next() {
		var item TranslatedItem
		if err := rows.Scan(&item.Code, &item.Description); err != nil {
			return nil, err
		}
		// Use code as label since activity type has no translations
		item.Label = item.Code
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetImprovStyles retrieves improv styles catalog
func (r *PostgresRepository) GetImprovStylesCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT isc.style_code, ist.label, ist.description
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
		if err := rows.Scan(&item.Code, &item.Label, &item.Description); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetImprovGoals retrieves improv goals catalog
func (r *PostgresRepository) GetImprovGoalsCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT igc.goal_code, igt.label, igt.description
        FROM improv_goals_catalog igc
        LEFT JOIN improv_goals_translation igt ON igc.goal_code = igt.goal_code AND igt.lang = $1
    `, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TranslatedItem
	for rows.Next() {
		var item TranslatedItem
		if err := rows.Scan(&item.Code, &item.Label, &item.Description); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetMusicGenres retrieves music genres catalog
func (r *PostgresRepository) GetMusicGenresCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT mgc.genre_code, mgt.label
        FROM music_genre_catalog mgc
        LEFT JOIN music_genre_translation mgt ON mgc.genre_code = mgt.genre_code AND mgt.lang = $1
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

// GetMusicInstruments retrieves music instruments catalog
func (r *PostgresRepository) GetMusicInstrumentsCatalog(lang string) ([]TranslatedItem, error) {
	if lang == "" {
		lang = "ru" // Default language
	}

	rows, err := r.db.Query(`
        SELECT mic.instrument_code, mit.label
        FROM music_instrument_catalog mic
        LEFT JOIN music_instrument_translation mit ON mic.instrument_code = mit.instrument_code AND mit.lang = $1
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
