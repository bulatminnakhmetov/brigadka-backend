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

// VerificationIntegrationTestSuite defines tests for the email verification flow
type VerificationIntegrationTestSuite struct {
	suite.Suite
	appUrl      string
	testToken   string // Used in test environment
	bearerToken string // JWT token for authenticated requests
	userID      int
	testEmail   string
}

// SetupSuite prepares the test environment
func (s *VerificationIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}

	// Register a test user to use for verification tests
	s.testEmail = fmt.Sprintf("test_verification_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
	s.registerTestUser()
}

// Helper function to generate a unique email
func generateTestEmail() string {
	return fmt.Sprintf("test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
}

// Register a test user for verification tests
func (s *VerificationIntegrationTestSuite) registerTestUser() {
	t := s.T()

	// Create test credentials
	testPassword := "TestPassword123!"

	// Register the user
	registerData := auth.RegisterRequest{
		Email:    s.testEmail,
		Password: testPassword,
	}

	registerJSON, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check if registration was successful
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	// Get tokens from response
	var authResp auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	assert.NoError(t, err)

	s.testToken = fmt.Sprintf("test-verification-token-%d", authResp.UserID)

	// Save the bearer token for authenticated requests
	s.bearerToken = authResp.Token
	s.userID = authResp.UserID
}

// TestVerificationStatus checks if a newly registered user is marked as unverified
func (s *VerificationIntegrationTestSuite) TestVerificationStatus() {
	t := s.T()

	// Create request to check verification status
	req, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verification-status", nil)
	req.Header.Set("Authorization", "Bearer "+s.bearerToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the response
	var statusResp auth.VerificationStatusResponse
	err = json.NewDecoder(resp.Body).Decode(&statusResp)
	assert.NoError(t, err)

	// A newly registered user should not be verified yet
	assert.False(t, statusResp.Verified)
}

// TestVerificationStatusUnauthorized tests access to verification status without auth
func (s *VerificationIntegrationTestSuite) TestVerificationStatusUnauthorized() {
	t := s.T()

	// Create request without auth token
	req, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verification-status", nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should be unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestResendVerification tests the resend verification endpoint
func (s *VerificationIntegrationTestSuite) TestResendVerification() {
	t := s.T()

	// Create request to resend verification email
	resendData := auth.ResendVerificationRequest{
		IgnoreCooldown: true, // Ignore cooldown for testing purposes
	}

	resendJSON, _ := json.Marshal(resendData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/resend-verification", bytes.NewBuffer(resendJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.bearerToken) // Add auth header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the response
	var resendResp auth.VerificationResponse
	err = json.NewDecoder(resp.Body).Decode(&resendResp)
	assert.NoError(t, err)

	// Check that the response indicates success
	assert.True(t, resendResp.Success)
	assert.Contains(t, resendResp.Message, "Verification email sent")
}

// TestResendVerificationUnauthorized tests resend verification without auth
func (s *VerificationIntegrationTestSuite) TestResendVerificationUnauthorized() {
	t := s.T()

	// Create request without auth token
	resendData := auth.ResendVerificationRequest{}

	resendJSON, _ := json.Marshal(resendData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/resend-verification", bytes.NewBuffer(resendJSON))
	req.Header.Set("Content-Type", "application/json")
	// No auth header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should be unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestResendVerificationTooSoon tests that the system prevents sending
// verification emails too frequently
func (s *VerificationIntegrationTestSuite) TestResendVerificationTooSoon() {
	t := s.T()

	// Try to resend immediately after the previous test
	resendData := auth.ResendVerificationRequest{}

	resendJSON, _ := json.Marshal(resendData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/resend-verification", bytes.NewBuffer(resendJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.bearerToken) // Add auth header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should get an error as we're trying to send too soon
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	assert.Contains(t, buf.String(), "email was sent recently")
}

// TestVerifyEmail tests the email verification process
func (s *VerificationIntegrationTestSuite) TestVerifyEmail() {
	t := s.T()

	// Create GET request with token as query parameter
	req, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify-email?token="+s.testToken, nil)
	req.Header.Set("Authorization", "Bearer "+s.bearerToken) // Add auth header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the response
	var verifyResp auth.VerificationResponse
	err = json.NewDecoder(resp.Body).Decode(&verifyResp)
	assert.NoError(t, err)

	// Check that the verification was successful
	assert.True(t, verifyResp.Success)
	assert.Contains(t, verifyResp.Message, "Email verified successfully")

	// Check verification status again to confirm user is now verified
	statusReq, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verification-status", nil)
	statusReq.Header.Set("Authorization", "Bearer "+s.bearerToken)

	statusResp, err := client.Do(statusReq)
	assert.NoError(t, err)
	defer statusResp.Body.Close()

	var updatedStatus auth.VerificationStatusResponse
	err = json.NewDecoder(statusResp.Body).Decode(&updatedStatus)
	assert.NoError(t, err)

	// User should now be verified
	assert.True(t, updatedStatus.Verified)

	// Check verification status with new token
	loginData := auth.LoginRequest{
		Email:    s.testEmail,
		Password: "TestPassword123!",
	}

	// Login with the test user to check if verification persists
	loginJSON, _ := json.Marshal(loginData)
	req, _ = http.NewRequest("POST", s.appUrl+"/api/auth/login", bytes.NewBuffer(loginJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check login was successful
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse response to get new token
	var loginResp auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	assert.NoError(t, err)

	newToken := loginResp.Token

	// Check verification status with new token
	statusReq, _ = http.NewRequest("GET", s.appUrl+"/api/auth/verification-status", nil)
	statusReq.Header.Set("Authorization", "Bearer "+newToken)

	statusResp, err = client.Do(statusReq)
	assert.NoError(t, err)
	defer statusResp.Body.Close()

	err = json.NewDecoder(statusResp.Body).Decode(&updatedStatus)
	assert.NoError(t, err)

	// User should still be verified after login with new token
	assert.True(t, updatedStatus.Verified)
}

// TestVerifyEmailUnauthorized tests verification without auth
func (s *VerificationIntegrationTestSuite) TestVerifyEmailUnauthorized() {
	t := s.T()

	// Create request without auth token
	req, _ := http.NewRequest("GET", s.appUrl+"/api/auth/verify-email?token="+s.testToken, nil)
	// No auth header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should be unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestInvalidVerificationToken tests verification with an invalid token
func (s *VerificationIntegrationTestSuite) TestInvalidVerificationToken() {
	t := s.T()

	// Register a new user to get a fresh token
	newEmail := generateTestEmail()
	registerData := auth.RegisterRequest{
		Email:    newEmail,
		Password: "TestPassword123!",
	}

	registerJSON, _ := json.Marshal(registerData)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)

	// Get the new user's token
	var authResp auth.AuthResponse
	err = json.NewDecoder(resp.Body).Decode(&authResp)
	assert.NoError(t, err)
	newUserToken := authResp.Token
	resp.Body.Close()

	// Try to verify with an invalid token
	req, _ = http.NewRequest("GET", s.appUrl+"/api/auth/verify-email?token=invalid-token-that-does-not-exist", nil)
	req.Header.Set("Authorization", "Bearer "+newUserToken) // Use the new user's token

	resp, err = client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should fail with unauthorized status for the token verification
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Read the error message
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	assert.Contains(t, buf.String(), "invalid verification token")
}

// TestVerificationIntegration runs the verification integration test suite
func TestVerificationIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(VerificationIntegrationTestSuite))
}
