package profile

import (
	"database/sql"
	"errors"
	"time"

	profilerepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"
)

// Общие типы профилей
const (
	ActivityTypeImprov = "improv"
	ActivityTypeMusic  = "music"
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

type UserRepository interface {
	BeginTx() (*sql.Tx, error)
	GetUserInfoByID(id int) (*user.UserInfo, error)
	UpdateUserInfo(tx *sql.Tx, userID int, info *user.UserInfo) error
}

// Repository interface for profile data access
type ProfileRepository interface {
	// Transaction handling
	BeginTx() (*sql.Tx, error)

	// User verification
	CheckUserExists(userID int) (bool, error)
	CheckProfileExists(userID int, activityType string) (bool, error)

	// Profile creation
	CreateBaseProfile(tx *sql.Tx, userID int, description, activityType string) (int, time.Time, error)
	CreateImprovProfile(tx *sql.Tx, profileID int, goal string, lookingForTeam bool) error
	AddImprovStyles(tx *sql.Tx, profileID int, styles []string) error
	AddMusicGenres(tx *sql.Tx, profileID int, genres []string) error
	AddMusicInstruments(tx *sql.Tx, profileID int, instruments []string) error

	// Profile retrieval
	GetProfile(profileID int) (*profilerepo.ProfileModel, error)
	GetImprovProfileDetails(profileID int) (string, bool, error)
	GetImprovStyles(profileID int) ([]string, error)
	GetMusicGenres(profileID int) ([]string, error)
	GetMusicInstruments(profileID int) ([]string, error)
	GetUserProfiles(userID int) (map[string]int, error)

	// Profile update
	UpdateProfileDescription(tx *sql.Tx, profileID int, description string) error
	UpdateImprovProfile(tx *sql.Tx, profileID int, goal string, lookingForTeam bool) error
	ClearImprovStyles(tx *sql.Tx, profileID int) error
	ClearMusicGenres(tx *sql.Tx, profileID int) error
	ClearMusicInstruments(tx *sql.Tx, profileID int) error

	// Validation
	ValidateImprovGoal(goal string) (bool, error)
	ValidateImprovStyle(style string) (bool, error)
	ValidateMusicGenre(genre string) (bool, error)
	ValidateMusicInstrument(instrument string) (bool, error)

	// Catalogs
	GetActivityTypesCatalog(lang string) ([]TranslatedItem, error)
	GetImprovStylesCatalog(lang string) ([]TranslatedItem, error)
	GetImprovGoalsCatalog(lang string) ([]TranslatedItem, error)
	GetMusicGenresCatalog(lang string) ([]TranslatedItem, error)
	GetMusicInstrumentsCatalog(lang string) ([]TranslatedItem, error)
}

// ProfileServiceImpl реализует интерфейс ProfileService
type ProfileServiceImpl struct {
	profileRepo ProfileRepository
	userRepo    UserRepository
}

// NewProfileService создает новый экземпляр сервиса профилей
func NewProfileService(profileRepo ProfileRepository, userRepo UserRepository) *ProfileServiceImpl {
	return &ProfileServiceImpl{
		profileRepo: profileRepo,
		userRepo:    userRepo,
	}
}

// CreateImprovProfile creates a new improv profile
func (s *ProfileServiceImpl) CreateImprovProfile(userID int, description string, goal string, styles []string, lookingForTeam bool) (*ImprovProfile, error) {
	// Check user exists
	exists, err := s.profileRepo.CheckUserExists(userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Get user info
	userInfo, err := s.userRepo.GetUserInfoByID(userID)
	if err != nil {
		return nil, err
	}

	// Check if profile already exists
	exists, err = s.profileRepo.CheckProfileExists(userID, ActivityTypeImprov)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Validate goal
	valid, err := s.profileRepo.ValidateImprovGoal(goal)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidImprovGoal
	}

	// Validate styles
	for _, style := range styles {
		valid, err = s.profileRepo.ValidateImprovStyle(style)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidImprovStyle
		}
	}

	// Start transaction
	tx, err := s.profileRepo.BeginTx()
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

	// Create base profile
	profileID, _, err := s.profileRepo.CreateBaseProfile(tx, userID, description, ActivityTypeImprov)
	if err != nil {
		return nil, err
	}

	// Create improv profile
	err = s.profileRepo.CreateImprovProfile(tx, profileID, goal, lookingForTeam)
	if err != nil {
		return nil, err
	}

	// Add improv styles
	err = s.profileRepo.AddImprovStyles(tx, profileID, styles)
	if err != nil {
		return nil, err
	}

	// Return created profile with user info
	return &ImprovProfile{
		Profile: Profile{
			ProfileID:   profileID,
			UserID:      userID,
			UserInfo:    *userInfo,
			Description: description,
		},
		Goal:           goal,
		Styles:         styles,
		LookingForTeam: lookingForTeam,
	}, nil
}

// CreateMusicProfile creates a new music profile
func (s *ProfileServiceImpl) CreateMusicProfile(userID int, description string, genres []string, instruments []string) (*MusicProfile, error) {
	// Check if instruments list is empty
	if len(instruments) == 0 {
		return nil, ErrEmptyInstruments
	}

	// Check user exists
	exists, err := s.profileRepo.CheckUserExists(userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Get user info
	userInfo, err := s.userRepo.GetUserInfoByID(userID)
	if err != nil {
		return nil, err
	}

	// Check if profile already exists
	exists, err = s.profileRepo.CheckProfileExists(userID, ActivityTypeMusic)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Validate genres
	for _, genre := range genres {
		valid, err := s.profileRepo.ValidateMusicGenre(genre)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidMusicGenre
		}
	}

	// Validate instruments
	for _, instrument := range instruments {
		valid, err := s.profileRepo.ValidateMusicInstrument(instrument)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidInstrument
		}
	}

	// Start transaction
	tx, err := s.profileRepo.BeginTx()
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

	// Create base profile
	profileID, _, err := s.profileRepo.CreateBaseProfile(tx, userID, description, ActivityTypeMusic)
	if err != nil {
		return nil, err
	}

	// Add music genres
	err = s.profileRepo.AddMusicGenres(tx, profileID, genres)
	if err != nil {
		return nil, err
	}

	// Add music instruments
	err = s.profileRepo.AddMusicInstruments(tx, profileID, instruments)
	if err != nil {
		return nil, err
	}

	// Return created profile with user info
	return &MusicProfile{
		Profile: Profile{
			ProfileID:   profileID,
			UserID:      userID,
			UserInfo:    *userInfo,
			Description: description,
		},
		Genres:      genres,
		Instruments: instruments,
	}, nil
}

// GetImprovProfile retrieves an improv profile by ID
func (s *ProfileServiceImpl) GetImprovProfile(profileID int) (*ImprovProfile, error) {
	// Get base profile
	baseProfile, err := s.profileRepo.GetProfile(profileID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Check profile type
	if baseProfile.ActivityType != ActivityTypeImprov {
		return nil, ErrInvalidActivityType
	}

	// Get user info
	userInfo, err := s.userRepo.GetUserInfoByID(baseProfile.UserID)
	if err != nil {
		return nil, err
	}

	// Get improv profile details
	goal, lookingForTeam, err := s.profileRepo.GetImprovProfileDetails(profileID)
	if err != nil {
		return nil, err
	}

	// Get styles
	styles, err := s.profileRepo.GetImprovStyles(profileID)
	if err != nil {
		return nil, err
	}

	return &ImprovProfile{
		Profile: Profile{
			ProfileID:   baseProfile.ID,
			UserID:      baseProfile.UserID,
			Description: baseProfile.Description,
			UserInfo:    *userInfo,
		},
		Goal:           goal,
		Styles:         styles,
		LookingForTeam: lookingForTeam,
	}, nil
}

// GetMusicProfile retrieves a music profile by ID
func (s *ProfileServiceImpl) GetMusicProfile(profileID int) (*MusicProfile, error) {
	// Get base profile
	baseProfile, err := s.profileRepo.GetProfile(profileID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Check profile type
	if baseProfile.ActivityType != ActivityTypeMusic {
		return nil, ErrInvalidActivityType
	}

	// Get user info
	userInfo, err := s.userRepo.GetUserInfoByID(baseProfile.UserID)
	if err != nil {
		return nil, err
	}

	// Get genres
	genres, err := s.profileRepo.GetMusicGenres(profileID)
	if err != nil {
		return nil, err
	}

	// Get instruments
	instruments, err := s.profileRepo.GetMusicInstruments(profileID)
	if err != nil {
		return nil, err
	}

	return &MusicProfile{
		Profile: Profile{
			ProfileID:   baseProfile.ID,
			UserID:      baseProfile.UserID,
			UserInfo:    *userInfo,
			Description: baseProfile.Description,
		},
		Genres:      genres,
		Instruments: instruments,
	}, nil
}

// UpdateImprovProfile updates an existing improv profile and user info
func (s *ProfileServiceImpl) UpdateImprovProfile(profileID int, description string, goal string, styles []string, lookingForTeam bool, fullName string, gender string, age int, cityID int) (*ImprovProfile, error) {
	// Get base profile to check if it exists and get user ID
	baseProfile, err := s.profileRepo.GetProfile(profileID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Check profile type
	if baseProfile.ActivityType != ActivityTypeImprov {
		return nil, ErrInvalidActivityType
	}

	// Validate goal
	valid, err := s.profileRepo.ValidateImprovGoal(goal)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidImprovGoal
	}

	// Validate styles
	for _, style := range styles {
		valid, err = s.profileRepo.ValidateImprovStyle(style)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidImprovStyle
		}
	}

	// Start transaction
	tx, err := s.profileRepo.BeginTx()
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

	// Update user info
	userInfo := &user.UserInfo{
		FullName: fullName,
		Gender:   gender,
		Age:      age,
		CityID:   cityID,
	}
	err = s.userRepo.UpdateUserInfo(tx, baseProfile.UserID, userInfo)
	if err != nil {
		return nil, err
	}

	// Update base profile
	err = s.profileRepo.UpdateProfileDescription(tx, profileID, description)
	if err != nil {
		return nil, err
	}

	// Update improv profile
	err = s.profileRepo.UpdateImprovProfile(tx, profileID, goal, lookingForTeam)
	if err != nil {
		return nil, err
	}

	// Clear and re-add styles
	err = s.profileRepo.ClearImprovStyles(tx, profileID)
	if err != nil {
		return nil, err
	}

	err = s.profileRepo.AddImprovStyles(tx, profileID, styles)
	if err != nil {
		return nil, err
	}

	// Return updated profile with user info
	return &ImprovProfile{
		Profile: Profile{
			ProfileID:   profileID,
			UserID:      baseProfile.UserID,
			UserInfo:    *userInfo,
			Description: description,
		},
		Goal:           goal,
		Styles:         styles,
		LookingForTeam: lookingForTeam,
	}, nil
}

// UpdateMusicProfile updates an existing music profile and user info
func (s *ProfileServiceImpl) UpdateMusicProfile(profileID int, description string, genres []string, instruments []string, fullName string, gender string, age int, cityID int) (*MusicProfile, error) {
	// Check if instruments list is empty
	if len(instruments) == 0 {
		return nil, ErrEmptyInstruments
	}

	// Get base profile to check if it exists and get user ID
	baseProfile, err := s.profileRepo.GetProfile(profileID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Check profile type
	if baseProfile.ActivityType != ActivityTypeMusic {
		return nil, ErrInvalidActivityType
	}

	// Validate genres
	for _, genre := range genres {
		valid, err := s.profileRepo.ValidateMusicGenre(genre)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidMusicGenre
		}
	}

	// Validate instruments
	for _, instrument := range instruments {
		valid, err := s.profileRepo.ValidateMusicInstrument(instrument)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidInstrument
		}
	}

	// Start transaction
	tx, err := s.profileRepo.BeginTx()
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

	// Update user info
	userInfo := &user.UserInfo{
		FullName: fullName,
		Gender:   gender,
		Age:      age,
		CityID:   cityID,
	}
	err = s.userRepo.UpdateUserInfo(tx, baseProfile.UserID, userInfo)
	if err != nil {
		return nil, err
	}

	// Update base profile
	err = s.profileRepo.UpdateProfileDescription(tx, profileID, description)
	if err != nil {
		return nil, err
	}

	// Clear and re-add genres
	err = s.profileRepo.ClearMusicGenres(tx, profileID)
	if err != nil {
		return nil, err
	}
	err = s.profileRepo.AddMusicGenres(tx, profileID, genres)
	if err != nil {
		return nil, err
	}

	// Clear and re-add instruments
	err = s.profileRepo.ClearMusicInstruments(tx, profileID)
	if err != nil {
		return nil, err
	}
	err = s.profileRepo.AddMusicInstruments(tx, profileID, instruments)
	if err != nil {
		return nil, err
	}

	// Return updated profile with user info
	return &MusicProfile{
		Profile: Profile{
			ProfileID:   profileID,
			UserID:      baseProfile.UserID,
			UserInfo:    *userInfo,
			Description: description,
		},
		Genres:      genres,
		Instruments: instruments,
	}, nil
}

// GetUserProfiles retrieves all profiles for a specific user
func (s *ProfileServiceImpl) GetUserProfiles(userID int) (map[string]int, error) {
	return s.profileRepo.GetUserProfiles(userID)
}

// GetActivityTypes returns activity types catalog with translations
func (s *ProfileServiceImpl) GetActivityTypes(lang string) ([]TranslatedItem, error) {
	items, err := s.profileRepo.GetActivityTypesCatalog(lang)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetImprovStyles returns improv styles catalog with translations
func (s *ProfileServiceImpl) GetImprovStyles(lang string) ([]TranslatedItem, error) {
	items, err := s.profileRepo.GetImprovStylesCatalog(lang)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetImprovGoals returns improv goals catalog with translations
func (s *ProfileServiceImpl) GetImprovGoals(lang string) ([]TranslatedItem, error) {
	items, err := s.profileRepo.GetImprovGoalsCatalog(lang)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetMusicGenres returns music genres catalog with translations
func (s *ProfileServiceImpl) GetMusicGenres(lang string) ([]TranslatedItem, error) {
	items, err := s.profileRepo.GetMusicGenresCatalog(lang)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetMusicInstruments returns music instruments catalog with translations
func (s *ProfileServiceImpl) GetMusicInstruments(lang string) ([]TranslatedItem, error) {
	items, err := s.profileRepo.GetMusicInstrumentsCatalog(lang)
	if err != nil {
		return nil, err
	}
	return items, nil
}
