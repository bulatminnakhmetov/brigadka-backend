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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite defines a set of integration tests for authentication
type AuthIntegrationTestSuite struct {
	suite.Suite
	appUrl string
}

// SetupSuite prepares the test environment before running all tests
func (s *AuthIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}
}

// Helper function to generate a unique email
func generateTestEmail() string {
	return fmt.Sprintf("test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// TestRegister tests the user registration endpoint
func (s *AuthIntegrationTestSuite) TestRegister() {
	t := s.T()

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

	// Check response status
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return status 201 Created")

	// Check response content
	var authResponse auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	assert.NoError(t, err)

	// Verify the response contains required fields
	assert.Greater(t, authResponse.UserID, 0, "User ID should be positive")
	assert.NotEmpty(t, authResponse.Token, "Token should not be empty")
	assert.NotEmpty(t, authResponse.RefreshToken, "Refresh token should not be empty")
}

// TestRegisterDuplicate tests registration with an existing email
func (s *AuthIntegrationTestSuite) TestRegisterDuplicate() {
	t := s.T()

	// Create unique test credentials
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register the first user
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
	resp.Body.Close()

	// Try to register with the same email
	req, _ = http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	duplicateResp, err := client.Do(req)
	assert.NoError(t, err)
	defer duplicateResp.Body.Close()

	// Should return conflict error
	assert.Equal(t, http.StatusConflict, duplicateResp.StatusCode, "Should return status 409 Conflict")
}

// TestLogin tests the login endpoint
func (s *AuthIntegrationTestSuite) TestLogin() {
	t := s.T()

	// Create unique test credentials
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register a user first
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
	resp.Body.Close()

	// Now attempt to login
	loginData := auth.LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, loginResp.StatusCode, "Should return status 200 OK")

	// Check response content
	var authResponse auth.AuthResponse
	err = json.NewDecoder(loginResp.Body).Decode(&authResponse)
	assert.NoError(t, err)

	// Verify the response contains required fields
	assert.Greater(t, authResponse.UserID, 0, "User ID should be positive")
	assert.NotEmpty(t, authResponse.Token, "Token should not be empty")
	assert.NotEmpty(t, authResponse.RefreshToken, "Refresh token should not be empty")
}

// TestLoginInvalidCredentials tests login with incorrect credentials
func (s *AuthIntegrationTestSuite) TestLoginInvalidCredentials() {
	t := s.T()

	// Create unique test credentials
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register a user first
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
	resp.Body.Close()

	// Try to login with incorrect password
	loginData := auth.LoginRequest{
		Email:    testEmail,
		Password: "WrongPassword123",
	}

	loginJSON, _ := json.Marshal(loginData)
	loginReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	assert.NoError(t, err)
	defer loginResp.Body.Close()

	// Should return unauthorized
	assert.Equal(t, http.StatusUnauthorized, loginResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestRefreshToken tests the token refresh endpoint
func (s *AuthIntegrationTestSuite) TestRefreshToken() {
	t := s.T()

	// Register a user to get initial tokens
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register
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

	var initialAuth auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&initialAuth)
	assert.NoError(t, err)
	resp.Body.Close()

	// Now use the refresh token to get a new token
	refreshData := auth.RefreshRequest{
		RefreshToken: initialAuth.RefreshToken,
	}

	refreshJSON, _ := json.Marshal(refreshData)
	refreshReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/refresh", bytes.NewBuffer(refreshJSON))
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshResp, err := client.Do(refreshReq)
	assert.NoError(t, err)
	defer refreshResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, refreshResp.StatusCode, "Should return status 200 OK")

	// Check response content
	var refreshedAuth auth.AuthResponse
	err = json.NewDecoder(refreshResp.Body).Decode(&refreshedAuth)
	assert.NoError(t, err)

	// Verify the response contains required fields and different token
	assert.Equal(t, initialAuth.UserID, refreshedAuth.UserID, "User ID should be the same")
	assert.NotEqual(t, initialAuth.Token, refreshedAuth.Token, "New token should be different")
	assert.NotEqual(t, initialAuth.RefreshToken, refreshedAuth.RefreshToken, "New refresh token should be different")
}

// TestRefreshTokenInvalid tests using an invalid refresh token
func (s *AuthIntegrationTestSuite) TestRefreshTokenInvalid() {
	t := s.T()

	// Use an invalid refresh token
	refreshData := auth.RefreshRequest{
		RefreshToken: "invalid.refresh.token",
	}

	refreshJSON, _ := json.Marshal(refreshData)
	refreshReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/refresh", bytes.NewBuffer(refreshJSON))
	refreshReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	refreshResp, err := client.Do(refreshReq)
	assert.NoError(t, err)
	defer refreshResp.Body.Close()

	// Should return unauthorized
	assert.Equal(t, http.StatusUnauthorized, refreshResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestVerifyToken tests the token verification endpoint
func (s *AuthIntegrationTestSuite) TestVerifyToken() {
	t := s.T()

	// Register a user to get a token
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register
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

	var authResponse auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	assert.NoError(t, err)
	resp.Body.Close()

	// Verify the token
	verifyReq, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer "+authResponse.Token)

	verifyResp, err := client.Do(verifyReq)
	assert.NoError(t, err)
	defer verifyResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, verifyResp.StatusCode, "Should return status 200 OK")

	// Read and parse response body
	var response map[string]string
	err = json.NewDecoder(verifyResp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "valid", response["status"], "Token status should be valid")
}

// TestVerifyTokenInvalid tests verifying an invalid token
func (s *AuthIntegrationTestSuite) TestVerifyTokenInvalid() {
	t := s.T()

	// Use an invalid token
	verifyReq, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify", nil)
	verifyReq.Header.Set("Authorization", "Bearer invalid.token.here")

	client := &http.Client{}
	verifyResp, err := client.Do(verifyReq)
	assert.NoError(t, err)
	defer verifyResp.Body.Close()

	// Should return unauthorized
	assert.Equal(t, http.StatusUnauthorized, verifyResp.StatusCode, "Should return status 401 Unauthorized")
}

// TestProtectedEndpoint tests accessing a protected endpoint with a valid token
func (s *AuthIntegrationTestSuite) TestProtectedEndpoint() {
	t := s.T()

	// Register a user to get a token
	testEmail := generateTestEmail()
	testPassword := "TestPassword123!"

	// Register
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

	var authResponse auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResponse)
	assert.NoError(t, err)
	resp.Body.Close()

	// Access a protected endpoint (using /api/chats as an example)
	protectedReq, _ := http.NewRequest("GET", s.appUrl+"/api/chats", nil)
	protectedReq.Header.Set("Authorization", "Bearer "+authResponse.Token)

	protectedResp, err := client.Do(protectedReq)
	assert.NoError(t, err)
	defer protectedResp.Body.Close()

	// Check response status - should allow access even if no chats exist yet
	assert.Equal(t, http.StatusOK, protectedResp.StatusCode, "Should return status 200 OK")
}

// TestProtectedEndpointUnauthorized tests accessing a protected endpoint without a token
func (s *AuthIntegrationTestSuite) TestProtectedEndpointUnauthorized() {
	t := s.T()

	// Try to access a protected endpoint without authentication
	req, _ := http.NewRequest("GET", s.appUrl+"/api/chats", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return status 401 Unauthorized")
}

// TestAuthIntegration runs the auth integration test suite
func TestAuthIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(AuthIntegrationTestSuite))
}
