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

// ProfileSearchTestSuite defines a test suite for profile search functionality
type ProfileSearchTestSuite struct {
	suite.Suite
	appUrl      string
	testDirPath string
	authToken   string // Token for the main test user that will perform searches
}

// TestProfile contains profile data for testing
type TestProfile struct {
	UserID         int
	AuthToken      string
	FullName       string
	Birthday       time.Time
	Age            int // For easier age-based testing
	Gender         string
	CityID         int
	Bio            string
	Goal           string
	ImprovStyles   []string
	LookingForTeam bool
	HasAvatar      bool
	HasVideo       bool
	CreatedAt      time.Time
}

// ProfileTemplate defines a template for creating test profiles
type ProfileTemplate struct {
	FullName       string
	BirthYear      int
	Gender         string
	CityID         int
	Bio            string
	Goal           string
	ImprovStyles   []string
	LookingForTeam bool
	HasAvatar      bool
	HasVideo       bool
}

// SetupSuite prepares the test environment before all tests
func (s *ProfileSearchTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	s.testDirPath = "testdata/search"
	os.MkdirAll(s.testDirPath, 0755)

	// Register a user who will perform searches
	authToken, _ := s.registerTestUser(s.T())
	s.authToken = authToken
}

// TearDownSuite cleans up after all tests
func (s *ProfileSearchTestSuite) TearDownSuite() {
	os.RemoveAll(s.testDirPath)
}

// Register a test user and return auth token and user ID
func (s *ProfileSearchTestSuite) registerTestUser(t *testing.T) (string, int) {
	// Create unique test credentials
	email := fmt.Sprintf("search_test_%d@example.com", time.Now().UnixNano())
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

// Helper to create test media files
func (s *ProfileSearchTestSuite) createTestMedia(t *testing.T, authToken string, withAvatar bool, withVideo bool) (int, []int) {
	var avatarID int
	var videoIDs []int

	if withAvatar || withVideo {
		// Create test image for avatar
		testImagePath := fmt.Sprintf("%s/avatar_%d.jpg", s.testDirPath, time.Now().UnixNano())
		testThumbnailPath := fmt.Sprintf("%s/thumb_%d.jpg", s.testDirPath, time.Now().UnixNano())
		createTestImage(testImagePath)
		createTestImage(testThumbnailPath)

		// Create test video
		testVideoPath := fmt.Sprintf("%s/video_%d.mp4", s.testDirPath, time.Now().UnixNano())
		createTestVideo(testVideoPath)

		if withAvatar {
			// Upload avatar
			avatarID = s.uploadMedia(t, authToken, testImagePath, testThumbnailPath)
		}

		if withVideo {
			// Upload video
			videoID := s.uploadMedia(t, authToken, testVideoPath, testThumbnailPath)
			videoIDs = append(videoIDs, videoID)
		}
	}

	return avatarID, videoIDs
}

// Helper to upload media and return media ID
func (s *ProfileSearchTestSuite) uploadMedia(t *testing.T, authToken, filePath, thumbnailPath string) int {
	req, err := createMultipartRequest(fmt.Sprintf("%s/api/media", s.appUrl), "file", filePath, "thumbnail", thumbnailPath, authToken)
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var mediaResponse struct {
		ID int `json:"id"`
	}
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse)
	assert.NoError(t, err)

	return mediaResponse.ID
}

// Returns standard profile templates for testing
func (s *ProfileSearchTestSuite) getStandardProfileTemplates() []ProfileTemplate {
	return []ProfileTemplate{
		{
			FullName:       "Alice Johnson",
			BirthYear:      1995,
			Gender:         "female",
			CityID:         1, // Moscow
			Bio:            "Improv performer with 5 years experience",
			Goal:           "career",
			ImprovStyles:   []string{"shortform", "longform"},
			LookingForTeam: true,
			HasAvatar:      true,
			HasVideo:       true,
		},
		{
			FullName:       "Bob Smith",
			BirthYear:      1988,
			Gender:         "male",
			CityID:         1, // Moscow
			Bio:            "Just starting with improv",
			Goal:           "hobby",
			ImprovStyles:   []string{"shortform"},
			LookingForTeam: true,
			HasAvatar:      true,
			HasVideo:       false,
		},
		{
			FullName:       "Carol Davis",
			BirthYear:      1990,
			Gender:         "female",
			CityID:         2, // Saint Petersburg
			Bio:            "Looking for experimental improv teams",
			Goal:           "career",
			ImprovStyles:   []string{"longform"},
			LookingForTeam: true,
			HasAvatar:      false,
			HasVideo:       true,
		},
		{
			FullName:       "David Wilson",
			BirthYear:      1982,
			Gender:         "male",
			CityID:         2, // Saint Petersburg
			Bio:            "Experienced improviser",
			Goal:           "career",
			ImprovStyles:   []string{"longform"},
			LookingForTeam: false,
			HasAvatar:      true,
			HasVideo:       true,
		},
		{
			FullName:       "Eva Martinez",
			BirthYear:      1998,
			Gender:         "female",
			CityID:         1, // Moscow
			Bio:            "New to improv",
			Goal:           "hobby",
			ImprovStyles:   []string{"shortform"},
			LookingForTeam: false,
			HasAvatar:      false,
			HasVideo:       false,
		},
	}
}

// Create test profiles based on templates
func (s *ProfileSearchTestSuite) createTestProfiles(t *testing.T, templates []ProfileTemplate) ([]TestProfile, time.Time) {
	testProfiles := make([]TestProfile, len(templates))

	for i, p := range templates {
		// Register a user for this profile
		authToken, userID := s.registerTestUser(t)

		// Upload media if needed
		avatarID, videoIDs := s.createTestMedia(t, authToken, p.HasAvatar, p.HasVideo)

		// Calculate birthday from birth year (Jan 1 of that year for simplicity)
		birthday := time.Date(p.BirthYear, time.January, 1, 0, 0, 0, 0, time.UTC)
		age := time.Now().Year() - birthday.Year()

		// Create profile
		profileData := map[string]interface{}{
			"user_id":          userID,
			"full_name":        p.FullName,
			"birthday":         birthday.Format("2006-01-02"),
			"gender":           p.Gender,
			"city_id":          p.CityID,
			"bio":              p.Bio,
			"goal":             p.Goal,
			"improv_styles":    p.ImprovStyles,
			"looking_for_team": p.LookingForTeam,
		}

		if p.HasAvatar {
			profileData["avatar"] = avatarID
		}

		if p.HasVideo {
			profileData["videos"] = videoIDs
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

		var profileResp profile.ProfileResponse
		err = json.NewDecoder(resp.Body).Decode(&profileResp)
		assert.NoError(t, err)

		// Store test profile data
		testProfiles[i] = TestProfile{
			UserID:         userID,
			AuthToken:      authToken,
			FullName:       p.FullName,
			Birthday:       birthday,
			Age:            age,
			Gender:         p.Gender,
			CityID:         p.CityID,
			Bio:            p.Bio,
			Goal:           p.Goal,
			ImprovStyles:   p.ImprovStyles,
			LookingForTeam: p.LookingForTeam,
			HasAvatar:      p.HasAvatar,
			HasVideo:       p.HasVideo,
			CreatedAt:      profileResp.CreatedAt,
		}
	}

	if len(testProfiles) > 0 {
		// Use the earliest created profile's creation time for filtering
		return testProfiles, testProfiles[0].CreatedAt
	}

	return nil, time.Time{}
}

// Helper function to execute a search request
func (s *ProfileSearchTestSuite) executeSearch(filter map[string]interface{}) (*profile.SearchResponse, error) {
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

// Helper to create test images and videos
func createTestImage(path string) {
	// Create a minimal valid JPEG file
	data := []byte{
		0xFF, 0xD8, // SOI marker
		0xFF, 0xE0, 0x00, 0x10, // APP0 marker
		'J', 'F', 'I', 'F', 0x00, // JFIF identifier
		0x01, 0x01, // version
		0x00,                   // units
		0x00, 0x01, 0x00, 0x01, // density
		0x00, 0x00, // thumbnail
		0xFF, 0xDB, 0x00, 0x43, 0x00, // DQT marker
		// Quantization table (simplified)
		0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07,
		0x07, 0x07, 0x09, 0x09, 0x08, 0x0A, 0x0C, 0x14,
		0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12, 0x13,
		0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A,
		0x1C, 0x1C, 0x20, 0x24, 0x2E, 0x27, 0x20, 0x22,
		0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29, 0x2C,
		0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39,
		0x3D, 0x38, 0x32, 0x3C, 0x2E, 0x33, 0x34, 0x32,
		// Remainder of JPEG structure
		0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00,
		0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09,
		0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00, 0xD2, 0xCF, 0x20,
		0xFF, 0xD9,
	}
	os.WriteFile(path, data, 0644)
}

func createTestVideo(path string) {
	// Create a minimal MP4 file header
	data := []byte{
		0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p',
		'm', 'p', '4', '2', 0x00, 0x00, 0x00, 0x00,
		'm', 'p', '4', '2', 'i', 's', 'o', 'm',
		0x00, 0x00, 0x00, 0x08, 'f', 'r', 'e', 'e',
	}
	os.WriteFile(path, data, 0644)
}

// Tests for individual search filters

// TestSearchByFullName tests searching profiles by full name
func (s *ProfileSearchTestSuite) TestSearchByFullName() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Search for "Alice"
	filter := map[string]interface{}{
		"full_name":     "Alice",
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Profiles))
	assert.Equal(t, "Alice Johnson", result.Profiles[0].FullName)

	// Search for "Bob"
	filter["full_name"] = "Bob"
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Profiles))
	assert.Equal(t, "Bob Smith", result.Profiles[0].FullName)

	// Search for nonexistent name
	filter["full_name"] = "Nonexistent"
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result.Profiles))
}

// TestSearchByLookingForTeam tests searching profiles by looking_for_team flag
func (s *ProfileSearchTestSuite) TestSearchByLookingForTeam() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	lookingTrue := true
	lookingFalse := false

	// Find profiles looking for team
	filter := map[string]interface{}{
		"looking_for_team": lookingTrue,
		"created_after":    createdAfter,
		"page":             1,
		"page_size":        10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles))

	// Find profiles not looking for team
	filter["looking_for_team"] = lookingFalse
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))
}

// TestSearchByAge tests searching profiles by age range
func (s *ProfileSearchTestSuite) TestSearchByAge() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	minAge := 25
	maxAge := 35

	// Find profiles within age range
	filter := map[string]interface{}{
		"age_min":       minAge,
		"age_max":       maxAge,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)

	// Verify ages of returned profiles
	for _, profile := range result.Profiles {
		age := time.Now().Year() - profile.Birthday.Year()
		assert.GreaterOrEqual(t, age, minAge)
		assert.LessOrEqual(t, age, maxAge)
	}

	// Test with just min age
	filter = map[string]interface{}{
		"age_min":       30,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err = s.executeSearch(filter)
	assert.NoError(t, err)

	for _, profile := range result.Profiles {
		age := time.Now().Year() - profile.Birthday.Year()
		assert.GreaterOrEqual(t, age, 30)
	}
}

// TestSearchByCity tests searching profiles by city
func (s *ProfileSearchTestSuite) TestSearchByCity() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Find profiles in Moscow (city_id = 1)
	filter := map[string]interface{}{
		"city_id":       1,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Equal(t, 1, profile.CityID)
	}

	// Find profiles in Saint Petersburg (city_id = 2)
	filter["city_id"] = 2
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Equal(t, 2, profile.CityID)
	}
}

// TestSearchByAvatar tests searching profiles by avatar presence
func (s *ProfileSearchTestSuite) TestSearchByAvatar() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	hasAvatar := true

	// Find profiles with avatar
	filter := map[string]interface{}{
		"has_avatar":    hasAvatar,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.NotNil(t, profile.Avatar)
	}

	// Find profiles without avatar
	filter["has_avatar"] = false
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Nil(t, profile.Avatar)
	}
}

// TestSearchByVideo tests searching profiles by video presence
func (s *ProfileSearchTestSuite) TestSearchByVideo() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	hasVideo := true

	// Find profiles with video
	filter := map[string]interface{}{
		"has_video":     hasVideo,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Greater(t, len(profile.Videos), 0)
	}

	// Find profiles without video
	filter["has_video"] = false
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Equal(t, 0, len(profile.Videos))
	}
}

// TestCombinedFilters tests searching with multiple filters combined
func (s *ProfileSearchTestSuite) TestCombinedFilters() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Find female profiles in Moscow with avatar who are looking for a team
	lookingTrue := true
	filter := map[string]interface{}{
		"genders":          []string{"female"},
		"city_id":          1,
		"has_avatar":       true,
		"looking_for_team": lookingTrue,
		"created_after":    createdAfter,
		"page":             1,
		"page_size":        10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Profiles))
	assert.Equal(t, "Alice Johnson", result.Profiles[0].FullName)

	// Find male profiles with career goal and video
	filter = map[string]interface{}{
		"genders":       []string{"male"},
		"goal":          "career",
		"has_video":     true,
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Profiles))
	assert.Equal(t, "David Wilson", result.Profiles[0].FullName)

	// Find profiles younger than 30 with shortform style
	filter = map[string]interface{}{
		"age_max":       30,
		"improv_styles": []string{"shortform"},
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(result.Profiles), 1)
}

// TestPagination tests that pagination works correctly
func (s *ProfileSearchTestSuite) TestPagination() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Get first page with 2 items per page
	filter := map[string]interface{}{
		"created_after": createdAfter,
		"page":          1,
		"page_size":     2,
	}

	result1, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result1.Profiles))
	assert.Equal(t, 1, result1.Page)
	assert.Equal(t, 2, result1.PageSize)
	assert.Equal(t, 5, result1.TotalCount) // Total of 5 profiles in the system

	// Get second page
	filter["page"] = 2
	result2, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result2.Profiles))
	assert.Equal(t, 2, result2.Page)

	// Get third page (should have 1 profile)
	filter["page"] = 3
	result3, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result3.Profiles))
	assert.Equal(t, 3, result3.Page)

	// Ensure we got different profiles on different pages
	page1Names := []string{result1.Profiles[0].FullName, result1.Profiles[1].FullName}
	page2Names := []string{result2.Profiles[0].FullName, result2.Profiles[1].FullName}
	page3Names := []string{result3.Profiles[0].FullName}

	// Check that all pages contain different profiles
	for _, name1 := range page1Names {
		for _, name2 := range page2Names {
			assert.NotEqual(t, name1, name2, "Found same profile on different pages")
		}
		for _, name3 := range page3Names {
			assert.NotEqual(t, name1, name3, "Found same profile on different pages")
		}
	}

	for _, name2 := range page2Names {
		for _, name3 := range page3Names {
			assert.NotEqual(t, name2, name3, "Found same profile on different pages")
		}
	}
}

// TestSearchByCreatedAfter tests filtering profiles by creation time
func (s *ProfileSearchTestSuite) TestSearchByCreatedAfter() {
	t := s.T()

	_, firstCreationTime := s.createTestProfiles(t, s.getStandardProfileTemplates())

	time.Sleep(1 * time.Second)

	_, secondCreationTime := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Test 1: Should find only the new profile when filtering by creation time
	filter := map[string]interface{}{
		"created_after": secondCreationTime,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(result.Profiles))

	// Test 2: Should find original profiles when using initial creation time
	filter["created_after"] = firstCreationTime

	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(result.Profiles)) // 5 before + 5 after

	// Test 3: Should find no profiles when using a future timestamp
	futureTime := time.Now().Add(1 * time.Hour)
	filter["created_after"] = futureTime

	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result.Profiles))
}

// Add these new test methods after the existing tests:

// TestSearchByMultipleGenders tests searching profiles with multiple gender options (OR logic)
func (s *ProfileSearchTestSuite) TestSearchByMultipleGenders() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Find profiles with male OR female gender - should match all profiles
	filter := map[string]interface{}{
		"genders":       []string{"male", "female"},
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(result.Profiles)) // Should find all 5 test profiles

	// Verify that each profile has one of the requested genders
	for _, profile := range result.Profiles {
		assert.Contains(t, []string{"male", "female"}, profile.Gender)
	}

	// Find profiles with just male gender
	filter["genders"] = []string{"male"}
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))

	for _, profile := range result.Profiles {
		assert.Equal(t, "male", profile.Gender)
	}
}

// TestSearchByMultipleGoals tests searching profiles with multiple goal options (OR logic)
func (s *ProfileSearchTestSuite) TestSearchByMultipleGoals() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Find profiles with either hobby OR career goals
	filter := map[string]interface{}{
		"goals":         []string{"career"},
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles)) // Should find 4 profiles with hobby or career goals

	// Verify that each returned profile has one of the requested goals
	for _, profile := range result.Profiles {
		assert.Contains(t, []string{"hobby", "career"}, profile.Goal)
	}

	// Find profiles with hobby, career goals (should include all 5 profiles)
	filter["goals"] = []string{"hobby", "career"}
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(result.Profiles)) // Should find all 5 profiles
}

// TestSearchByMultipleImprovStyles tests searching profiles with multiple improv style options (AND logic)
func (s *ProfileSearchTestSuite) TestSearchByMultipleImprovStyles() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Find profiles with BOTH shortform AND longform styles
	filter := map[string]interface{}{
		"improv_styles": []string{"shortform", "longform"},
		"created_after": createdAfter,
		"page":          1,
		"page_size":     10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)

	// Should find only Alice who has both styles
	assert.Equal(t, 1, len(result.Profiles))
	assert.Equal(t, "Alice Johnson", result.Profiles[0].FullName)

	// Verify Alice has both requested styles
	var hasShortform, hasLongform bool
	for _, style := range result.Profiles[0].ImprovStyles {
		if style == "shortform" {
			hasShortform = true
		}
		if style == "longform" {
			hasLongform = true
		}
	}
	assert.True(t, hasShortform, "Profile should have shortform style")
	assert.True(t, hasLongform, "Profile should have longform style")

	// Test with just one style - should return all profiles with that style
	filter["improv_styles"] = []string{"shortform"}
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result.Profiles))

	for _, profile := range result.Profiles {
		hasStyle := false
		for _, style := range profile.ImprovStyles {
			if style == "shortform" {
				hasStyle = true
				break
			}
		}
		assert.True(t, hasStyle, "Profile should have shortform style")
	}
}

// TestComplexCombinedFilters tests searching with a complex combination of multiple filters
func (s *ProfileSearchTestSuite) TestComplexCombinedFilters() {
	t := s.T()

	// Create test profiles just for this test
	_, createdAfter := s.createTestProfiles(t, s.getStandardProfileTemplates())

	// Complex search for:
	// - Profiles looking for a team
	// - With either hobby or career goals
	// - With shortform style
	// - Either male or female gender
	// - In Moscow (city_id = 1)
	lookingTrue := true
	filter := map[string]interface{}{
		"looking_for_team": lookingTrue,
		"goals":            []string{"hobby", "career"},
		"improv_styles":    []string{"shortform"},
		"genders":          []string{"male", "female"},
		"city_id":          1,
		"created_after":    createdAfter,
		"page":             1,
		"page_size":        10,
	}

	result, err := s.executeSearch(filter)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Profiles))

	// Should return profiles that match ALL these criteria
	for _, profile := range result.Profiles {
		assert.Equal(t, 1, profile.CityID)
		assert.True(t, profile.LookingForTeam)
		assert.Contains(t, []string{"hobby", "career"}, profile.Goal)
		assert.Contains(t, []string{"male", "female"}, profile.Gender)

		hasShortform := false
		for _, style := range profile.ImprovStyles {
			if style == "shortform" {
				hasShortform = true
				break
			}
		}
		assert.True(t, hasShortform)
	}

	// Even more specific search: add an age constraint
	filter["age_min"] = 25
	filter["age_max"] = 30
	result, err = s.executeSearch(filter)
	assert.NoError(t, err)

	for _, profile := range result.Profiles {
		age := time.Now().Year() - profile.Birthday.Year()
		assert.GreaterOrEqual(t, age, 25)
		assert.LessOrEqual(t, age, 30)
	}
}

// TestProfileSearch runs the profile search test suite
func TestProfileSearch(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}
	suite.Run(t, new(ProfileSearchTestSuite))
}
