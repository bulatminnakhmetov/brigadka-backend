package profile

import (
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"
)

type TranslatedItem = profile.TranslatedItem

type UserInfo = user.UserInfo

// Profile представляет базовый профиль пользователя
type Profile struct {
	ProfileID   int    `json:"profile_id"`
	UserID      int    `json:"user_id"`
	Description string `json:"description"`
	UserInfo
}

// ImprovProfile представляет профиль пользователя для импровизации
type ImprovProfile struct {
	Profile
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// MusicProfile представляет профиль пользователя для музыки
type MusicProfile struct {
	Profile
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}
