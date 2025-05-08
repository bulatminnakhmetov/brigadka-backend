package profile

import (
	"database/sql"
	"errors"
	"log"
	"time"

	mediarepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/media"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
	profilerepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
)

// Возможные ошибки сервиса
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrProfileAlreadyExists = errors.New("profile already exists for this user")
	ErrProfileNotFound      = errors.New("profile not found")
	ErrInvalidImprovStyle   = errors.New("invalid improv style")
	ErrInvalidImprovGoal    = errors.New("invalid improv goal")
	ErrInvalidGender        = errors.New("invalid gender")
	ErrInvalidCity          = errors.New("invalid city")
)

// TranslatedItem represents a catalog item with translations
type TranslatedItem struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// City represents a city
type City struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Media struct {
	ID           int    `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// Profile represents profile data for response
type Profile struct {
	FullName       string    `json:"full_name"`
	Birthday       time.Time `json:"birthday,omitempty"`
	Gender         string    `json:"gender,omitempty"`
	CityID         int       `json:"city_id,omitempty"`
	Bio            string    `json:"bio,omitempty"`
	Goal           string    `json:"goal,omitempty"`
	LookingForTeam bool      `json:"looking_for_team"`
	ImprovStyles   []string  `json:"improv_styles,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Avatar         *Media    `json:"avatar,omitempty"`
	Videos         []Media   `json:"videos,omitempty"`
}

// ProfileCreateRequest represents data needed to create a profile
type ProfileCreateRequest struct {
	UserID         int       `json:"user_id" validate:"required"`
	FullName       string    `json:"full_name" validate:"required"`
	Birthday       time.Time `json:"birthday"`
	Gender         string    `json:"gender"`
	CityID         int       `json:"city_id"`
	Bio            string    `json:"bio"`
	Goal           string    `json:"goal"`
	ImprovStyles   []string  `json:"improv_styles"`
	LookingForTeam bool      `json:"looking_for_team"`
	Avatar         *int      `json:"avatar,omitempty"`
	Videos         []int     `json:"videos,omitempty"`
}

// ProfileUpdateRequest represents data needed to update a profile
type ProfileUpdateRequest struct {
	FullName       *string    `json:"full_name,omitempty"`
	Birthday       *time.Time `json:"birthday,omitempty"`
	Gender         *string    `json:"gender,omitempty"`
	CityID         *int       `json:"city_id,omitempty"`
	Bio            *string    `json:"bio,omitempty"`
	Goal           *string    `json:"goal,omitempty"`
	ImprovStyles   []string   `json:"improv_styles,omitempty"`
	LookingForTeam *bool      `json:"looking_for_team,omitempty"`
	Avatar         *int       `json:"avatar,omitempty"`
	Videos         []int      `json:"videos,omitempty"`
}

type MediaRepository interface {
	GetMediaByIDs(mediaIDs []int) ([]mediarepo.Media, error)
	GetMediaByID(mediaID int) (*mediarepo.Media, error)
}

type ProfileRepository interface {
	BeginTx() (*sql.Tx, error)
	CheckUserExists(userID int) (bool, error)
	CheckProfileExists(userID int) (bool, error)
	CreateProfile(tx *sql.Tx, profile *profile.ProfileModel) (time.Time, error)
	AddImprovStyles(tx *sql.Tx, userID int, styles []string) error
	GetProfile(userID int) (*profile.ProfileModel, error)
	GetProfileByUserID(userID int) (*profile.ProfileModel, error)

	GetProfileAvatar(userID int) (*int, error)
	SetProfileAvatar(tx *sql.Tx, userID int, mediaID int) error
	RemoveAvatar(tx *sql.Tx, userID int) error

	GetProfileVideos(userID int) ([]int, error)
	SetProfileVideos(tx *sql.Tx, userID int, videos []int) error

	ValidateMediaRole(role string) (bool, error)
	GetImprovStyles(userID int) ([]string, error)
	UpdateProfile(tx *sql.Tx, profile *profile.UpdateProfileModel) error
	ClearImprovStyles(tx *sql.Tx, userID int) error
	ClearProfileMedia(tx *sql.Tx, userID int, role string) error
	ValidateImprovGoal(goal string) (bool, error)
	ValidateImprovStyle(style string) (bool, error)
	ValidateGender(gender string) (bool, error)
	ValidateCity(cityID int) (bool, error)
	GetImprovStylesCatalog(lang string) ([]profile.TranslatedItem, error)
	GetImprovGoalsCatalog(lang string) ([]profile.TranslatedItem, error)
	GetGendersCatalog(lang string) ([]profile.TranslatedItem, error)
	GetCities() ([]struct {
		ID   int
		Name string
	}, error)
	SearchProfiles(
		fullName *string,
		lookingForTeam *bool,
		goal *string,
		improvStyles []string,
		birthDateMin *time.Time,
		birthDateMax *time.Time,
		gender *string,
		cityID *int,
		hasAvatar *bool,
		hasVideo *bool,
		createdAfter *time.Time,
		page int,
		pageSize int,
	) ([]*profile.ProfileModel, int, error)
}

// ProfileServiceImpl реализует интерфейс ProfileService
type ProfileServiceImpl struct {
	profileRepo ProfileRepository
	mediaRepo   MediaRepository
}

// NewProfileService создает новый экземпляр сервиса профилей
func NewProfileService(profileRepo ProfileRepository, mediaRepo MediaRepository) *ProfileServiceImpl {
	return &ProfileServiceImpl{
		profileRepo: profileRepo,
		mediaRepo:   mediaRepo,
	}
}

func convertMedia(media *mediarepo.Media) *Media {
	if media == nil {
		return nil
	}
	return &Media{
		ID:           media.ID,
		URL:          media.URL,
		ThumbnailURL: media.ThumbnailURL,
	}
}

func convertMediaList(mediaList []mediarepo.Media) []Media {
	converted := make([]Media, len(mediaList))
	for i, media := range mediaList {
		converted[i] = *convertMedia(&media)
	}
	return converted
}

// convertToProfile преобразует данные из репозитория в структуру для ответа
func convertToProfile(profile *profilerepo.ProfileModel, styles []string, avatar *mediarepo.Media, videos []mediarepo.Media) *Profile {
	return &Profile{
		FullName:       profile.FullName,
		Birthday:       profile.Birthday,
		Gender:         profile.Gender,
		CityID:         profile.CityID,
		Bio:            profile.Bio,
		Goal:           profile.Goal,
		LookingForTeam: profile.LookingForTeam,
		ImprovStyles:   styles,
		CreatedAt:      profile.CreatedAt,
		Avatar:         convertMedia(avatar),
		Videos:         convertMediaList(videos),
	}
}

// CreateProfile creates a new profile
func (s *ProfileServiceImpl) CreateProfile(req ProfileCreateRequest) (*Profile, error) {
	// Check user exists
	exists, err := s.profileRepo.CheckUserExists(req.UserID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Check if profile already exists
	exists, err = s.profileRepo.CheckProfileExists(req.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProfileAlreadyExists
	}

	// Validate fields
	valid, err := s.profileRepo.ValidateGender(req.Gender)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidGender
	}

	valid, err = s.profileRepo.ValidateCity(req.CityID)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidCity
	}

	valid, err = s.profileRepo.ValidateImprovGoal(req.Goal)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrInvalidImprovGoal
	}

	// Validate styles
	for _, style := range req.ImprovStyles {
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
	}()

	// Create profile
	profileModel := &profilerepo.ProfileModel{
		UserID:         req.UserID,
		FullName:       req.FullName,
		Birthday:       req.Birthday,
		Gender:         req.Gender,
		CityID:         req.CityID,
		Bio:            req.Bio,
		Goal:           req.Goal,
		LookingForTeam: req.LookingForTeam,
	}

	_, err = s.profileRepo.CreateProfile(tx, profileModel)
	if err != nil {
		return nil, err
	}

	// Add improv styles if provided
	if len(req.ImprovStyles) > 0 {
		err = s.profileRepo.AddImprovStyles(tx, req.UserID, req.ImprovStyles)
		if err != nil {
			return nil, err
		}
	}

	if req.Avatar != nil {
		err := s.profileRepo.SetProfileAvatar(tx, req.UserID, *req.Avatar)
		if err != nil {
			return nil, err
		}
	}

	if req.Videos != nil {
		err := s.profileRepo.SetProfileVideos(tx, req.UserID, req.Videos)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return s.GetProfile(req.UserID)
}

func (s *ProfileServiceImpl) ExpandProfile(profile *profile.ProfileModel) (*Profile, error) {
	if profile == nil {
		return nil, nil
	}

	// Get improv styles
	styles, err := s.profileRepo.GetImprovStyles(profile.UserID)
	if err != nil {
		log.Printf("failed to get improv styles: %v", err)
	}

	// Get avatar
	var avatar *mediarepo.Media
	if profile.Avatar != nil {
		media, err := s.mediaRepo.GetMediaByID(*profile.Avatar)
		if media != nil {
			avatar = media
		}
		if err != nil {
			log.Printf("failed to get avatar media: %v", err)
		}
	}

	// Get videos
	videos, err := s.mediaRepo.GetMediaByIDs(profile.Videos)
	if err != nil {
		log.Printf("failed to get videos media: %v", err)
	}
	return convertToProfile(profile, styles, avatar, videos), nil
}

// GetProfileByUserID retrieves a profile by user ID
func (s *ProfileServiceImpl) GetProfile(userID int) (*Profile, error) {
	// Check user exists
	exists, err := s.profileRepo.CheckUserExists(userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	// Get profile
	profile, err := s.profileRepo.GetProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	return s.ExpandProfile(profile)
}

// UpdateProfile updates an existing profile
func (s *ProfileServiceImpl) UpdateProfile(userID int, req ProfileUpdateRequest) (*Profile, error) {
	// Get profile to check if it exists
	profile, err := s.profileRepo.GetProfileByUserID(userID)
	if err != nil {
		if errors.Is(err, profilerepo.ErrProfileNotExists) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Validate fields
	if req.Gender != nil {
		valid, err := s.profileRepo.ValidateGender(*req.Gender)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidGender
		}
	}

	if req.CityID != nil {
		valid, err := s.profileRepo.ValidateCity(*req.CityID)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidCity
		}
	}

	if req.Goal != nil {
		valid, err := s.profileRepo.ValidateImprovGoal(*req.Goal)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrInvalidImprovGoal
		}
	}

	// Validate styles
	for _, style := range req.ImprovStyles {
		valid, err := s.profileRepo.ValidateImprovStyle(style)
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
	}()

	// Update profile
	updateProfileModel := &profilerepo.UpdateProfileModel{
		UserID:         profile.UserID,
		FullName:       req.FullName,
		Birthday:       req.Birthday,
		Gender:         req.Gender,
		CityID:         req.CityID,
		Bio:            req.Bio,
		Goal:           req.Goal,
		LookingForTeam: req.LookingForTeam,
	}

	err = s.profileRepo.UpdateProfile(tx, updateProfileModel)
	if err != nil {
		return nil, err
	}

	// Clear and re-add styles
	err = s.profileRepo.ClearImprovStyles(tx, userID)
	if err != nil {
		return nil, err
	}

	if len(req.ImprovStyles) > 0 {
		err = s.profileRepo.AddImprovStyles(tx, userID, req.ImprovStyles)
		if err != nil {
			return nil, err
		}
	}

	if req.Avatar != nil {
		err := s.profileRepo.SetProfileAvatar(tx, userID, *req.Avatar)
		if err != nil {
			return nil, err
		}
	}

	if req.Videos != nil {
		err := s.profileRepo.SetProfileVideos(tx, userID, req.Videos)
		if err != nil {
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return s.GetProfile(userID)
}

// GetImprovStyles returns improv styles catalog with translations
func (s *ProfileServiceImpl) GetImprovStyles(lang string) ([]TranslatedItem, error) {
	repoItems, err := s.profileRepo.GetImprovStylesCatalog(lang)
	if err != nil {
		return nil, err
	}

	items := make([]TranslatedItem, len(repoItems))
	for i, item := range repoItems {
		items[i] = TranslatedItem{
			Code:        item.Code,
			Label:       item.Label,
			Description: item.Description,
		}
	}
	return items, nil
}

// GetImprovGoals returns improv goals catalog with translations
func (s *ProfileServiceImpl) GetImprovGoals(lang string) ([]TranslatedItem, error) {
	repoItems, err := s.profileRepo.GetImprovGoalsCatalog(lang)
	if err != nil {
		return nil, err
	}

	items := make([]TranslatedItem, len(repoItems))
	for i, item := range repoItems {
		items[i] = TranslatedItem{
			Code:        item.Code,
			Label:       item.Label,
			Description: item.Description,
		}
	}
	return items, nil
}

// GetGenders returns gender catalog with translations
func (s *ProfileServiceImpl) GetGenders(lang string) ([]TranslatedItem, error) {
	repoItems, err := s.profileRepo.GetGendersCatalog(lang)
	if err != nil {
		return nil, err
	}

	items := make([]TranslatedItem, len(repoItems))
	for i, item := range repoItems {
		items[i] = TranslatedItem{
			Code:        item.Code,
			Label:       item.Label,
			Description: item.Description,
		}
	}
	return items, nil
}

// GetCities returns available cities
func (s *ProfileServiceImpl) GetCities() ([]City, error) {
	repoCities, err := s.profileRepo.GetCities()
	if err != nil {
		return nil, err
	}

	cities := make([]City, len(repoCities))
	for i, city := range repoCities {
		cities[i] = City{
			ID:   city.ID,
			Name: city.Name,
		}
	}
	return cities, nil
}
