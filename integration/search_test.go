package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// SearchIntegrationTestSuite defines a suite for search integration tests
type SearchIntegrationTestSuite struct {
	suite.Suite
	appUrl    string
	testUsers map[string]*testUser // Map to store test users by scenario
}

// testUser holds user details and associated profiles
type testUser struct {
	UserID      int
	Token       string
	ProfileID   int
	ProfileType string
}

// SetupSuite prepares the common test environment
func (s *SearchIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default value for local testing
	}
	s.testUsers = make(map[string]*testUser)

	// Create test users with profiles for different test scenarios
	s.createTestUserWithImprovProfile("improv_user")
	s.createTestUserWithMusicProfile("music_user")
	s.createTestUserWithFilters("filtered_user", "Smith", 2, 28)
}

// createTestUserWithImprovProfile creates a test user with an improv profile
func (s *SearchIntegrationTestSuite) createTestUserWithImprovProfile(key string) {
	t := s.T()

	// Create a test user
	userID, token, err := s.createTestUser(fmt.Sprintf("improv_%d", time.Now().UnixNano()), "New York", "male", 30)
	assert.NoError(t, err, "Failed to create test user")

	// Create an improv profile
	profileID, err := s.createImprovProfile(userID, token, "Improv profile for search tests", "Hobby", []string{"Short Form"}, true)
	assert.NoError(t, err, "Failed to create improv profile")

	// Store user and profile info for later use
	s.testUsers[key] = &testUser{
		UserID:      userID,
		Token:       token,
		ProfileID:   profileID,
		ProfileType: profile.ActivityTypeImprov,
	}
}

// createTestUserWithMusicProfile creates a test user with a music profile
func (s *SearchIntegrationTestSuite) createTestUserWithMusicProfile(key string) {
	t := s.T()

	// Create a test user
	userID, token, err := s.createTestUser(fmt.Sprintf("musician_%d", time.Now().UnixNano()), "Los Angeles", "female", 25)
	assert.NoError(t, err, "Failed to create test user")

	// Create a music profile
	profileID, err := s.createMusicProfile(userID, token, "Music profile for search tests", []string{"rock"}, []string{"bass_guitar"})
	assert.NoError(t, err, "Failed to create music profile")

	// Store user and profile info for later use
	s.testUsers[key] = &testUser{
		UserID:      userID,
		Token:       token,
		ProfileID:   profileID,
		ProfileType: profile.ActivityTypeMusic,
	}
}

// createTestUserWithFilters creates a test user with specific attributes for filtering tests
func (s *SearchIntegrationTestSuite) createTestUserWithFilters(key, name string, cityID, age int) {
	t := s.T()

	// Create a test user with specific attributes
	userID, token, err := s.createTestUser(name, "", "male", age)
	assert.NoError(t, err, "Failed to create test user")

	// Create a profile with specific attributes for search filtering
	profileID, err := s.createMusicProfile(userID, token, "Profile for filtering tests", []string{"jazz"}, []string{"piano"})
	assert.NoError(t, err, "Failed to create profile for filtering")

	// Store user and profile info for later use
	s.testUsers[key] = &testUser{
		UserID:      userID,
		Token:       token,
		ProfileID:   profileID,
		ProfileType: profile.ActivityTypeMusic,
	}
}

// Helper function to create a test user
func (s *SearchIntegrationTestSuite) createTestUser(namePart, city, gender string, age int) (int, string, error) {
	// Generate a unique email for the user
	testEmail := fmt.Sprintf("search_test_%s_%d@example.com", namePart, time.Now().UnixNano())
	testPassword := "TestSearch123"

	// Set cityID based on the city name
	cityID := 1 // Default to first city
	if city == "Los Angeles" {
		cityID = 2
	}

	// Register a test user
	registerData := auth.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
		FullName: fmt.Sprintf("Test %s", namePart),
		Gender:   gender,
		Age:      age,
		CityID:   cityID,
	}

	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	if err != nil {
		return 0, "", fmt.Errorf("failed to register test user: %v", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		return 0, "", fmt.Errorf("failed to register test user. Status: %d", registerResp.StatusCode)
	}

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode register response: %v", err)
	}

	return registerResult.User.UserID, registerResult.Token, nil
}

// Helper function to create an improv profile
func (s *SearchIntegrationTestSuite) createImprovProfile(userID int, token, description, goal string, styles []string, lookingForTeam bool) (int, error) {
	// Create improv profile request
	createData := profile.CreateImprovProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  description,
			ActivityType: profile.ActivityTypeImprov,
		},
		Goal:           goal,
		Styles:         styles,
		LookingForTeam: lookingForTeam,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	if err != nil {
		return 0, fmt.Errorf("failed to create improv profile: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("failed to create improv profile. Status: %d", createResp.StatusCode)
	}

	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	if err != nil {
		return 0, fmt.Errorf("failed to decode profile creation response: %v", err)
	}

	return createdProfile.ProfileID, nil
}

// Helper function to create a music profile
func (s *SearchIntegrationTestSuite) createMusicProfile(userID int, token, description string, genres, instruments []string) (int, error) {
	// Create music profile request
	createData := profile.CreateMusicProfileRequest{
		CreateProfileRequest: profile.CreateProfileRequest{
			UserID:       userID,
			Description:  description,
			ActivityType: profile.ActivityTypeMusic,
		},
		Genres:      genres,
		Instruments: instruments,
	}

	createJSON, _ := json.Marshal(createData)
	createReq, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(createJSON))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	createResp, err := client.Do(createReq)
	if err != nil {
		return 0, fmt.Errorf("failed to create music profile: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("failed to create music profile. Status: %d", createResp.StatusCode)
	}

	var createdProfile profile.Profile
	err = json.NewDecoder(createResp.Body).Decode(&createdProfile)
	if err != nil {
		return 0, fmt.Errorf("failed to decode profile creation response: %v", err)
	}

	return createdProfile.ProfileID, nil
}

// TestSearchProfilesPOST tests the POST search endpoint integration
func (s *SearchIntegrationTestSuite) TestSearchProfilesPOST() {
	t := s.T()

	t.Run("Basic search", func(t *testing.T) {
		// Make a basic search request
		reqBody := search.ProfileSearchRequest{
			Limit: 10,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check response status
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify we have results (should include our test users)
		assert.GreaterOrEqual(t, result.TotalCount, 3, "Should find at least our 3 test users")
		assert.Greater(t, len(result.Results), 0, "Should have at least one result")

		// Check pagination values
		assert.GreaterOrEqual(t, result.TotalPages, 1)
		assert.Equal(t, 1, result.CurrentPage)
		assert.Equal(t, 10, result.PageSize)
	})

	t.Run("Filtered search by activity type", func(t *testing.T) {
		// Search for improv profiles
		reqBody := search.ProfileSearchRequest{
			ActivityType: profile.ActivityTypeImprov,
			Limit:        10,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify all results are improv profiles
		for _, item := range result.Results {
			assert.Equal(t, profile.ActivityTypeImprov, item.ActivityType)
		}

		// Verify our test user is among results
		found := false
		for _, res := range result.Results {
			if res.ProfileID == s.testUsers["improv_user"].ProfileID {
				found = true
				break
			}
		}
		assert.True(t, found, "Improv test user should be found in results")
	})

	t.Run("Filtered search by name", func(t *testing.T) {
		// Use a user we created with a specific name
		user := s.testUsers["filtered_user"]

		reqBody := search.ProfileSearchRequest{
			FullName: "Smith",
			Limit:    10,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["filtered_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify the filtered user is found
		found := false
		for _, res := range result.Results {
			if res.ProfileID == user.ProfileID {
				found = true
				assert.Contains(t, res.FullName, "Smith", "Name should contain 'Smith'")
				break
			}
		}
		assert.True(t, found, "User with 'Smith' in name should be found")
	})

	t.Run("Search with improv criteria", func(t *testing.T) {
		improvUser := s.testUsers["improv_user"]
		lookingForTeam := true

		reqBody := search.ProfileSearchRequest{
			ActivityType:         profile.ActivityTypeImprov,
			ImprovGoal:           "Hobby",
			ImprovLookingForTeam: &lookingForTeam,
			Limit:                10,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify our improv user with specific criteria is in results
		found := false
		for _, res := range result.Results {
			if res.ProfileID == improvUser.ProfileID {
				found = true
				assert.Equal(t, "Hobby", res.ImprovGoal)
				assert.NotNil(t, res.ImprovLookingForTeam)
				assert.True(t, *res.ImprovLookingForTeam)
				break
			}
		}
		assert.True(t, found, "Improv user with matching criteria should be found")
	})

	t.Run("Search with music criteria", func(t *testing.T) {
		musicUser := s.testUsers["music_user"]

		reqBody := search.ProfileSearchRequest{
			ActivityType:     profile.ActivityTypeMusic,
			MusicGenres:      []string{"rock"},
			MusicInstruments: []string{"bass_guitar"},
			Limit:            10,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["music_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify our music user is found
		found := false
		for _, res := range result.Results {
			if res.ProfileID == musicUser.ProfileID {
				found = true
				assert.Contains(t, res.MusicGenres, "rock")
				assert.Contains(t, res.MusicInstruments, "bass_guitar")
				break
			}
		}
		assert.True(t, found, "Music user with matching criteria should be found")
	})

	t.Run("Invalid age range", func(t *testing.T) {
		ageMin := 30
		ageMax := 20

		// Make request with invalid age range
		reqBody := search.ProfileSearchRequest{
			AgeMin: &ageMin,
			AgeMax: &ageMax,
		}
		body, err := json.Marshal(reqBody)
		assert.NoError(t, err)

		// Create request with authentication
		req, err := http.NewRequest("POST", s.appUrl+"/api/search/profiles", bytes.NewBuffer(body))
		assert.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Check response status - should be "Bad Request"
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestSearchProfilesGET tests the GET search endpoint integration
func (s *SearchIntegrationTestSuite) TestSearchProfilesGET() {
	t := s.T()

	t.Run("Basic GET search", func(t *testing.T) {
		// Create request with authentication
		req, err := http.NewRequest("GET", s.appUrl+"/api/search/profiles?activity_type=improv", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify all results have the improv activity type
		foundImprovUser := false
		for _, item := range result.Results {
			assert.Equal(t, profile.ActivityTypeImprov, item.ActivityType)
			if item.ProfileID == s.testUsers["improv_user"].ProfileID {
				foundImprovUser = true
			}
		}
		assert.True(t, foundImprovUser, "Our improv test user should be found")
	})

	t.Run("Complex GET search", func(t *testing.T) {
		musicUser := s.testUsers["music_user"]

		// Construct URL with multiple parameters
		url := fmt.Sprintf("%s/api/search/profiles?activity_type=music&music_instrument=bass_guitar&music_genre=rock", s.appUrl)

		// Create request with authentication
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.testUsers["music_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify our specific music user is found
		found := false
		for _, res := range result.Results {
			if res.ProfileID == musicUser.ProfileID {
				found = true
				assert.Equal(t, profile.ActivityTypeMusic, res.ActivityType)
				assert.Contains(t, res.MusicGenres, "rock")
				assert.Contains(t, res.MusicInstruments, "bass_guitar")
				break
			}
		}
		assert.True(t, found, "Music user with matching criteria should be found")
	})

	t.Run("Pagination test", func(t *testing.T) {
		// Test with a small limit and page offset
		url := fmt.Sprintf("%s/api/search/profiles?limit=2&offset=0", s.appUrl)

		// Create request with authentication
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var page1Result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&page1Result)
		assert.NoError(t, err)

		// We should get exactly 2 results on page 1
		assert.Equal(t, 2, len(page1Result.Results))
		assert.Equal(t, 1, page1Result.CurrentPage)
		assert.Equal(t, 2, page1Result.PageSize)

		// Now check the second page
		url = fmt.Sprintf("%s/api/search/profiles?limit=2&offset=2", s.appUrl)

		// Create request with authentication
		req2, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req2.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		resp2, err := client.Do(req2)
		assert.NoError(t, err)
		defer resp2.Body.Close()

		var page2Result search.ProfileSearchResponse
		err = json.NewDecoder(resp2.Body).Decode(&page2Result)
		assert.NoError(t, err)

		// Page 2 should have different results than page 1
		assert.Equal(t, 2, page2Result.CurrentPage)

		if len(page2Result.Results) > 0 && len(page1Result.Results) > 0 {
			assert.NotEqual(t, page1Result.Results[0].ProfileID, page2Result.Results[0].ProfileID,
				"Results on different pages should be different")
		}
	})

	t.Run("Search by age range", func(t *testing.T) {
		// Find users in age range 25-30
		url := fmt.Sprintf("%s/api/search/profiles?age_min=25&age_max=30", s.appUrl)

		// Create request with authentication
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.testUsers["improv_user"].Token)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result search.ProfileSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		assert.NoError(t, err)

		// Verify all results are in the specified age range
		for _, profile := range result.Results {
			if profile.Age != nil {
				age := *profile.Age
				assert.GreaterOrEqual(t, age, 25)
				assert.LessOrEqual(t, age, 30)
			}
		}
	})
}

// TestSearchIntegration runs the suite of search integration tests
func TestSearchIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(SearchIntegrationTestSuite))
}
