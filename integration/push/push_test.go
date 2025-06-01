package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/integration"
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
	user, err := integration.RegisterUser(s.appUrl)
	if err != nil {
		s.T().Fatalf("Failed to register test user: %v", err)
	}
	return user.Token
}

// Helper function to generate a unique device token
func generateUniqueToken() string {
	return fmt.Sprintf("test_token_%d_%d", os.Getpid(), time.Now().UnixNano())
}

// TestRegisterPushToken tests registering a push notification token
func (s *PushIntegrationTestSuite) TestRegisterPushToken() {
	t := s.T()

	// Create test token data
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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
	tokenData := push.RegisterTokenRequest{
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

// TestUnregisterOtherUserToken tests that a user cannot unregister another user's token
func (s *PushIntegrationTestSuite) TestUnregisterOtherUserToken() {
	t := s.T()

	// Register a second user
	secondUserToken := s.registerTestUser()

	// First, register a token for the second user
	token := generateUniqueToken()
	tokenData := push.RegisterTokenRequest{
		Token:    token,
		Platform: "ios",
		DeviceID: "test-device-id-second-user",
	}

	// Register token for second user
	tokenJSON, _ := json.Marshal(tokenData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secondUserToken) // Use second user's auth token

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Now try to unregister the token using the first user's credentials
	req, err = http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken) // Use first user's auth token

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should get an error response since the token doesn't belong to this user
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return error when trying to delete another user's token")

	// Verify the token still exists by having the second user delete it successfully
	req, err = http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secondUserToken) // Use second user's auth token

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should succeed for the correct user
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should allow the token owner to delete their token")
}

// TestUnregisterNonExistentToken tests unregistering a token that doesn't exist
func (s *PushIntegrationTestSuite) TestUnregisterNonExistentToken() {
	t := s.T()

	// Create request with a token that was never registered
	tokenData := push.RegisterTokenRequest{
		Token:    "non-existent-token-" + generateUniqueToken(),
		Platform: "ios",
		DeviceID: "test-device-id",
	}

	tokenJSON, err := json.Marshal(tokenData)
	assert.NoError(t, err)

	req, err := http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should get an error response since the token doesn't exist
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return error when trying to delete non-existent token")
}

// TestRegisterSameTokenDifferentUsers tests registering the same token for different users
func (s *PushIntegrationTestSuite) TestRegisterSameTokenDifferentUsers() {
	t := s.T()

	// Register a second user
	secondUserToken := s.registerTestUser()

	// Create a token
	token := generateUniqueToken()
	tokenData := push.RegisterTokenRequest{
		Token:    token,
		Platform: "ios",
		DeviceID: "test-device-id-shared",
	}

	// Register for first user
	tokenJSON, _ := json.Marshal(tokenData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Register same token for second user and it belongs to them now
	req, _ = http.NewRequest("POST", s.appUrl+"/api/push/register", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secondUserToken)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should allow registering the same token for different users")
	resp.Body.Close()

	// First user should not be able to unregister their instance of the token because it belongs to the second user now
	req, _ = http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.authToken)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()

	// Second user should also be able to unregister their instance of the token
	req, _ = http.NewRequest("DELETE", s.appUrl+"/api/push/unregister", bytes.NewBuffer(tokenJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secondUserToken)

	resp, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestPushIntegration runs the push integration test suite
func TestPushIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(PushIntegrationTestSuite))
}
