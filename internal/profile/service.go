package profile

import (
	"database/sql"
	"errors"
	"time"
)

// Возможные ошибки сервиса
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidActivityType  = errors.New("invalid activity type")
	ErrProfileAlreadyExists = errors.New("profile already exists for this user")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrInvalidImprovStyle   = errors.New("invalid improv style")
	ErrInvalidImprovGoal    = errors.New("invalid improv goal")

	// Ошибки для музыкального профиля
	ErrInvalidMusicGenre = errors.New("invalid music genre")
	ErrInvalidInstrument = errors.New("invalid instrument")
	ErrEmptyInstruments  = errors.New("instruments list cannot be empty for music profile")
)

// ProfileServiceImpl реализует интерфейс ProfileService
type ProfileServiceImpl struct {
	db *sql.DB
}

// NewProfileService создает новый экземпляр сервиса профилей
func NewProfileService(db *sql.DB) ProfileService {
	return &ProfileServiceImpl{
		db: db,
	}
}

// CreateImprovProfile создает новый профиль импровизации
func (s *ProfileServiceImpl) CreateImprovProfile(userID int, description string, goal string, styles []string) (*Profile, error) {
	// Проверяем существование пользователя
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Проверяем, что у пользователя еще нет профиля
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Проверяем, что цель импровизации существует
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_goals_catalog WHERE goal_code = $1)", goal).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidImprovGoal
	}

	// Проверяем, что все стили импровизации существуют
	for _, style := range styles {
		err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM improv_style_catalog WHERE style_code = $1)", style).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidImprovStyle
		}
	}

	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Создаем базовый профиль
	var profileID int
	var createdAt time.Time
	err = tx.QueryRow(`
        INSERT INTO profiles (user_id, description, activity_type) 
        VALUES ($1, $2, $3) 
        RETURNING profile_id, created_at
    `, userID, description, ActivityTypeImprov).Scan(&profileID, &createdAt)
	if err != nil {
		return nil, err
	}

	// Создаем профиль импровизации
	_, err = tx.Exec(`
        INSERT INTO improv_profiles (profile_id, goal)
        VALUES ($1, $2)
    `, profileID, goal)
	if err != nil {
		return nil, err
	}

	// Добавляем стили импровизации
	for _, style := range styles {
		_, err = tx.Exec(`
            INSERT INTO improv_profile_styles (profile_id, style)
            VALUES ($1, $2)
        `, profileID, style)
		if err != nil {
			return nil, err
		}
	}

	// Возвращаем созданный профиль
	return &Profile{
		ProfileID:    profileID,
		UserID:       userID,
		Description:  description,
		ActivityType: ActivityTypeImprov,
		CreatedAt:    createdAt,
	}, nil
}

// CreateMusicProfile создает новый музыкальный профиль
func (s *ProfileServiceImpl) CreateMusicProfile(userID int, description string, genres []string, instruments []string) (*Profile, error) {
	// Проверяем существование пользователя
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Проверяем, что у пользователя еще нет профиля
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Проверяем, что список инструментов не пуст
	if len(instruments) == 0 {
		return nil, ErrEmptyInstruments
	}

	// Проверяем и добавляем жанры музыки
	for _, genre := range genres {
		// Проверяем существование жанра
		err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM music_genre_catalog WHERE genre_code = $1)", genre).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			err = ErrInvalidMusicGenre
			return nil, err
		}
	}

	// Проверяем и добавляем инструменты
	for _, instrument := range instruments {
		// Проверяем существование инструмента
		err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM music_instrument_catalog WHERE instrument_code = $1)", instrument).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			err = ErrInvalidInstrument
			return nil, err
		}
	}

	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()

	// Создаем базовый профиль
	var profileID int
	var createdAt time.Time
	err = tx.QueryRow(`
        INSERT INTO profiles (user_id, description, activity_type) 
        VALUES ($1, $2, $3) 
        RETURNING profile_id, created_at
    `, userID, description, ActivityTypeMusic).Scan(&profileID, &createdAt)
	if err != nil {
		return nil, err
	}

	// Добавляем жанры музыки
	for _, genre := range genres {
		_, err = tx.Exec(`
            INSERT INTO music_profile_genres (profile_id, genre_code)
            VALUES ($1, $2)
        `, profileID, genre)
		if err != nil {
			return nil, err
		}
	}

	// Добавляем инструменты
	for _, instrument := range instruments {
		_, err = tx.Exec(`
            INSERT INTO music_profile_instruments (profile_id, instrument_code)
            VALUES ($1, $2)
        `, profileID, instrument)
		if err != nil {
			return nil, err
		}
	}

	// Возвращаем созданный профиль
	return &Profile{
		ProfileID:    profileID,
		UserID:       userID,
		Description:  description,
		ActivityType: ActivityTypeMusic,
		CreatedAt:    createdAt,
	}, nil
}

// GetProfile получает профиль по ID с детальной информацией в зависимости от типа
func (s *ProfileServiceImpl) GetProfile(profileID int) (*ProfileResponse, error) {
	// Получаем базовый профиль
	var profile Profile
	err := s.db.QueryRow(`
        SELECT profile_id, user_id, description, activity_type, created_at 
        FROM profiles WHERE profile_id = $1
    `, profileID).Scan(&profile.ProfileID, &profile.UserID, &profile.Description, &profile.ActivityType, &profile.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Создаем ответ с базовым профилем
	response := &ProfileResponse{
		Profile: &profile,
	}

	// В зависимости от типа активности, добавляем детальную информацию
	switch profile.ActivityType {
	case ActivityTypeImprov:
		var improvProfile ImprovProfile
		improvProfile.Profile = &profile

		// Получаем цель импровизации
		var goal sql.NullString
		err := s.db.QueryRow(`
            SELECT goal 
            FROM improv_profiles 
            WHERE profile_id = $1
        `, profileID).Scan(&goal)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		if goal.Valid {
			improvProfile.Goal = goal.String
		}

		// Получаем стили импровизации
		rows, err := s.db.Query(`
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

		improvProfile.Styles = styles
		response.ImprovDetail = &improvProfile

	case ActivityTypeMusic:
		var musicProfile MusicProfile
		musicProfile.Profile = &profile

		// Получаем жанры музыки
		genreRows, err := s.db.Query(`
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

		musicProfile.Genres = genres

		// Получаем инструменты
		instrumentRows, err := s.db.Query(`
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

		musicProfile.Instruments = instruments
		response.MusicDetail = &musicProfile
	}

	return response, nil
}
