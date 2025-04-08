package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var baseURL string

func init() {
	baseURL = os.Getenv("TEST_APP_URL")
	if baseURL == "" {
		baseURL = "http://test-app:8080"
	}
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func TestAuthAPI(t *testing.T) {
	// Ждем, пока сервер запустится
	time.Sleep(2 * time.Second)

	t.Run("Register new user", func(t *testing.T) {
		creds := credentials{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := makeRequest("/register", creds)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response authResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "User registered", response.Message)
	})

	t.Run("Register duplicate user", func(t *testing.T) {
		creds := credentials{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := makeRequest("/register", creds)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response authResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Email already registered", response.Error)
	})

	t.Run("Login with correct credentials", func(t *testing.T) {
		creds := credentials{
			Email:    "test@example.com",
			Password: "password123",
		}

		resp, err := makeRequest("/login", creds)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response authResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.Token)
	})

	t.Run("Login with wrong password", func(t *testing.T) {
		creds := credentials{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		resp, err := makeRequest("/login", creds)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var response authResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid email or password", response.Error)
	})

	t.Run("Login with non-existent email", func(t *testing.T) {
		creds := credentials{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		resp, err := makeRequest("/login", creds)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		var response authResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Equal(t, "User not found", response.Error)
	})
}

func makeRequest(endpoint string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}
