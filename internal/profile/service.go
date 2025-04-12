package profile

import (
	"database/sql"
	"errors"
)

// Возможные ошибки сервиса
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidActivityType  = errors.New("invalid activity type")
	ErrProfileAlreadyExists = errors.New("profile already exists for this user")
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

// CreateProfile создает новый профиль пользователя
func (s *ProfileServiceImpl) CreateProfile(userID int, description string, activityType string) (*Profile, error) {
	// Проверяем существование пользователя
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Проверяем, что тип активности существует в каталоге
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM activity_type_catalog WHERE activity_type = $1)", activityType).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidActivityType
	}

	// Проверяем, что у пользователя еще нет профиля
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)", userID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Создаем новый профиль
	var profile Profile
	err = s.db.QueryRow(
		"INSERT INTO profiles (user_id, description, activity_type) VALUES ($1, $2, $3) RETURNING profile_id, user_id, description, activity_type, created_at",
		userID, description, activityType,
	).Scan(&profile.ProfileID, &profile.UserID, &profile.Description, &profile.ActivityType, &profile.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}
