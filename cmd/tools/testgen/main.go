package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

// ProfileData represents test profile data loaded from JSON
type ProfileData struct {
	FullName       string   `json:"full_name"`
	Gender         string   `json:"gender"`
	Bio            string   `json:"bio"`
	Goal           string   `json:"goal"`
	ImprovStyles   []string `json:"improv_styles"`
	LookingForTeam bool     `json:"looking_for_team"`
	CityID         int      `json:"city_id"`
	// Age will be used to calculate birthday
	Age int `json:"age"`
}

// TestUser represents a registered test user
type TestUser struct {
	Email        string
	Password     string
	UserID       int
	Token        string
	RefreshToken string
}

// AuthRequest for login/register requests
type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse for login/register responses
type AuthResponse struct {
	UserID       int    `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// MediaResponse for media upload responses
type MediaResponse struct {
	ID           int    `json:"id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// ProfileCreateRequest represents the request body for creating a profile
type ProfileCreateRequest struct {
	UserID         int      `json:"user_id"`
	FullName       string   `json:"full_name"`
	Birthday       string   `json:"birthday"` // Format: "YYYY-MM-DD"
	Gender         string   `json:"gender"`
	CityID         int      `json:"city_id"`
	Bio            string   `json:"bio"`
	Goal           string   `json:"goal"`
	ImprovStyles   []string `json:"improv_styles"`
	LookingForTeam bool     `json:"looking_for_team"`
	Avatar         int      `json:"avatar"`
	Videos         []int    `json:"videos"`
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Get API URL from environment or use default
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080/api"
	}

	// Read profile data from JSON file
	profiles, err := loadProfileData("testdata/test_profiles.json")
	if err != nil {
		log.Fatalf("Failed to load profile data: %v", err)
	}

	log.Printf("Loaded %d profile configurations", len(profiles))

	// Register users and create profiles
	for i, profileData := range profiles {
		log.Printf("Creating profile %d/%d: %s", i+1, len(profiles), profileData.FullName)

		// Register user with random email
		user, err := registerUser(apiURL, i+1)
		if err != nil {
			log.Printf("Failed to register user: %v", err)
			panic(err)
		}

		// Calculate birthday from age
		birthday := calculateBirthday(profileData.Age)

		// Get gender-specific avatar directory
		avatarDir := "testdata/avatars/male"
		if profileData.Gender == "female" {
			avatarDir = "testdata/avatars/female"
		}

		// Upload avatar
		avatarID, err := uploadRandomMedia(apiURL, user.Token, avatarDir)
		if err != nil {
			log.Printf("Failed to upload avatar: %v", err)
			continue
		}

		// Upload 3 random videos
		var videoIDs []int
		for j := 0; j < 3; j++ {
			videoID, err := uploadRandomMedia(apiURL, user.Token, "testdata/videos/thumbnail")
			if err != nil {
				log.Printf("Failed to upload video %d: %v", j+1, err)
				continue
			}
			videoIDs = append(videoIDs, videoID)
		}

		// Create profile
		err = createProfile(apiURL, user.Token, ProfileCreateRequest{
			UserID:         user.UserID,
			FullName:       profileData.FullName,
			Birthday:       birthday,
			Gender:         profileData.Gender,
			CityID:         profileData.CityID,
			Bio:            profileData.Bio,
			Goal:           profileData.Goal,
			ImprovStyles:   profileData.ImprovStyles,
			LookingForTeam: profileData.LookingForTeam,
			Avatar:         avatarID,
			Videos:         videoIDs,
		})
		if err != nil {
			log.Printf("Failed to create profile: %v", err)
			continue
		}

		log.Printf("Successfully created profile for %s", profileData.FullName)

		// Small delay to avoid overwhelming the server
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Profile seeding completed")
}

// loadProfileData loads profile data from JSON file
func loadProfileData(filePath string) ([]ProfileData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open profile data file: %w", err)
	}
	defer file.Close()

	var profiles []ProfileData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&profiles); err != nil {
		return nil, fmt.Errorf("failed to decode profile data: %w", err)
	}

	return profiles, nil
}

// registerUser registers a new user with a random email
func registerUser(apiURL string, index int) (*TestUser, error) {
	email := fmt.Sprintf("test_user_%d_%d@example.com", index, time.Now().UnixNano())
	password := "TestPassword123!"

	reqBody, _ := json.Marshal(AuthRequest{
		Email:    email,
		Password: password,
	})

	resp, err := http.Post(apiURL+"/auth/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}

	testToken := fmt.Sprintf("test-verification-token-%d", authResp.UserID)

	req, _ := http.NewRequest("GET", apiURL+"/auth/verify-email?token="+testToken, nil)

	client := &http.Client{}
	resp, err = client.Do(req)
	defer resp.Body.Close()

	if err != nil || resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("email verification failed with status %d: %s", resp.StatusCode, string(body))
	}

	resp, err = http.Post(apiURL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
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

// uploadRandomMedia uploads a random media file from the specified directory
func uploadRandomMedia(apiURL, token, dirPath string) (int, error) {
	// Get a random file from the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	if len(files) == 0 {
		return 0, fmt.Errorf("no files found in directory %s", dirPath)
	}

	// Select a random file
	randIndex := rand.Intn(len(files))
	file := files[randIndex]
	filePath := filepath.Join(dirPath, file.Name())

	// Create a multipart request
	responseBody, err := uploadFile(apiURL, token, filePath, filePath)
	if err != nil {
		return 0, err
	}

	// Parse response to get media ID
	var mediaResp MediaResponse
	if err := json.Unmarshal(responseBody, &mediaResp); err != nil {
		return 0, fmt.Errorf("failed to parse media response: %w", err)
	}

	return mediaResp.ID, nil
}

// uploadFile uploads a file with thumbnail to the media endpoint
func uploadFile(apiURL, token, filePath, thumbnailPath string) ([]byte, error) {
	// Create a buffer to write our multipart form to
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	// Add thumbnail
	thumbnail, err := os.Open(thumbnailPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open thumbnail %s: %w", thumbnailPath, err)
	}
	defer thumbnail.Close()

	thumbPart, err := writer.CreateFormFile("thumbnail", filepath.Base(thumbnailPath))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(thumbPart, thumbnail)
	if err != nil {
		return nil, err
	}

	// Close the writer
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// Create request
	req, err := http.NewRequest("POST", apiURL+"/media", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("media upload failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

// createProfile creates a profile for the user
func createProfile(apiURL, token string, profile ProfileCreateRequest) error {
	reqBody, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL+"/profiles", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("profile creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// calculateBirthday calculates a birthday date from an age
func calculateBirthday(age int) string {
	now := time.Now()

	// Generate random month and day
	month := rand.Intn(12) + 1
	day := rand.Intn(28) + 1 // Safe for all months

	// Calculate birth year based on age
	birthYear := now.Year() - age

	// If this year's birthday hasn't happened yet, subtract one more year
	birthDate := time.Date(birthYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if birthDate.After(now) {
		birthYear--
	}

	return fmt.Sprintf("%d-%02d-%02d", birthYear, month, day)
}

// createDefaultThumbnail creates a simple JPEG thumbnail file
func createDefaultThumbnail(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

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

	return os.WriteFile(path, data, 0644)
}

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}
