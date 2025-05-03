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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// MediaIntegrationTestSuite defines a set of integration tests for media operations
type MediaIntegrationTestSuite struct {
	suite.Suite
	appUrl        string
	authToken     string
	testImagePath string
	testVideoPath string
}

// SetupSuite prepares the test environment before running all tests
func (s *MediaIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	// Set paths for test media files
	s.testImagePath = filepath.Join("testdata", "test_image.jpg")
	s.testVideoPath = filepath.Join("testdata", "test_video.mp4")

	// Create test directory if it doesn't exist
	os.MkdirAll("testdata", 0755)

	// Generate test image file
	s.createTestImage()

	// Generate test video file
	s.createTestVideo()

	// Register a test user and get authentication token
	s.authToken = s.registerTestUser()
}

// TearDownSuite cleans up after all tests have run
func (s *MediaIntegrationTestSuite) TearDownSuite() {
	// Clean up test files
	os.Remove(s.testImagePath)
	os.Remove(s.testVideoPath)
	os.Remove("testdata")
}

// Helper function to generate a test image
func (s *MediaIntegrationTestSuite) createTestImage() {
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
	_ = os.WriteFile(s.testImagePath, data, 0644)
}

// Helper function to generate a test video
func (s *MediaIntegrationTestSuite) createTestVideo() {
	// Create a very simple MP4 file header
	data := []byte{
		0x00, 0x00, 0x00, 0x18, 'f', 't', 'y', 'p',
		'm', 'p', '4', '2', 0x00, 0x00, 0x00, 0x00,
		'm', 'p', '4', '2', 'i', 's', 'o', 'm',
		0x00, 0x00, 0x00, 0x08, 'f', 'r', 'e', 'e',
	}
	_ = os.WriteFile(s.testVideoPath, data, 0644)
}

// Helper function to generate a unique email
func generateTestEmail() string {
	return fmt.Sprintf("test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// Helper function to register a test user and return the auth token
func (s *MediaIntegrationTestSuite) registerTestUser() string {
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
	if err != nil {
		s.T().Fatalf("Failed to register test user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		s.T().Fatalf("Failed to register test user. Status: %d", resp.StatusCode)
	}

	var authResponse auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	if err != nil {
		s.T().Fatalf("Failed to decode auth response: %v", err)
	}

	return authResponse.Token
}

// Helper function to create a multipart request with a file
func createMultipartRequest(url, fieldName, filePath string, authToken string) (*http.Request, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
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

// TestUploadMediaImage tests uploading an image
func (s *MediaIntegrationTestSuite) TestUploadMediaImage() {
	t := s.T()

	req, err := createMultipartRequest(s.appUrl+"/api/media/upload", "file", s.testImagePath, s.authToken)
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response content
	var mediaResponse media.MediaResponse
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse)
	assert.NoError(t, err)

	// Verify the response contains a media ID
	assert.Greater(t, mediaResponse.ID, 0, "Media ID should be positive")
}

// TestUploadMediaVideo tests uploading a video
func (s *MediaIntegrationTestSuite) TestUploadMediaVideo() {
	t := s.T()

	req, err := createMultipartRequest(s.appUrl+"/api/media/upload", "file", s.testVideoPath, s.authToken)
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response content
	var mediaResponse media.MediaResponse
	err = json.NewDecoder(resp.Body).Decode(&mediaResponse)
	assert.NoError(t, err)

	// Verify the response contains a media ID
	assert.Greater(t, mediaResponse.ID, 0, "Media ID should be positive")
}

// TestUploadMediaNoAuth tests uploading without authentication
func (s *MediaIntegrationTestSuite) TestUploadMediaNoAuth() {
	t := s.T()

	req, err := createMultipartRequest(s.appUrl+"/api/media/upload", "file", s.testImagePath, "")
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return status 401 Unauthorized")
}

// TestUploadInvalidFile tests uploading an invalid file type
func (s *MediaIntegrationTestSuite) TestUploadInvalidFile() {
	t := s.T()

	// Create a temporary invalid file
	invalidFilePath := filepath.Join("testdata", "invalid_file.txt")
	err := os.WriteFile(invalidFilePath, []byte("This is not an image or video"), 0644)
	assert.NoError(t, err)
	defer os.Remove(invalidFilePath)

	req, err := createMultipartRequest(s.appUrl+"/api/media/upload", "file", invalidFilePath, s.authToken)
	assert.NoError(t, err)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return bad request for invalid file type
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request")
}

// TestUploadNoFile tests submitting a request without a file
func (s *MediaIntegrationTestSuite) TestUploadNoFile() {
	t := s.T()

	// Create a request without a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req, err := http.NewRequest("POST", s.appUrl+"/api/media/upload", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return bad request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request")
}

// TestMediaIntegration runs the media integration test suite
func TestMediaIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(MediaIntegrationTestSuite))
}
