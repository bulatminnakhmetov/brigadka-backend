package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// StyleMatchOrderingTestSuite defines a test suite for testing result ordering based on style matches
type StyleMatchOrderingTestSuite struct {
	suite.Suite
	appUrl      string
	testDirPath string
	authToken   string
	userID      int
	createdAt   time.Time
}

// SetupSuite prepares the test environment before all tests
func (s *StyleMatchOrderingTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	s.testDirPath = "testdata/style_match"
	os.MkdirAll(s.testDirPath, 0755)

	// Register a user who will perform searches
	var err error
	s.authToken, s.userID = s.registerTestUser(s.T())
	assert.NotEmpty(s.T(), s.authToken)
	assert.NotZero(s.T(), s.userID)

	// Create a profile for the searching user with specific improv styles
	searcherProfile := map[string]interface{}{
		"user_id":          s.userID,
		"full_name":        "Test Searcher",
		"birthday":         "1990-01-01",
		"gender":           "male",
		"city_id":          1,
		"bio":              "Test searcher bio",
		"goal":             "hobby",
		"improv_styles":    []string{"shortform", "longform", "musical"},
		"looking_for_team": true,
	}

	reqBody, _ := json.Marshal(searcherProfile)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/profiles", s.appUrl), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(s.T(), err)
	if resp != nil {
		defer resp.Body.Close()
		assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

		var profileResp profile.ProfileResponse
		err = json.NewDecoder(resp.Body).Decode(&profileResp)
		assert.NoError(s.T(), err)
		s.createdAt = profileResp.CreatedAt
	}
}

// TearDownSuite cleans up after all tests
func (s *StyleMatchOrderingTestSuite) TearDownSuite() {
	os.RemoveAll(s.testDirPath)
}

// Register a test user and return auth token and user ID
func (s *StyleMatchOrderingTestSuite) registerTestUser(t *testing.T) (string, int) {
	// Create unique test credentials
	email := fmt.Sprintf("style_match_test_%d@example.com", time.Now().UnixNano())
	password := "TestPassword123!"

	// Prepare registration request
	registerData := map[string]string{
		"email":    email,
		"password": password,
	}

	registerJSON, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/auth/register", s.appUrl), bytes.NewBuffer(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var authResponse struct {
		Token  string `json:"token"`
		UserID int    `json:"user_id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	assert.NoError(t, err)

	return authResponse.Token, authResponse.UserID
}

// Create a test profile with given styles
func (s *StyleMatchOrderingTestSuite) createProfileWithStyles(t *testing.T, fullName string, styles []string, gender string) int {
	// Register a user for this profile
	authToken, userID := s.registerTestUser(t)

	// Create profile with specified styles
	profileData := map[string]interface{}{
		"user_id":          userID,
		"full_name":        fullName,
		"birthday":         "1990-01-01",
		"gender":           gender,
		"city_id":          1,
		"bio":              "Test bio",
		"goal":             "hobby",
		"improv_styles":    styles,
		"looking_for_team": true,
	}

	reqBody, _ := json.Marshal(profileData)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/profiles", s.appUrl), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	return userID
}

// Helper function to execute a search request
func (s *StyleMatchOrderingTestSuite) executeSearch(filter map[string]interface{}) (*profile.SearchResponse, error) {
	reqBody, _ := json.Marshal(filter)

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/profiles/search", s.appUrl), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search request failed with status code: %d", resp.StatusCode)
	}

	var result profile.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// TestStyleMatchOrdering tests that profiles are ordered by matching styles
func (s *StyleMatchOrderingTestSuite) TestStyleMatchOrdering() {
	t := s.T()

	// Create profiles with different levels of style matches to the searcher
	// Searcher has: shortform, longform, musical

	// 3 matches (all styles match)
	threeMatchesID := s.createProfileWithStyles(t, "Three Matches", []string{"shortform", "longform", "musical"}, "male")

	// 2 matches
	twoMatchesID1 := s.createProfileWithStyles(t, "Two Matches 1", []string{"shortform", "longform", "battles"}, "male")
	twoMatchesID2 := s.createProfileWithStyles(t, "Two Matches 2", []string{"shortform", "musical", "rap"}, "male")

	// 1 match
	oneMatchID1 := s.createProfileWithStyles(t, "One Match 1", []string{"shortform", "absurd", "realistic"}, "male")
	oneMatchID2 := s.createProfileWithStyles(t, "One Match 2", []string{"longform", "realistic", "absurd"}, "male")

	// No matches
	noMatchesID := s.createProfileWithStyles(t, "No Matches", []string{"absurd", "realistic", "playback"}, "male")

	// Execute search with no additional filters
	filter := map[string]interface{}{
		"created_after": s.createdAt,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Profiles), 6, "Should return at least the 6 created profiles")

	// Extract the user IDs in the order they were returned
	orderedUserIDs := make([]int, 0, len(result.Profiles))
	for _, p := range result.Profiles {
		if p.UserID != s.userID { // Exclude the searcher's profile
			orderedUserIDs = append(orderedUserIDs, p.UserID)
		}
	}

	// Check ordering rules - profiles with more style matches should be ranked higher

	// Find the positions of our test profiles
	threeMatchesPos := indexOf(threeMatchesID, orderedUserIDs)
	twoMatchesPos1 := indexOf(twoMatchesID1, orderedUserIDs)
	twoMatchesPos2 := indexOf(twoMatchesID2, orderedUserIDs)
	oneMatchesPos1 := indexOf(oneMatchID1, orderedUserIDs)
	oneMatchesPos2 := indexOf(oneMatchID2, orderedUserIDs)
	noMatchesPos := indexOf(noMatchesID, orderedUserIDs)

	// Verify ordering
	assert.NotEqual(t, -1, threeMatchesPos, "Three matches profile should be in results")
	assert.NotEqual(t, -1, twoMatchesPos1, "Two matches profile 1 should be in results")
	assert.NotEqual(t, -1, twoMatchesPos2, "Two matches profile 2 should be in results")
	assert.NotEqual(t, -1, oneMatchesPos1, "One match profile 1 should be in results")
	assert.NotEqual(t, -1, oneMatchesPos2, "One match profile 2 should be in results")
	assert.NotEqual(t, -1, noMatchesPos, "No matches profile should be in results")

	// Three matches should come before all others
	assert.Less(t, threeMatchesPos, twoMatchesPos1)
	assert.Less(t, threeMatchesPos, twoMatchesPos2)
	assert.Less(t, threeMatchesPos, oneMatchesPos1)
	assert.Less(t, threeMatchesPos, oneMatchesPos2)
	assert.Less(t, threeMatchesPos, noMatchesPos)

	// Two matches should come before one and zero matches
	assert.Less(t, twoMatchesPos1, oneMatchesPos1)
	assert.Less(t, twoMatchesPos1, oneMatchesPos2)
	assert.Less(t, twoMatchesPos1, noMatchesPos)
	assert.Less(t, twoMatchesPos2, oneMatchesPos1)
	assert.Less(t, twoMatchesPos2, oneMatchesPos2)
	assert.Less(t, twoMatchesPos2, noMatchesPos)

	// One match should come before no matches
	assert.Less(t, oneMatchesPos1, noMatchesPos)
	assert.Less(t, oneMatchesPos2, noMatchesPos)
}

// TestStyleMatchOrderingWithFilters tests that ordering by style match works with other filters
func (s *StyleMatchOrderingTestSuite) TestStyleMatchOrderingWithFilters() {
	t := s.T()

	s.createProfileWithStyles(t, "Two Female", []string{"shortform", "longform", "battles"}, "male")
	s.createProfileWithStyles(t, "Three Female", []string{"shortform", "longform", "musical"}, "male")
	s.createProfileWithStyles(t, "One Female", []string{"shortform", "absurd", "realistic"}, "male")

	twoMatchesID := s.createProfileWithStyles(t, "Two Female", []string{"shortform", "longform", "battles"}, "female")
	threeMatchesID := s.createProfileWithStyles(t, "Three Female", []string{"shortform", "longform", "musical"}, "female")
	oneMatchID := s.createProfileWithStyles(t, "One Female", []string{"shortform", "absurd", "realistic"}, "female")

	// Execute search with gender filter (female only)
	filter := map[string]interface{}{
		"genders":       []string{"female"},
		"created_after": s.createdAt,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Profiles), 3, "Should return at only the 3 female profiles")

	// Extract the user IDs in the order they were returned
	orderedUserIDs := make([]int, 0, len(result.Profiles))
	for _, p := range result.Profiles {
		if p.UserID != s.userID { // Exclude the searcher's profile
			orderedUserIDs = append(orderedUserIDs, p.UserID)
		}
	}

	// Check ordering - even with the gender filter, style matches should determine order
	threeMatchesPos := indexOf(threeMatchesID, orderedUserIDs)
	twoMatchesPos := indexOf(twoMatchesID, orderedUserIDs)
	oneMatchesPos := indexOf(oneMatchID, orderedUserIDs)

	assert.NotEqual(t, -1, threeMatchesPos, "Three matches profile should be in results")
	assert.NotEqual(t, -1, twoMatchesPos, "Two matches profile should be in results")
	assert.NotEqual(t, -1, oneMatchesPos, "One match profile should be in results")

	// Three matches should come before two matches
	assert.Less(t, threeMatchesPos, twoMatchesPos)

	// Two matches should come before one match
	assert.Less(t, twoMatchesPos, oneMatchesPos)
}

// TestCreationDateSecondaryOrdering tests that when style matches are equal, creation date is used as secondary sort
func (s *StyleMatchOrderingTestSuite) TestCreationDateSecondaryOrdering() {
	t := s.T()

	// Create three profiles with the same number of style matches (2)
	// Searcher has: shortform, longform, musical

	// First profile (oldest)
	twoMatchesID1 := s.createProfileWithStyles(t, "Two Matches Old", []string{"shortform", "longform", "battles"}, "male")

	// Wait a bit to ensure different creation timestamps
	time.Sleep(1 * time.Second)

	// Second profile (middle)
	twoMatchesID2 := s.createProfileWithStyles(t, "Two Matches Middle", []string{"shortform", "longform", "rap"}, "male")

	// Wait a bit more
	time.Sleep(1 * time.Second)

	// Third profile (newest)
	twoMatchesID3 := s.createProfileWithStyles(t, "Two Matches New", []string{"shortform", "longform", "playback"}, "male")

	// Execute search
	filter := map[string]interface{}{
		"created_after": s.createdAt,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)

	// Extract the user IDs in the order they were returned
	orderedUserIDs := make([]int, 0, len(result.Profiles))
	for _, p := range result.Profiles {
		if p.UserID != s.userID { // Exclude the searcher's profile
			orderedUserIDs = append(orderedUserIDs, p.UserID)
		}
	}

	// Find positions
	pos1 := indexOf(twoMatchesID1, orderedUserIDs)
	pos2 := indexOf(twoMatchesID2, orderedUserIDs)
	pos3 := indexOf(twoMatchesID3, orderedUserIDs)

	assert.NotEqual(t, -1, pos1)
	assert.NotEqual(t, -1, pos2)
	assert.NotEqual(t, -1, pos3)

	// With equal style matches, newer profiles should come first (DESC ordering by created_at)
	assert.Less(t, pos3, pos2, "Newest profile should come before middle profile")
	assert.Less(t, pos2, pos1, "Middle profile should come before oldest profile")
}

// Helper function to update a profile's gender
func (s *StyleMatchOrderingTestSuite) updateProfileGender(t *testing.T, userID int, gender string) {
	// Register a user for this profile
	authToken, _ := s.registerTestUser(t)

	// Update gender
	updateData := map[string]interface{}{
		"gender": gender,
	}

	reqBody, _ := json.Marshal(updateData)

	// Get auth token for this user ID (in a real implementation, you'd need proper authentication)
	// For testing purposes, we're using a new token, but in reality this would be the user's token

	req, _ := http.NewRequest("PATCH", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID), bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
}

// Helper to find the index of a user ID in a slice
func indexOf(userID int, slice []int) int {
	for i, id := range slice {
		if id == userID {
			return i
		}
	}
	return -1
}

// TestStyleMatchOrdering runs the style match ordering test suite
func TestStyleMatchOrdering(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}
	suite.Run(t, new(StyleMatchOrderingTestSuite))
}
