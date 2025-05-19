package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMediaService is a mock implementation of MediaService
type MockMediaService struct {
	mock.Mock
}

// UploadMedia implements MediaService interface
func (m *MockMediaService) UploadMedia(userID int, fileHeader, thumbnailHeader media.UploadedFile) (*media.Media, error) {
	args := m.Called(userID, fileHeader, thumbnailHeader)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*media.Media), args.Error(1)
}

// Helper function to create a multipart request with file uploads
func createMultipartRequest(t *testing.T, fileContent, thumbnailContent []byte) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// Add file
	part, err := writer.CreateFormFile("file", "test.jpg")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, bytes.NewReader(fileContent))
	if err != nil {
		return nil, err
	}

	// Add thumbnail
	part, err = writer.CreateFormFile("thumbnail", "thumbnail.jpg")
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, bytes.NewReader(thumbnailContent))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "/api/media", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add user_id to the context
	ctx := context.WithValue(req.Context(), "user_id", 123)
	req = req.WithContext(ctx)

	return req, nil
}

func TestMediaHandler_UploadMedia_Success(t *testing.T) {
	// Create mock service
	mockService := new(MockMediaService)

	// Setup expected return values
	mockService.On("UploadMedia", 123, mock.AnythingOfType("*media.FileHeaderWrapper"), mock.AnythingOfType("*media.FileHeaderWrapper")).
		Return(&media.Media{
			ID:           42,
			URL:          "https://example.com/media/42.jpg",
			ThumbnailURL: "https://example.com/media/42_thumb.jpg",
		}, nil)

	// Create handler with mock service
	handler := NewMediaHandler(mockService, 10, 100)

	// Create test request
	fileContent := []byte("fake image content")
	thumbnailContent := []byte("fake thumbnail content")
	req, err := createMultipartRequest(t, fileContent, thumbnailContent)
	assert.NoError(t, err)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.UploadMedia(rr, req)

	// Check status code
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse response
	var response MediaResponse
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Verify response
	assert.Equal(t, 42, response.ID)
	assert.Equal(t, "https://example.com/media/42.jpg", response.URL)
	assert.Equal(t, "https://example.com/media/42_thumb.jpg", response.ThumbnailURL)

	// Verify service was called
	mockService.AssertExpectations(t)
}

func TestMediaHandler_UploadMedia_Unauthorized(t *testing.T) {
	// Create mock service
	mockService := new(MockMediaService)

	// Create handler with mock service
	handler := NewMediaHandler(mockService, 10, 100)

	// Create test request without user_id in context
	fileContent := []byte("fake image content")
	thumbnailContent := []byte("fake thumbnail content")
	req, err := createMultipartRequest(t, fileContent, thumbnailContent)
	assert.NoError(t, err)

	// Remove user_id from context
	req = req.WithContext(context.Background())

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call the handler
	handler.UploadMedia(rr, req)

	// Check status code and error message
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Unauthorized")

	// Verify service was not called
	mockService.AssertNotCalled(t, "UploadMedia")
}

func TestMediaHandler_UploadMedia_ServiceErrors(t *testing.T) {
	tests := []struct {
		name          string
		serviceErr    error
		expectedCode  int
		expectedError string
	}{
		{
			name:          "Invalid file type error",
			serviceErr:    media.ErrInvalidFileType,
			expectedCode:  http.StatusBadRequest,
			expectedError: "Invalid file type",
		},
		{
			name:          "File too big error",
			serviceErr:    media.ErrFileTooBig,
			expectedCode:  http.StatusRequestEntityTooLarge,
			expectedError: "File too large",
		},
		{
			name:          "Generic error",
			serviceErr:    errors.New("some internal error"),
			expectedCode:  http.StatusInternalServerError,
			expectedError: "Internal server error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMediaService)

			// Setup expected return values
			mockService.On("UploadMedia", 123, mock.AnythingOfType("*media.FileHeaderWrapper"), mock.AnythingOfType("*media.FileHeaderWrapper")).
				Return(nil, tc.serviceErr)

			// Create handler with mock service
			handler := NewMediaHandler(mockService, 10, 100)

			// Create test request
			fileContent := []byte("fake image content")
			thumbnailContent := []byte("fake thumbnail content")
			req, err := createMultipartRequest(t, fileContent, thumbnailContent)
			assert.NoError(t, err)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call the handler
			handler.UploadMedia(rr, req)

			// Check status code and error message
			assert.Equal(t, tc.expectedCode, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedError)

			// Verify service was called
			mockService.AssertExpectations(t)
		})
	}
}

func TestMediaHandler_ConcurrentUploads(t *testing.T) {
	// Create mock service that sleeps to simulate work
	mockService := new(MockMediaService)

	var serviceMu sync.Mutex
	activeServiceCalls := 0
	maxServiceCalls := 0

	// Setup success return for all calls with a sleep to simulate work
	mockService.On("UploadMedia", 123, mock.AnythingOfType("*media.FileHeaderWrapper"), mock.AnythingOfType("*media.FileHeaderWrapper")).
		Run(func(args mock.Arguments) {
			// Count active calls in the service method execution
			serviceMu.Lock()
			activeServiceCalls++
			if activeServiceCalls > maxServiceCalls {
				maxServiceCalls = activeServiceCalls
			}
			serviceMu.Unlock()

			// Sleep to simulate work being done
			time.Sleep(1 * time.Second)

			serviceMu.Lock()
			activeServiceCalls--
			serviceMu.Unlock()
		}).
		Return(&media.Media{
			ID:           42,
			URL:          "https://example.com/media/42.jpg",
			ThumbnailURL: "https://example.com/media/42_thumb.jpg",
		}, nil)

	// Create handler with a limit of 3 concurrent uploads
	maxConcurrent := 3
	handler := NewMediaHandler(mockService, maxConcurrent, 100)

	// Create sample content
	fileContent := []byte("fake image content")
	thumbnailContent := []byte("fake thumbnail content")

	// Prepare for concurrent testing
	var wg sync.WaitGroup
	// We'll try with more requests than allowed concurrently
	numRequests := maxConcurrent * 2

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create new request for each goroutine
			req, err := createMultipartRequest(t, fileContent, thumbnailContent)
			if err != nil {
				t.Error(err)
				return
			}

			rr := httptest.NewRecorder()

			// Call the handler - semaphore is acquired inside this method
			handler.UploadMedia(rr, req)

			// Check status
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that we never exceeded the max concurrent limit by checking
	// the maximum number of simultaneous service method calls
	assert.LessOrEqual(t, maxServiceCalls, maxConcurrent,
		"Too many concurrent uploads processed: got %d, expected maximum %d",
		maxServiceCalls, maxConcurrent)

	// Verify service was called the expected number of times
	mockService.AssertNumberOfCalls(t, "UploadMedia", numRequests)
}

func TestMediaHandler_MissingFiles(t *testing.T) {
	// Create mock service
	mockService := new(MockMediaService)

	// Create handler with mock service
	handler := NewMediaHandler(mockService, 10, 100)

	// Test case: missing file
	t.Run("Missing main file", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Only add thumbnail
		part, err := writer.CreateFormFile("thumbnail", "thumbnail.jpg")
		assert.NoError(t, err)
		_, err = io.Copy(part, bytes.NewReader([]byte("thumbnail content")))
		assert.NoError(t, err)

		writer.Close()

		req, err := http.NewRequest("POST", "/api/media", body)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Add user_id to context
		ctx := context.WithValue(req.Context(), "user_id", 123)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.UploadMedia(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Could not get file")
	})

	// Test case: missing thumbnail
	t.Run("Missing thumbnail", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Only add main file
		part, err := writer.CreateFormFile("file", "main.jpg")
		assert.NoError(t, err)
		_, err = io.Copy(part, bytes.NewReader([]byte("file content")))
		assert.NoError(t, err)

		writer.Close()

		req, err := http.NewRequest("POST", "/api/media", body)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Add user_id to context
		ctx := context.WithValue(req.Context(), "user_id", 123)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.UploadMedia(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Could not get thumbnail")
	})
}
