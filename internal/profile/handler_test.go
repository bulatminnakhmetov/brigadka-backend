package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Создаем мок для ProfileService
type MockProfileService struct {
	mock.Mock
}

func (m *MockProfileService) CreateImprovProfile(userID int, description string, goal string, styles []string, lookingForTeam bool) (*ImprovProfile, error) {
	args := m.Called(userID, description, goal, styles, lookingForTeam)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImprovProfile), args.Error(1)
}

func (m *MockProfileService) CreateMusicProfile(userID int, description string, genres []string, instruments []string) (*MusicProfile, error) {
	args := m.Called(userID, description, genres, instruments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MusicProfile), args.Error(1)
}

func (m *MockProfileService) GetImprovProfile(profileID int) (*ImprovProfile, error) {
	args := m.Called(profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImprovProfile), args.Error(1)
}

func (m *MockProfileService) GetMusicProfile(profileID int) (*MusicProfile, error) {
	args := m.Called(profileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MusicProfile), args.Error(1)
}

func (m *MockProfileService) GetActivityTypes(lang string) (ActivityTypeCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ActivityTypeCatalog), args.Error(1)
}

func (m *MockProfileService) GetImprovStyles(lang string) (ImprovStyleCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ImprovStyleCatalog), args.Error(1)
}

func (m *MockProfileService) GetImprovGoals(lang string) (ImprovGoalCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(ImprovGoalCatalog), args.Error(1)
}

func (m *MockProfileService) GetMusicGenres(lang string) (MusicGenreCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(MusicGenreCatalog), args.Error(1)
}

func (m *MockProfileService) GetMusicInstruments(lang string) (MusicInstrumentCatalog, error) {
	args := m.Called(lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(MusicInstrumentCatalog), args.Error(1)
}

func (m *MockProfileService) UpdateImprovProfile(profileID int, description string, goal string, styles []string, lookingForTeam bool) (*ImprovProfile, error) {
	args := m.Called(profileID, description, goal, styles, lookingForTeam)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ImprovProfile), args.Error(1)
}

func (m *MockProfileService) UpdateMusicProfile(profileID int, description string, genres []string, instruments []string) (*MusicProfile, error) {
	args := m.Called(profileID, description, genres, instruments)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MusicProfile), args.Error(1)
}

func (m *MockProfileService) GetUserProfiles(userID int) (map[string]int, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

func TestGetImprovProfileHandler(t *testing.T) {
	t.Run("Successful get", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup test profile
		testProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Improv Description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		// Setup mock
		mockService.On("GetImprovProfile", 1).Return(testProfile, nil)

		// Create test request
		req, _ := http.NewRequest("GET", "/api/profiles/1/improv", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Check response body
		var response ImprovProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, ActivityTypeImprov, response.ActivityType)
		assert.Equal(t, testProfile.Goal, response.Goal)
		assert.Equal(t, testProfile.Styles, response.Styles)
		assert.Equal(t, testProfile.LookingForTeam, response.LookingForTeam)
	})

	t.Run("Profile not found", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return not found error
		mockService.On("GetImprovProfile", 999).Return(nil, ErrProfileNotFound)

		// Create test request
		req, _ := http.NewRequest("GET", "/api/profiles/999/improv", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile not found")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Invalid profile ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create test request with non-numeric ID
		req, _ := http.NewRequest("GET", "/api/profiles/abc/improv", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid profile ID")
	})
}

func TestGetMusicProfileHandler(t *testing.T) {
	t.Run("Successful get", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup test profile
		testProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Music Description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}

		// Setup mock
		mockService.On("GetMusicProfile", 1).Return(testProfile, nil)

		// Create test request
		req, _ := http.NewRequest("GET", "/api/profiles/1/music", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Check response body
		var response MusicProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, ActivityTypeMusic, response.ActivityType)
		assert.ElementsMatch(t, testProfile.Genres, response.Genres)
		assert.ElementsMatch(t, testProfile.Instruments, response.Instruments)
	})

	t.Run("Profile not found", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return not found error
		mockService.On("GetMusicProfile", 999).Return(nil, ErrProfileNotFound)

		// Create test request
		req, _ := http.NewRequest("GET", "/api/profiles/999/music", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile not found")

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}

func TestCreateImprovProfileHandler(t *testing.T) {
	t.Run("Successful create", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create test profile
		testProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Improv Description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		// Setup mock
		mockService.On("CreateImprovProfile", 1, "Test Improv Description", "Hobby", []string{"Short Form"}, true).Return(testProfile, nil)

		// Create test request
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Improv Description",
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Check response body
		var response ImprovProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, ActivityTypeImprov, response.ActivityType)
		assert.Equal(t, testProfile.Goal, response.Goal)
		assert.Equal(t, testProfile.Styles, response.Styles)
		assert.Equal(t, testProfile.LookingForTeam, response.LookingForTeam)
	})

	// Add more test cases for validation errors, service errors, etc.
}

func TestCreateMusicProfileHandler(t *testing.T) {
	t.Run("Successful create", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create test profile
		testProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Test Music Description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}

		// Setup mock
		mockService.On("CreateMusicProfile", 1, "Test Music Description", []string{"rock"}, []string{"guitar"}).Return(testProfile, nil)

		// Create test request
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Music Description",
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Check response body
		var response MusicProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testProfile.ProfileID, response.ProfileID)
		assert.Equal(t, testProfile.UserID, response.UserID)
		assert.Equal(t, testProfile.Description, response.Description)
		assert.Equal(t, ActivityTypeMusic, response.ActivityType)
		assert.Equal(t, testProfile.Genres, response.Genres)
		assert.Equal(t, testProfile.Instruments, response.Instruments)
	})
}

func TestUpdateImprovProfileHandler(t *testing.T) {
	t.Run("Successful update", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Original description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetImprovProfile", 1).Return(originalProfile, nil)

		// Setup updated profile
		updatedProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Updated description",
				ActivityType: ActivityTypeImprov,
				CreatedAt:    time.Now(),
			},
			Goal:           "Career",
			Styles:         []string{"Long Form", "Short Form"},
			LookingForTeam: false,
		}
		mockService.On("UpdateImprovProfile", 1, "Updated description", "Career", []string{"Long Form", "Short Form"}, false).Return(updatedProfile, nil)

		// Create request body
		reqBody := UpdateImprovProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated description",
			},
			Goal:           "Career",
			Styles:         []string{"Long Form", "Short Form"},
			LookingForTeam: false,
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateImprovProfile(rr, req)

		// Check the status code
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Parse response
		var response ImprovProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response data
		assert.Equal(t, updatedProfile.ProfileID, response.ProfileID)
		assert.Equal(t, updatedProfile.UserID, response.UserID)
		assert.Equal(t, updatedProfile.Description, response.Description)
		assert.Equal(t, updatedProfile.ActivityType, response.ActivityType)
		assert.Equal(t, updatedProfile.Goal, response.Goal)
		assert.Equal(t, updatedProfile.Styles, response.Styles)
		assert.Equal(t, updatedProfile.LookingForTeam, response.LookingForTeam)
	})
}

func TestUpdateMusicProfileHandler(t *testing.T) {
	t.Run("Successful update", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup original profile
		originalProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       1,
				Description:  "Original music description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetMusicProfile", 2).Return(originalProfile, nil)

		// Setup updated profile
		updatedProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    2,
				UserID:       1,
				Description:  "Updated music description",
				ActivityType: ActivityTypeMusic,
				CreatedAt:    time.Now(),
			},
			Genres:      []string{"rock", "jazz", "blues"},
			Instruments: []string{"guitar", "piano"},
		}
		mockService.On("UpdateMusicProfile", 2, "Updated music description", []string{"rock", "jazz", "blues"}, []string{"guitar", "piano"}).Return(updatedProfile, nil)

		// Create request body
		reqBody := UpdateMusicProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated music description",
			},
			Genres:      []string{"rock", "jazz", "blues"},
			Instruments: []string{"guitar", "piano"},
		}
		jsonBody, _ := json.Marshal(reqBody)

		// Create a request
		req, _ := http.NewRequest("PUT", "/api/profiles/2/music", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create a ResponseRecorder
		rr := httptest.NewRecorder()

		// Call the handler
		handler.UpdateMusicProfile(rr, req)

		// Check the status code
		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)

		// Parse response
		var response MusicProfile
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Check response data
		assert.NotNil(t, response)
		assert.Equal(t, updatedProfile.ProfileID, response.ProfileID)
		assert.Equal(t, updatedProfile.UserID, response.UserID)
		assert.Equal(t, updatedProfile.Description, response.Description)
		assert.Equal(t, updatedProfile.ActivityType, response.ActivityType)
		assert.ElementsMatch(t, updatedProfile.Genres, response.Genres)
		assert.ElementsMatch(t, updatedProfile.Instruments, response.Instruments)
	})
}

// Additional tests for specific improv profile creation
func TestCreateImprovProfileHandler_Errors(t *testing.T) {
	t.Run("Invalid JSON body", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create invalid request body
		reqBody := []byte(`{invalid json}`)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request body")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateImprovProfile")
	})

	t.Run("Invalid user ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with invalid user ID
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      0, // Invalid ID
				Description: "Test Description",
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid user_id")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateImprovProfile")
	})

	t.Run("Empty goal", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with empty goal
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Description",
			},
			Goal:           "", // Empty goal
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Improv goal is required")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateImprovProfile")
	})

	t.Run("Empty styles", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with empty styles
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Description",
			},
			Goal:           "Hobby",
			Styles:         []string{}, // Empty styles
			LookingForTeam: true,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one improv style is required")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateImprovProfile")
	})

	t.Run("Method not allowed", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create GET request instead of POST
		req, _ := http.NewRequest("GET", "/api/profiles/improv", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateImprovProfile")
	})

	t.Run("Service error - Profile already exists", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return an error
		mockService.On("CreateImprovProfile", 1, "Test Description", "Hobby", []string{"Short Form"}, true).Return(nil, ErrProfileAlreadyExists)

		// Create request
		reqData := CreateImprovProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Description",
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile already exists")

		// Verify mock was called
		mockService.AssertExpectations(t)
	})
}

// Additional tests for specific music profile creation
func TestCreateMusicProfileHandler_Errors(t *testing.T) {
	t.Run("Invalid JSON body", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create invalid request body
		reqBody := []byte(`{invalid json}`)
		req, _ := http.NewRequest("POST", "/api/profiles/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request body")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Invalid user ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with invalid user ID
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      0, // Invalid ID
				Description: "Test Description",
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid user_id")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Empty instruments", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with empty instruments
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Description",
			},
			Genres:      []string{"rock"},
			Instruments: []string{}, // Empty instruments
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one instrument is required")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Method not allowed", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create GET request instead of POST
		req, _ := http.NewRequest("GET", "/api/profiles/music", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

		// Verify no mock calls
		mockService.AssertNotCalled(t, "CreateMusicProfile")
	})

	t.Run("Service error - Profile already exists", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return an error
		mockService.On("CreateMusicProfile", 1, "Test Description", []string{"rock"}, []string{"guitar"}).Return(nil, ErrProfileAlreadyExists)

		// Create request
		reqData := CreateMusicProfileRequest{
			CreateProfileRequest: CreateProfileRequest{
				UserID:      1,
				Description: "Test Description",
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("POST", "/api/profiles/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.CreateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile already exists")

		// Verify mock was called
		mockService.AssertExpectations(t)
	})
}

// Additional tests for updating improv profile
func TestUpdateImprovProfileHandler_Errors(t *testing.T) {
	t.Run("Invalid JSON body", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Improv profile",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		mockService.On("GetImprovProfile", 1).Return(improvProfile, nil)

		// Create invalid request body
		reqBody := []byte(`{invalid json}`)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request body")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})

	t.Run("Invalid profile ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with non-numeric profile ID
		req, _ := http.NewRequest("PUT", "/api/profiles/abc/improv", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid profile ID")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})

	t.Run("Missing authentication", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request without user ID in context
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "Unauthorized")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})

	t.Run("Profile not found", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup mock to return profile not found error
		mockService.On("GetImprovProfile", 999).Return(nil, ErrProfileNotFound)

		// Create request
		req, _ := http.NewRequest("PUT", "/api/profiles/999/improv", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), "Profile not found")

		// Verify mock was called
		mockService.AssertExpectations(t)
	})

	t.Run("Unauthorized user", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup profile owned by user 2
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       2, // Different from the user in the context
				Description:  "Improv profile",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}

		mockService.On("GetImprovProfile", 1).Return(improvProfile, nil)

		// Create request
		reqData := UpdateImprovProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Unauthorized update attempt",
			},
			Goal:           "Career",
			Styles:         []string{"Harold"},
			LookingForTeam: false,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID 1 to context (different from profile owner)
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), "Forbidden")

		// Verify mock was called
		mockService.AssertExpectations(t)
	})

	t.Run("Empty goal", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup a valid profile
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Improv profile",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetImprovProfile", 1).Return(improvProfile, nil)

		// Create request with empty goal
		reqData := UpdateImprovProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated description",
			},
			Goal:           "", // Empty goal
			Styles:         []string{"Harold"},
			LookingForTeam: false,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Improv goal is required")

		// Verify GetProfile was called but not UpdateImprovProfile
		mockService.AssertCalled(t, "GetImprovProfile", 1)
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})

	t.Run("Empty styles", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup a valid profile
		improvProfile := &ImprovProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Improv profile",
				ActivityType: ActivityTypeImprov,
			},
			Goal:           "Hobby",
			Styles:         []string{"Short Form"},
			LookingForTeam: true,
		}
		mockService.On("GetImprovProfile", 1).Return(improvProfile, nil)

		// Create request with empty styles
		reqData := UpdateImprovProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated description",
			},
			Goal:           "Career",
			Styles:         []string{}, // Empty styles
			LookingForTeam: false,
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/improv", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one improv style is required")

		// Verify GetProfile was called but not UpdateImprovProfile
		mockService.AssertCalled(t, "GetImprovProfile", 1)
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})

	t.Run("Method not allowed", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create GET request instead of PUT
		req, _ := http.NewRequest("GET", "/api/profiles/1/improv", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateImprovProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

		// Verify no mock calls
		mockService.AssertNotCalled(t, "UpdateImprovProfile")
	})
}

// Additional tests for updating music profile
func TestUpdateMusicProfileHandler_Errors(t *testing.T) {
	t.Run("Invalid JSON body", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup a valid profile
		musicProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Music profile",
				ActivityType: ActivityTypeMusic,
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetMusicProfile", 1).Return(musicProfile, nil)

		// Create invalid request body
		reqBody := []byte(`{invalid json}`)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Invalid request body")

		// Verify no mock calls
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Empty instruments", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup a valid profile
		musicProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Music profile",
				ActivityType: ActivityTypeMusic,
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetMusicProfile", 1).Return(musicProfile, nil)

		// Create request with empty instruments
		reqData := UpdateMusicProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated description",
			},
			Genres:      []string{"jazz", "blues"},
			Instruments: []string{}, // Empty instruments
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "At least one instrument is required")

		// Verify GetProfile was called but not UpdateMusicProfile
		mockService.AssertCalled(t, "GetMusicProfile", 1)
		mockService.AssertNotCalled(t, "UpdateMusicProfile")
	})

	t.Run("Service error during update", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Setup a valid profile
		musicProfile := &MusicProfile{
			Profile: Profile{
				ProfileID:    1,
				UserID:       1,
				Description:  "Music profile",
				ActivityType: ActivityTypeMusic,
			},
			Genres:      []string{"rock"},
			Instruments: []string{"guitar"},
		}
		mockService.On("GetMusicProfile", 1).Return(musicProfile, nil)

		// Setup mock to return error during update
		mockService.On("UpdateMusicProfile", 1, "Updated description", []string{"rock", "jazz"}, []string{"guitar", "piano"}).Return(nil, errors.New("database error"))

		// Create valid request
		reqData := UpdateMusicProfileRequest{
			UpdateProfileRequest: UpdateProfileRequest{
				Description: "Updated description",
			},
			Genres:      []string{"rock", "jazz"},
			Instruments: []string{"guitar", "piano"},
		}
		reqBody, _ := json.Marshal(reqData)
		req, _ := http.NewRequest("PUT", "/api/profiles/1/music", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.UpdateMusicProfile(rr, req)

		// Check status code
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database error")

		// Verify both mocks were called
		mockService.AssertExpectations(t)
	})
}

func TestGetUserProfilesHandler(t *testing.T) {
	t.Run("Successful retrieval", func(t *testing.T) {
		// Setup expected profiles
		expectedProfiles := map[string]int{
			ActivityTypeImprov: 10,
			ActivityTypeMusic:  20,
		}

		// Create mock service
		mockService := new(MockProfileService)
		mockService.On("GetUserProfiles", 1).Return(expectedProfiles, nil)

		handler := NewProfileHandler(mockService)

		// Create request
		req, _ := http.NewRequest("GET", "/api/users/1/profiles", nil)

		// Add user ID to context (authenticated as same user)
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserProfiles(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse response
		var response UserProfilesResponse
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify profiles match expected
		assert.Equal(t, expectedProfiles, response.Profiles)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})

	t.Run("Unauthorized access", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request without authentication
		req, _ := http.NewRequest("GET", "/api/users/1/profiles", nil)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserProfiles(rr, req)

		// Check response
		assert.Equal(t, http.StatusUnauthorized, rr.Code)

		// Verify no calls were made to the service
		mockService.AssertNotCalled(t, "GetUserProfiles")
	})

	t.Run("Forbidden access (different user)", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request
		req, _ := http.NewRequest("GET", "/api/users/2/profiles", nil)

		// Add different user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserProfiles(rr, req)

		// Check response
		assert.Equal(t, http.StatusForbidden, rr.Code)

		// Verify no calls were made to the service
		mockService.AssertNotCalled(t, "GetUserProfiles")
	})

	t.Run("Invalid user ID", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Create request with invalid user ID
		req, _ := http.NewRequest("GET", "/api/users/invalid/profiles", nil)

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserProfiles(rr, req)

		// Check response
		assert.Equal(t, http.StatusBadRequest, rr.Code)

		// Verify no calls were made to the service
		mockService.AssertNotCalled(t, "GetUserProfiles")
	})

	t.Run("Service error", func(t *testing.T) {
		// Create mock service that returns an error
		mockService := new(MockProfileService)
		mockService.On("GetUserProfiles", 1).Return(nil, errors.New("database error"))

		handler := NewProfileHandler(mockService)

		// Create request
		req, _ := http.NewRequest("GET", "/api/users/1/profiles", nil)

		// Add user ID to context
		ctx := context.WithValue(req.Context(), "user_id", 1)
		req = req.WithContext(ctx)

		// Create response recorder
		rr := httptest.NewRecorder()

		// Call handler
		handler.GetUserProfiles(rr, req)

		// Check response
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		// Verify mock expectations
		mockService.AssertExpectations(t)
	})
}
