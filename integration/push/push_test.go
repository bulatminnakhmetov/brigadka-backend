package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/push"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// PushIntegrationTestSuite defines a set of integration tests for push token operations
type PushIntegrationTestSuite struct {
	suite.Suite
	appUrl    string
	authToken string
}

// SetupSuite prepares the test environment before running all tests
func (s *PushIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	// Register a test user and get authentication token
	s.authToken = s.registerTestUser()
}

// Helper function to generate a unique email
func generateTestEmail() string {
	return fmt.Sprintf("test_push_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// Helper function to register a test user and return the auth token
func (s *PushIntegrationTestSuite) registerTestUser() string {
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

// Helper function to generate a unique device token
func generateUniqueToken() string {
	return fmt.Sprintf("test_token_%d_%d", os.Getpid(), time.Now().UnixNano())
}

// TestRegisterPushToken tests registering a push notification token
func (s *PushIntegrationTestSuite) TestRegisterPushToken() {
	t := s.T()

	// Create test token data
	tokenData := push.TokenRequest{
		Token:    generateUniqueToken(),
		Platform: "ios", // Test with iOS platform
		DeviceID: "test-device-id-ios",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Parse response body
	var responseData map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, "success", responseData["status"], "Response status should be 'success'")
}

// TestRegisterAndroidPushToken tests registering an Android push notification token
func (s *PushIntegrationTestSuite) TestRegisterAndroidPushToken() {
	t := s.T()

	// Create test token data for Android
	tokenData := push.TokenRequest{
		Token:    generateUniqueToken(),
		Platform: "android",
		DeviceID: "test-device-id-android",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Parse response body
	var responseData map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, "success", responseData["status"], "Response status should be 'success'")
}

// TestRegisterInvalidPlatform tests registering a token with an invalid platform
func (s *PushIntegrationTestSuite) TestRegisterInvalidPlatform() {
	t := s.T()

	// Create test token data with invalid platform
	tokenData := push.TokenRequest{
		Token:    generateUniqueToken(),
		Platform: "windows", // This should be invalid
		DeviceID: "test-device-id",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response - should be Bad Request
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should return status 500 Internal Server Error for invalid platform")
}

// TestRegisterPushTokenNoAuth tests registering a token without authentication
func (s *PushIntegrationTestSuite) TestRegisterPushTokenNoAuth() {
	t := s.T()

	// Create test token data
	tokenData := push.TokenRequest{
		Token:    generateUniqueToken(),
		Platform: "ios",
		DeviceID: "test-device-id",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request without auth token
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response - should be unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return status 401 Unauthorized")
}

// TestRegisterEmptyToken tests registering an empty token
func (s *PushIntegrationTestSuite) TestRegisterEmptyToken() {
	t := s.T()

	// Create test token data with empty token
	tokenData := push.TokenRequest{
		Token:    "", // Empty token
		Platform: "ios",
		DeviceID: "test-device-id",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response - should be Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for empty token")
}

// TestRegisterNoPlatform tests registering a token without a platform
func (s *PushIntegrationTestSuite) TestRegisterNoPlatform() {
	t := s.T()

	// Create test token data with no platform
	tokenData := push.TokenRequest{
		Token:    generateUniqueToken(),
		Platform: "", // Empty platform
		DeviceID: "test-device-id",
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response - should be Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for empty platform")
}

// TestUnregisterPushToken tests unregistering a push notification token
func (s *PushIntegrationTestSuite) TestUnregisterPushToken() {
	t := s.T()

	// First, register a token
	token := generateUniqueToken()
	tokenData := push.TokenRequest{
		Token:    token,
		Platform: "ios",
		DeviceID: "test-device-id-unregister",
	}

	// Register the token first
	tokenJSON, _ := json.Marshal(tokenData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Now unregister the token
	req, err = http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Parse response body
	var responseData map[string]string
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, "success", responseData["status"], "Response status should be 'success'")
}

// TestUnregisterEmptyToken tests unregistering an empty token
func (s *PushIntegrationTestSuite) TestUnregisterEmptyToken() {
	t := s.T()

	// Create test token data with empty token
	tokenData := push.TokenRequest{
		Token: "", // Empty token
	}

	// Marshal token data to JSON
	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	// Create request
	req, err := http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response - should be Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return status 400 Bad Request for empty token")
}

// TestPushIntegration runs the push integration test suite
func TestPushIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(PushIntegrationTestSuite))
}
