package profile

import (
	"github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
	serviceProfile "github.com/bulatminnakhmetov/brigadka-backend/internal/service/profile"
)

// Request models

// CreateProfileRequest represents the base request for creating a profile
type CreateProfileRequest struct {
	UserID      int    `json:"user_id"`
	Description string `json:"description"`
}

// CreateImprovProfileRequest represents a request to create an improv profile
type CreateImprovProfileRequest struct {
	CreateProfileRequest
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// CreateMusicProfileRequest represents a request to create a music profile
type CreateMusicProfileRequest struct {
	CreateProfileRequest
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// UpdateProfileRequest represents the base request for updating a profile
type UpdateProfileRequest struct {
	UpdateUserInfoRequest
	Description string `json:"description"`
}

type UpdateUserInfoRequest struct {
	UserInfoDTO
}

// UpdateImprovProfileRequest represents a request to update an improv profile
type UpdateImprovProfileRequest struct {
	UpdateProfileRequest
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// UpdateMusicProfileRequest represents a request to update a music profile
type UpdateMusicProfileRequest struct {
	UpdateProfileRequest
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

// Response models

// UserInfoDTO represents user information in API responses
type UserInfoDTO struct {
	FullName string `json:"full_name"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty"`
	CityID   int    `json:"city_id,omitempty"`
}

// ProfileDTO represents a base profile in API responses
type ProfileDTO struct {
	UserInfoDTO
	ProfileID   int    `json:"profile_id"`
	Description string `json:"description"`
}

// ImprovProfileDTO represents an improv profile in API responses
type ImprovProfileDTO struct {
	ProfileDTO
	Goal           string   `json:"goal"`
	Styles         []string `json:"styles"`
	LookingForTeam bool     `json:"looking_for_team"`
}

// MusicProfileDTO represents a music profile in API responses
type MusicProfileDTO struct {
	ProfileDTO
	Genres      []string `json:"genres,omitempty"`
	Instruments []string `json:"instruments,omitempty"`
}

type VideoDTO struct {
	Url          string `json:"url"`
	ThumbnailUrl string `json:"thumbnail_url"`
}

// MediaDTO represents profile media information in API responses
type MediaDTO struct {
	Avatar string     `json:"avatar,omitempty"`
	Videos []VideoDTO `json:"videos,omitempty"`
}

// Combined response models

// ImprovProfileResponse represents the full improv profile response including user info and media
type ImprovProfileResponse struct {
	Profile ImprovProfileDTO `json:"profile"`
	Media   MediaDTO         `json:"media"`
}

// MusicProfileResponse represents the full music profile response including user info and media
type MusicProfileResponse struct {
	Profile MusicProfileDTO `json:"profile"`
	Media   MediaDTO        `json:"media"`
}

// UserProfilesResponse represents the response format for user profiles
type UserProfilesResponse struct {
	Profiles map[string]int `json:"profiles"` // activity_type -> profile_id
}

type CatalogResponse struct {
	Items []serviceProfile.TranslatedItem `json:"items"`
}

// Conversion functions: Service models to API DTOs

func ToUserInfoDTO(userInfo *user.UserInfo) UserInfoDTO {
	if userInfo == nil {
		return UserInfoDTO{}
	}

	return UserInfoDTO{
		FullName: userInfo.FullName,
		Gender:   userInfo.Gender,
		Age:      userInfo.Age,
		CityID:   userInfo.CityID,
	}
}

func ToProfileDTO(profile serviceProfile.Profile) ProfileDTO {
	return ProfileDTO{
		UserInfoDTO: ToUserInfoDTO(&profile.UserInfo),
		ProfileID:   profile.ProfileID,
		Description: profile.Description,
	}
}

func ToImprovProfileDTO(profile *serviceProfile.ImprovProfile) ImprovProfileDTO {
	if profile == nil {
		return ImprovProfileDTO{}
	}

	return ImprovProfileDTO{
		ProfileDTO:     ToProfileDTO(profile.Profile),
		Goal:           profile.Goal,
		Styles:         profile.Styles,
		LookingForTeam: profile.LookingForTeam,
	}
}

func ToMusicProfileDTO(profile *serviceProfile.MusicProfile) MusicProfileDTO {
	if profile == nil {
		return MusicProfileDTO{}
	}

	return MusicProfileDTO{
		ProfileDTO:  ToProfileDTO(profile.Profile),
		Genres:      profile.Genres,
		Instruments: profile.Instruments,
	}
}

func ToMediaDTO(profileMedia *media.ProfileMedia) MediaDTO {
	if profileMedia == nil {
		return MediaDTO{}
	}

	mediaDTO := MediaDTO{
		Avatar: profileMedia.Avatar,
	}

	if len(profileMedia.Videos) > 0 {
		videos := make([]VideoDTO, len(profileMedia.Videos))
		for i, video := range profileMedia.Videos {
			videos[i].Url = video.Url
			videos[i].ThumbnailUrl = video.ThumbnailUrl
		}
		mediaDTO.Videos = videos
	}

	return mediaDTO
}

func ToImprovProfileResponse(profile *serviceProfile.ImprovProfile, profileMedia *media.ProfileMedia) ImprovProfileResponse {
	return ImprovProfileResponse{
		Profile: ToImprovProfileDTO(profile),
		Media:   ToMediaDTO(profileMedia),
	}
}

func ToMusicProfileResponse(profile *serviceProfile.MusicProfile, profileMedia *media.ProfileMedia) MusicProfileResponse {
	return MusicProfileResponse{
		Profile: ToMusicProfileDTO(profile),
		Media:   ToMediaDTO(profileMedia),
	}
}

// Conversion functions: API DTOs to Service models

func ToUserInfo(dto UserInfoDTO) user.UserInfo {
	return user.UserInfo{
		FullName: dto.FullName,
		Gender:   dto.Gender,
		Age:      dto.Age,
		CityID:   dto.CityID,
	}
}
