package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/media"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ProfileIntegrationTestSuite defines a set of integration tests for profile operations
type ProfileIntegrationTestSuite struct {
	suite.Suite
	appUrl      string
	testDirPath string
}

// SetupSuite prepares the test environment before running all tests
func (s *ProfileIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	// Create test directory if it doesn't exist
	s.testDirPath = "testdata"
	os.MkdirAll(s.testDirPath, 0755)
}

// TearDownSuite cleans up after all tests have run
func (s *ProfileIntegrationTestSuite) TearDownSuite() {
	// Clean up test files
	os.RemoveAll(s.testDirPath)
}

// Generate a unique email for test user
func generateTestEmail() string {
	return fmt.Sprintf("profile_test_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// Register a test user and return auth token and user ID
func (s *ProfileIntegrationTestSuite) registerTestUser(t *testing.T) (string, int) {
	// Create unique test credentials
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Prepare registration request
	registerData := auth.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
	}

	registerJSON, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var authResponse auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	assert.NoError(t, err)

	return authResponse.Token, authResponse.UserID
}

// Helper function to create test files and upload to media service
func (s *ProfileIntegrationTestSuite) uploadTestMedia(t *testing.T, authToken string) (int, []int) {
	// Create test image files
	testImagePath := filepath.Join(s.testDirPath, fmt.Sprintf("avatar_%d.jpg", time.Now().UnixNano()))
	createTestJPEGFile(testImagePath)

	videoPath := filepath.Join(s.testDirPath, fmt.Sprintf("video_%d.mp4", time.Now().UnixNano()))
	createTestMP4File(videoPath)

	// Upload avatar
	avatarID, err := s.uploadFile(testImagePath, authToken)
	assert.NoError(t, err)

	// Upload two video files
	var mediaIDs []int
	for i := 0; i < 2; i++ {
		mediaID, err := s.uploadFile(videoPath, authToken)
		assert.NoError(t, err)
		mediaIDs = append(mediaIDs, mediaID)
	}

	return avatarID, mediaIDs
}

// Upload a file and return its media ID
func (s *ProfileIntegrationTestSuite) uploadFile(filePath string, authToken string) (int, error) {
	// Both main file and thumbnail are required now
	thumbnailPath := filepath.Join(s.testDirPath, fmt.Sprintf("thumb_%s", filepath.Base(filePath)))
	createTestJPEGFile(thumbnailPath)

	req, err := createMultipartRequest(s.appUrl+"/api/media", "file", filePath, "thumbnail", thumbnailPath, authToken)
	if err != nil {
		return 0, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to upload file, status: %d", resp.StatusCode)
	}

	var mediaResponse media.MediaResponse
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse)
	if err != nil {
		return 0, err
	}

	return mediaResponse.ID, nil
}

// Helper function to create a multipart request with a file and thumbnail
func createMultipartRequest(url, fileField, filePath, thumbField, thumbPath, authToken string) (*http.Request, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	thumb, err := os.Open(thumbPath)
	if err != nil {
		return nil, err
	}
	defer thumb.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add main file
	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	// Add thumbnail file
	thumbPart, err := writer.CreateFormFile(thumbField, filepath.Base(thumbPath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(thumbPart, thumb)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	return req, nil
}

// Helper function to create a test JPEG file
func createTestJPEGFile(path string) {
	// Create a simple 1x1 black JPEG file
	data := []byte{
		0xFF, 0xD8, // SOI marker
		0xFF, 0xE0, 0x00, 0x10, // APP0 marker
		'J', 'F', 'I', 'F', 0x00, // JFIF identifier
		0x01, 0x01, // version
		0x00,                   // units (0 = no units)
		0x00, 0x01, 0x00, 0x01, // X and Y densities
		0x00, 0x00, // thumbnail width/height
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
		// Rest of the JPEG structure
		0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00,
		0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09,
		0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00, 0xD2, 0xCF, 0x20,
		0xFF, 0xD9,
	}
	_ = os.WriteFile(path, data, 0644)
}

// Helper function to create a test MP4 file
func createTestMP4File(path string) {
	// Create a very simple MP4 file header
	data := []byte{
		0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p',
		'm', 'p', '4', '2', 0x00, 0x00, 0x00, 0x00,
		'm', 'p', '4', '2', 'i', 's', 'o', 'm',
		0x00, 0x00, 0x00, 0x08, 'f', 'r', 'e', 'e',
	}
	_ = os.WriteFile(path, data, 0644)
}

// TestCreateProfile tests creating a new profile
func (s *ProfileIntegrationTestSuite) TestCreateProfile() {
	t := s.T()

	// Register a new user for this test
	authToken, userID := s.registerTestUser(t)

	// Upload test media for this user
	avatarID, mediaIDs := s.uploadTestMedia(t, authToken)

	// Prepare create request
	createReqMap := map[string]interface{}{
		"user_id":          userID,
		"full_name":        "Test User",
		"birthday":         "1990-01-01", // Use string format instead of Date struct
		"gender":           "male",
		"city_id":          1,
		"bio":              "Test bio",
		"goal":             "hobby",
		"improv_styles":    []string{"shortform", "longform"},
		"looking_for_team": true,
		"avatar":           avatarID,
		"videos":           mediaIDs,
	}

	reqBody, _ := json.Marshal(createReqMap)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(reqBody))
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

	assert.Equal(t, "Test User", profileResp.FullName)
	assert.Equal(t, "1990-01-01", profileResp.Birthday.Format("2006-01-02"))
	assert.Equal(t, "male", profileResp.Gender)
	assert.Equal(t, 1, profileResp.CityID)
	assert.Equal(t, "Test bio", profileResp.Bio)
	assert.Equal(t, "hobby", profileResp.Goal)
	assert.Equal(t, []string{"shortform", "longform"}, profileResp.ImprovStyles)
	assert.True(t, profileResp.LookingForTeam)
	assert.Equal(t, avatarID, profileResp.Avatar.ID)
	assert.Equal(t, len(mediaIDs), len(profileResp.Videos))
}

// TestUpdateProfile tests updating a profile
func (s *ProfileIntegrationTestSuite) TestUpdateProfile() {
	t := s.T()

	// Register a new user for this test
	authToken, userID := s.registerTestUser(t)

	// Upload test media for this user
	avatarID, mediaIDs := s.uploadTestMedia(t, authToken)

	// First create a profile to update
	createReqMap := map[string]interface{}{
		"user_id":          userID,
		"full_name":        "Profile To Update",
		"birthday":         "1990-01-01",
		"gender":           "male",
		"city_id":          1,
		"bio":              "Original bio",
		"goal":             "hobby",
		"improv_styles":    []string{"shortform"},
		"looking_for_team": true,
	}

	reqBody, _ := json.Marshal(createReqMap)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Now update the created profile
	newFullName := "Updated User"
	newBio := "Updated bio"
	lookingForTeam := false
	newBirthday := "1995-05-05" // Use string format for date

	updateReqMap := map[string]interface{}{
		"full_name":        newFullName,
		"bio":              newBio,
		"looking_for_team": lookingForTeam,
		"birthday":         newBirthday,
		"avatar":           avatarID,
		"videos":           mediaIDs,
	}

	updateBody, _ := json.Marshal(updateReqMap)

	// Fix URL path - need to include user ID
	updateURL := fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID)
	updateReq, _ := http.NewRequest("PATCH", updateURL, bytes.NewBuffer(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+authToken)

	resp, err = client.Do(updateReq)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var profileResp profile.ProfileResponse
	err = json.NewDecoder(resp.Body).Decode(&profileResp)
	assert.NoError(t, err)

	assert.Equal(t, newFullName, profileResp.FullName)
	assert.Equal(t, newBio, profileResp.Bio)
	assert.Equal(t, lookingForTeam, profileResp.LookingForTeam)
	assert.Equal(t, newBirthday, profileResp.Birthday.Format("2006-01-02"))
	assert.NotNil(t, profileResp.Avatar)
	assert.Equal(t, avatarID, profileResp.Avatar.ID)
	assert.Equal(t, len(mediaIDs), len(profileResp.Videos))
}

// TestGetProfile tests retrieving a profile
func (s *ProfileIntegrationTestSuite) TestGetProfile() {
	t := s.T()

	// Register a new user for this test
	authToken, userID := s.registerTestUser(t)

	// Create a profile first to ensure we have something to retrieve
	createReqMap := map[string]interface{}{
		"user_id":          userID,
		"full_name":        "Test User For Get",
		"birthday":         "1990-01-01",
		"gender":           "male",
		"city_id":          1,
		"bio":              "Test bio for get",
		"goal":             "hobby",
		"improv_styles":    []string{"longform"},
		"looking_for_team": true,
	}

	reqBody, _ := json.Marshal(createReqMap)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Now get the profile
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID), nil)
	getReq.Header.Set("Authorization", "Bearer "+authToken)

	resp, err = client.Do(getReq)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var profileResp profile.ProfileResponse
	err = json.NewDecoder(resp.Body).Decode(&profileResp)
	assert.NoError(t, err)

	assert.Equal(t, "Test User For Get", profileResp.FullName)
	assert.Equal(t, "Test bio for get", profileResp.Bio)
}

// TestGetCatalogData tests retrieving catalog data
func (s *ProfileIntegrationTestSuite) TestGetCatalogData() {
	t := s.T()

	// Register a new user for this test (just for the auth token)
	authToken, _ := s.registerTestUser(t)

	// Test getting improv styles
	req, _ := http.NewRequest("GET", s.appUrl+"/api/profiles/catalog/improv-styles?lang=en", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test getting goals
	req, _ = http.NewRequest("GET", s.appUrl+"/api/profiles/catalog/improv-goals?lang=en", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test getting genders
	req, _ = http.NewRequest("GET", s.appUrl+"/api/profiles/catalog/genders?lang=en", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Test getting cities
	req, _ = http.NewRequest("GET", s.appUrl+"/api/profiles/catalog/cities", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var cities []profile.City
	err = json.NewDecoder(resp.Body).Decode(&cities)
	assert.NoError(t, err)
	assert.Greater(t, len(cities), 0)
}

// TestUpdateProfileWithInvalidData tests updating a profile with invalid data
func (s *ProfileIntegrationTestSuite) TestUpdateProfileWithInvalidData() {
	t := s.T()

	// Register a new user for this test
	authToken, userID := s.registerTestUser(t)

	// First create a valid profile
	createReqMap := map[string]interface{}{
		"user_id":          userID,
		"full_name":        "Test User For Invalid Update",
		"birthday":         "1990-01-01",
		"gender":           "male",
		"city_id":          1,
		"bio":              "Bio",
		"looking_for_team": true,
		"goal":             "hobby",
	}

	reqBody, _ := json.Marshal(createReqMap)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Try to update with invalid data
	invalidGender := "invalid_gender"
	updateReqMap := map[string]interface{}{
		"gender": invalidGender,
	}

	reqBody, _ = json.Marshal(updateReqMap)
	updateURL := fmt.Sprintf("%s/api/profiles/%d", s.appUrl, userID)
	req, _ = http.NewRequest("PATCH", updateURL, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestGetNonExistentProfile tests retrieving a profile that doesn't exist
func (s *ProfileIntegrationTestSuite) TestGetNonExistentProfile() {
	t := s.T()

	// Register a new user for this test
	authToken, _ := s.registerTestUser(t)

	// Use a high user ID that doesn't exist
	nonExistentID := 999999

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/profiles/%d", s.appUrl, nonExistentID), nil)
	req.Header.Set("Authorization", "Bearer "+authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestProfileIntegration runs the profile integration test suite
func TestProfileIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(ProfileIntegrationTestSuite))
}
