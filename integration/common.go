package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
)

// TestUser represents a registered test user
type TestUser struct {
	Email        string
	Password     string
	UserID       int
	Token        string
	RefreshToken string
}

// registerUser registers a new user with a random email
func RegisterUser(appURL string) (*TestUser, error) {
	email := fmt.Sprintf("test_user_%d@example.com", time.Now().UnixNano())
	password := "TestPassword123!"

	reqBody, _ := json.Marshal(auth.RegisterRequest{
		Email:    email,
		Password: password,
	})

	resp, err := http.Post(appURL+"/api/auth/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp auth.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	testToken := fmt.Sprintf("test-verification-token-%d", authResp.UserID)

	req, _ := http.NewRequest("GET", appURL+"/api/auth/verify-email?token="+testToken, nil)

	client := &http.Client{}
	resp, err = client.Do(req)
	defer resp.Body.Close()

	if err != nil || resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("email verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	resp, err = http.Post(appURL+"/api/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	return &TestUser{
		Email:        email,
		Password:     password,
		UserID:       authResp.UserID,
		Token:        authResp.Token,
		RefreshToken: authResp.RefreshToken,
	}, nil
}
