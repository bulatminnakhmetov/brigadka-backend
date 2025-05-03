package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/search"
)

// MockSearchService is a mock implementation of SearchService for testing
type MockSearchService struct {
	searchProfilesFn func(ctx context.Context, req search.ProfileSearchRequest) (*search.ProfileSearchResponse, error)
}

// SearchProfiles calls the mock function
func (m *MockSearchService) SearchProfiles(ctx context.Context, req search.ProfileSearchRequest) (*search.ProfileSearchResponse, error) {
	return m.searchProfilesFn(ctx, req)
}

func TestSearchHandler_SearchProfiles(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		requestBody        interface{}
		mockResponse       *search.ProfileSearchResponse
		mockError          error
		expectedStatusCode int
		validateResponse   func(t *testing.T, response *bytes.Buffer)
	}{
		{
			name:               "Invalid method",
			method:             http.MethodGet,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:               "Invalid request body",
			method:             http.MethodPost,
			requestBody:        "invalid json",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Invalid age range",
			method: http.MethodPost,
			requestBody: search.ProfileSearchRequest{
				AgeMin: intPtr(30),
				AgeMax: intPtr(20),
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Service returns invalid params error",
			method:             http.MethodPost,
			requestBody:        search.ProfileSearchRequest{},
			mockError:          ErrInvalidSearchParams,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "Service returns other error",
			method:             http.MethodPost,
			requestBody:        search.ProfileSearchRequest{},
			mockError:          errors.New("database error"),
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:   "Successful search",
			method: http.MethodPost,
			requestBody: search.ProfileSearchRequest{
				FullName: "John",
				Limit:    10,
			},
			mockResponse: &search.ProfileSearchResponse{
				Results: []search.ProfileSearchResult{
					{
						ProfileID:    1,
						UserID:       101,
						FullName:     "John Doe",
						ActivityType: "improv",
						Description:  "Improv enthusiast",
					},
				},
				TotalCount:  1,
				CurrentPage: 1,
				TotalPages:  1,
				PageSize:    10,
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, response *bytes.Buffer) {
				var result search.ProfileSearchResponse
				err := json.NewDecoder(response).Decode(&result)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if len(result.Results) != 1 {
					t.Errorf("Expected 1 result, got %d", len(result.Results))
				}
				if result.Results[0].FullName != "John Doe" {
					t.Errorf("Expected name 'John Doe', got '%s'", result.Results[0].FullName)
				}
			},
		},
		{
			name:   "Default limit applied",
			method: http.MethodPost,
			requestBody: search.ProfileSearchRequest{
				FullName: "John",
				Limit:    0, // Should be set to default
			},
			mockResponse: &search.ProfileSearchResponse{
				Results:     []search.ProfileSearchResult{},
				TotalCount:  0,
				CurrentPage: 1,
				TotalPages:  0,
				PageSize:    20, // Default limit
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, response *bytes.Buffer) {
				var result search.ProfileSearchResponse
				err := json.NewDecoder(response).Decode(&result)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.PageSize != 20 {
					t.Errorf("Expected default page size 20, got %d", result.PageSize)
				}
			},
		},
		{
			name:   "Maximum limit applied",
			method: http.MethodPost,
			requestBody: search.ProfileSearchRequest{
				FullName: "John",
				Limit:    500, // Should be capped at 100
			},
			mockResponse: &search.ProfileSearchResponse{
				Results:     []search.ProfileSearchResult{},
				TotalCount:  0,
				CurrentPage: 1,
				TotalPages:  0,
				PageSize:    100, // Max limit
			},
			expectedStatusCode: http.StatusOK,
			validateResponse: func(t *testing.T, response *bytes.Buffer) {
				var result search.ProfileSearchResponse
				err := json.NewDecoder(response).Decode(&result)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if result.PageSize != 100 {
					t.Errorf("Expected max page size 100, got %d", result.PageSize)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := &MockSearchService{
				searchProfilesFn: func(ctx context.Context, req search.ProfileSearchRequest) (*search.ProfileSearchResponse, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			// Create handler with mock service
			handler := NewSearchHandler(mockService)

			// Create request
			var reqBody []byte
			var err error

			switch body := tt.requestBody.(type) {
			case string:
				reqBody = []byte(body)
			default:
				reqBody, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest(tt.method, "/api/search/profiles", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.SearchProfiles(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, rr.Code)
			}

			// Validate response if needed
			if tt.validateResponse != nil {
				tt.validateResponse(t, rr.Body)
			}
		})
	}
}

func TestSearchHandler_SearchProfilesGet(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		queryParams        map[string]string
		queryArrayParams   map[string][]string
		mockResponse       *search.ProfileSearchResponse
		mockError          error
		expectedStatusCode int
		validateResponse   func(t *testing.T, response *bytes.Buffer)
		validateRequest    func(t *testing.T, req search.ProfileSearchRequest)
	}{
		{
			name:               "Invalid method",
			method:             http.MethodPost,
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:   "Basic search",
			method: http.MethodGet,
			queryParams: map[string]string{
				"full_name": "John",
				"city_id":   "1",
				"gender":    "male",
			},
			mockResponse: &search.ProfileSearchResponse{
				Results:     []search.ProfileSearchResult{},
				TotalCount:  0,
				CurrentPage: 1,
				TotalPages:  0,
				PageSize:    20,
			},
			expectedStatusCode: http.StatusOK,
			validateRequest: func(t *testing.T, req search.ProfileSearchRequest) {
				if req.FullName != "John" {
					t.Errorf("Expected full_name='John', got '%s'", req.FullName)
				}
				if req.CityID == nil || *req.CityID != 1 {
					t.Errorf("Expected city_id=1, got %v", req.CityID)
				}
				if req.Gender != "male" {
					t.Errorf("Expected gender='male', got '%s'", req.Gender)
				}
			},
		},
		{
			name:   "Age range",
			method: http.MethodGet,
			queryParams: map[string]string{
				"age_min": "20",
				"age_max": "30",
			},
			mockResponse: &search.ProfileSearchResponse{
				Results: []search.ProfileSearchResult{},
			},
			expectedStatusCode: http.StatusOK,
			validateRequest: func(t *testing.T, req search.ProfileSearchRequest) {
				if req.AgeMin == nil || *req.AgeMin != 20 {
					t.Errorf("Expected age_min=20, got %v", req.AgeMin)
				}
				if req.AgeMax == nil || *req.AgeMax != 30 {
					t.Errorf("Expected age_max=30, got %v", req.AgeMax)
				}
			},
		},
		{
			name:   "Improv search",
			method: http.MethodGet,
			queryParams: map[string]string{
				"activity_type":           "improv",
				"improv_goal":             "Career",
				"improv_looking_for_team": "true",
			},
			queryArrayParams: map[string][]string{
				"improv_style": {"Short Form", "Long Form"},
			},
			mockResponse: &search.ProfileSearchResponse{
				Results: []search.ProfileSearchResult{},
			},
			expectedStatusCode: http.StatusOK,
			validateRequest: func(t *testing.T, req search.ProfileSearchRequest) {
				if req.ActivityType != "improv" {
					t.Errorf("Expected activity_type='improv', got '%s'", req.ActivityType)
				}
				if req.ImprovGoal != "Career" {
					t.Errorf("Expected improv_goal='Career', got '%s'", req.ImprovGoal)
				}
				if req.ImprovLookingForTeam == nil || *req.ImprovLookingForTeam != true {
					t.Errorf("Expected improv_looking_for_team=true, got %v", req.ImprovLookingForTeam)
				}
				if len(req.ImprovStyles) != 2 || req.ImprovStyles[0] != "Short Form" || req.ImprovStyles[1] != "Long Form" {
					t.Errorf("Expected improv_styles=['Short Form', 'Long Form'], got %v", req.ImprovStyles)
				}
			},
		},
		{
			name:   "Music search",
			method: http.MethodGet,
			queryParams: map[string]string{
				"activity_type": "music",
			},
			queryArrayParams: map[string][]string{
				"music_genre":      {"rock", "jazz"},
				"music_instrument": {"guitar", "piano"},
			},
			mockResponse: &search.ProfileSearchResponse{
				Results: []search.ProfileSearchResult{},
			},
			expectedStatusCode: http.StatusOK,
			validateRequest: func(t *testing.T, req search.ProfileSearchRequest) {
				if req.ActivityType != "music" {
					t.Errorf("Expected activity_type='music', got '%s'", req.ActivityType)
				}
				if len(req.MusicGenres) != 2 || req.MusicGenres[0] != "rock" || req.MusicGenres[1] != "jazz" {
					t.Errorf("Expected music_genres=['rock', 'jazz'], got %v", req.MusicGenres)
				}
				if len(req.MusicInstruments) != 2 || req.MusicInstruments[0] != "guitar" || req.MusicInstruments[1] != "piano" {
					t.Errorf("Expected music_instruments=['guitar', 'piano'], got %v", req.MusicInstruments)
				}
			},
		},
		{
			name:   "Service error",
			method: http.MethodGet,
			queryParams: map[string]string{
				"full_name": "John",
			},
			mockError:          errors.New("database error"),
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:   "Pagination",
			method: http.MethodGet,
			queryParams: map[string]string{
				"limit":  "25",
				"offset": "50",
			},
			mockResponse: &search.ProfileSearchResponse{
				Results:     []search.ProfileSearchResult{},
				TotalCount:  0,
				CurrentPage: 3,
				TotalPages:  0,
				PageSize:    25,
			},
			expectedStatusCode: http.StatusOK,
			validateRequest: func(t *testing.T, req search.ProfileSearchRequest) {
				if req.Limit != 25 {
					t.Errorf("Expected limit=25, got %d", req.Limit)
				}
				if req.Offset != 50 {
					t.Errorf("Expected offset=50, got %d", req.Offset)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRequest search.ProfileSearchRequest

			// Create mock service
			mockService := &MockSearchService{
				searchProfilesFn: func(ctx context.Context, req search.ProfileSearchRequest) (*search.ProfileSearchResponse, error) {
					capturedRequest = req
					return tt.mockResponse, tt.mockError
				},
			}

			// Create handler with mock service
			handler := NewSearchHandler(mockService)

			// Create request with query parameters
			req := httptest.NewRequest(tt.method, "/api/search/profiles", nil)
			q := req.URL.Query()

			// Add regular query params
			for key, value := range tt.queryParams {
				q.Add(key, value)
			}

			// Add array query params
			for key, values := range tt.queryArrayParams {
				for _, value := range values {
					q.Add(key, value)
				}
			}

			req.URL.RawQuery = q.Encode()

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.SearchProfilesGet(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatusCode, rr.Code)
			}

			// Validate the request that was passed to the service
			if tt.validateRequest != nil && tt.mockError == nil {
				tt.validateRequest(t, capturedRequest)
			}

			// Validate response if needed
			if tt.validateResponse != nil {
				tt.validateResponse(t, rr.Body)
			}
		})
	}
}

// Helper function to create integer pointers
func intPtr(i int) *int {
	return &i
}
