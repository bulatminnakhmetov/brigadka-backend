package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMessagingService implements MessagingService for testing
type MockMessagingService struct {
	mock.Mock
}

func (m *MockMessagingService) AddMessage(messageID string, chatID string, userID int, content string) (time.Time, error) {
	args := m.Called(messageID, chatID, userID, content)
	return args.Get(0).(time.Time), args.Error(1)
}

func (m *MockMessagingService) GetUserChatRooms(userID int) (map[string]struct{}, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]struct{}), args.Error(1)
}

func (m *MockMessagingService) GetChatParticipantsForBroadcast(chatID string) ([]int, error) {
	args := m.Called(chatID)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockMessagingService) CreateChat(ctx context.Context, chatID string, creatorID int, chatName string, participants []int) error {
	args := m.Called(ctx, chatID, creatorID, chatName, participants)
	return args.Error(0)
}

func (m *MockMessagingService) GetUserChats(userID int) ([]Chat, error) {
	args := m.Called(userID)
	return args.Get(0).([]Chat), args.Error(1)
}

func (m *MockMessagingService) GetChatDetails(chatID string, userID int) (*Chat, error) {
	args := m.Called(chatID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Chat), args.Error(1)
}

func (m *MockMessagingService) GetChatMessages(chatID string, userID int, limit int, offset int) ([]ChatMessage, error) {
	args := m.Called(chatID, userID, limit, offset)
	return args.Get(0).([]ChatMessage), args.Error(1)
}

func (m *MockMessagingService) IsUserInChat(userID int, chatID string) (bool, error) {
	args := m.Called(userID, chatID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMessagingService) AddParticipant(chatID string, userID int) error {
	args := m.Called(chatID, userID)
	return args.Error(0)
}

func (m *MockMessagingService) RemoveParticipant(chatID string, userID int) error {
	args := m.Called(chatID, userID)
	return args.Error(0)
}

func (m *MockMessagingService) AddReaction(reactionID string, messageID string, userID int, reactionCode string) error {
	args := m.Called(reactionID, messageID, userID, reactionCode)
	return args.Error(0)
}

func (m *MockMessagingService) GetChatIDForMessage(messageID string) (string, error) {
	args := m.Called(messageID)
	return args.String(0), args.Error(1)
}

func (m *MockMessagingService) RemoveReaction(messageID string, userID int, reactionCode string) error {
	args := m.Called(messageID, userID, reactionCode)
	return args.Error(0)
}

func (m *MockMessagingService) StoreTypingIndicator(userID int, chatID string) error {
	args := m.Called(userID, chatID)
	return args.Error(0)
}

func (m *MockMessagingService) GetChatParticipants(chatID string) ([]int, error) {
	args := m.Called(chatID)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockMessagingService) StoreReadReceipt(userID int, chatID string, messageID string) error {
	args := m.Called(userID, chatID, messageID)
	return args.Error(0)
}

// Test CreateChat handler
func TestHandler_CreateChat(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		userID         int
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "Success",
			requestBody:    `{"chat_id":"123","chat_name":"Test Chat","participants":[1,2,3]}`,
			userID:         1,
			serviceError:   nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Empty Participants",
			requestBody:    `{"chat_id":"123","chat_name":"Test Chat","participants":[]}`,
			userID:         1,
			serviceError:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Duplicate Chat ID",
			requestBody:    `{"chat_id":"123","chat_name":"Test Chat","participants":[1,2,3]}`,
			userID:         1,
			serviceError:   error(errPrimaryKey()),
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "Server Error",
			requestBody:    `{"chat_id":"123","chat_name":"Test Chat","participants":[1,2,3]}`,
			userID:         1,
			serviceError:   error(errGeneric()),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("CreateChat", mock.Anything, "123", tt.userID, "Test Chat", []int{1, 2, 3}).Return(tt.serviceError).Maybe()

			// Create handler
			handler := NewHandler(mockService)

			// Create request
			req, err := http.NewRequest("POST", "/chats", strings.NewReader(tt.requestBody))
			require.NoError(t, err)

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", tt.userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Call handler
			handler.CreateChat(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test GetUserChats handler
func TestHandler_GetUserChats(t *testing.T) {
	tests := []struct {
		name           string
		userID         int
		chats          []Chat
		serviceError   error
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			chats: []Chat{
				{ChatID: "123", ChatName: "Test Chat", CreatedAt: time.Now(), IsGroup: true, Participants: []int{1, 2, 3}},
			},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Service Error",
			userID:         1,
			chats:          []Chat{},
			serviceError:   error(errGeneric()),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("GetUserChats", tt.userID).Return(tt.chats, tt.serviceError)

			// Create handler
			handler := NewHandler(mockService)

			// Create request
			req, err := http.NewRequest("GET", "/chats", nil)
			require.NoError(t, err)

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", tt.userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Call handler
			handler.GetUserChats(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If success, verify response body
			if tt.expectedStatus == http.StatusOK {
				var response []Chat
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, len(tt.chats), len(response))
				for i := range tt.chats {
					assert.Equal(t, tt.chats[i].ChatID, response[i].ChatID)
					assert.Equal(t, tt.chats[i].ChatName, response[i].ChatName)
					assert.Equal(t, tt.chats[i].IsGroup, response[i].IsGroup)
					assert.Equal(t, tt.chats[i].Participants, response[i].Participants)
					assert.WithinDuration(t, tt.chats[i].CreatedAt, response[i].CreatedAt, time.Second)
				}
			}

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test GetChatDetails handler
func TestHandler_GetChatDetails(t *testing.T) {
	chatID := "123"

	tests := []struct {
		name           string
		userID         int
		chat           *Chat
		serviceError   error
		expectedStatus int
	}{
		{
			name:   "Success",
			userID: 1,
			chat: &Chat{
				ChatID:       chatID,
				ChatName:     "Test Chat",
				CreatedAt:    time.Now(),
				IsGroup:      true,
				Participants: []int{1, 2, 3},
			},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User Not In Chat",
			userID:         1,
			chat:           nil,
			serviceError:   error(errUserNotInChat()),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Service Error",
			userID:         1,
			chat:           nil,
			serviceError:   error(errGeneric()),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("GetChatDetails", chatID, tt.userID).Return(tt.chat, tt.serviceError)

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Get("/chats/{chatID}", handler.GetChatDetails)

			// Create request
			req, err := http.NewRequest("GET", "/chats/"+chatID, nil)
			require.NoError(t, err)

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", tt.userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If success, verify response body
			if tt.expectedStatus == http.StatusOK {
				var response Chat
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.chat.ChatID, response.ChatID)
				assert.Equal(t, tt.chat.ChatName, response.ChatName)
				assert.Equal(t, tt.chat.IsGroup, response.IsGroup)
				assert.Equal(t, tt.chat.Participants, response.Participants)
				assert.WithinDuration(t, tt.chat.CreatedAt, response.CreatedAt, time.Second)
			}

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test GetChatMessages handler
func TestHandler_GetChatMessages(t *testing.T) {
	chatID := "123"

	tests := []struct {
		name           string
		userID         int
		queryParams    string
		messages       []ChatMessage
		serviceError   error
		expectedStatus int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "Success Default Pagination",
			userID:         1,
			queryParams:    "",
			messages:       []ChatMessage{{MessageID: "m1", ChatID: chatID, SenderID: 2, Content: "Hello", SentAt: time.Now()}},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "Success With Pagination",
			userID:         1,
			queryParams:    "?limit=10&offset=20",
			messages:       []ChatMessage{{MessageID: "m1", ChatID: chatID, SenderID: 2, Content: "Hello", SentAt: time.Now()}},
			serviceError:   nil,
			expectedStatus: http.StatusOK,
			expectedLimit:  10,
			expectedOffset: 20,
		},
		{
			name:           "User Not In Chat",
			userID:         1,
			queryParams:    "",
			messages:       nil,
			serviceError:   error(errUserNotInChat()),
			expectedStatus: http.StatusNotFound,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "Service Error",
			userID:         1,
			queryParams:    "",
			messages:       nil,
			serviceError:   error(errGeneric()),
			expectedStatus: http.StatusInternalServerError,
			expectedLimit:  50,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("GetChatMessages", chatID, tt.userID, tt.expectedLimit, tt.expectedOffset).Return(tt.messages, tt.serviceError)

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Get("/chats/{chatID}/messages", handler.GetChatMessages)

			// Create request
			req, err := http.NewRequest("GET", "/chats/"+chatID+"/messages"+tt.queryParams, nil)
			require.NoError(t, err)

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", tt.userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If success, verify response body
			if tt.expectedStatus == http.StatusOK {
				var response []ChatMessage
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				require.Equal(t, len(tt.messages), len(response))
				for i := range tt.messages {
					assert.Equal(t, tt.messages[i].ChatID, response[i].ChatID)
					assert.Equal(t, tt.messages[i].Content, response[i].Content)
					assert.Equal(t, tt.messages[i].SenderID, response[i].SenderID)
					assert.Equal(t, tt.messages[i].MessageID, response[i].MessageID)
					assert.WithinDuration(t, tt.messages[i].SentAt, response[i].SentAt, time.Second)
				}
			}

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test SendMessage handler
func TestSendMessage(t *testing.T) {
	chatID := "123"
	messageID := "msg123"
	content := "Hello, world!"
	userID := 1

	tests := []struct {
		name            string
		requestBody     string
		inChat          bool
		sentAt          time.Time
		isUserInChatErr error
		addMessageErr   error
		broadcastUsers  []int
		expectedStatus  int
	}{
		{
			name:            "Success",
			requestBody:     `{"message_id":"msg123","content":"Hello, world!"}`,
			inChat:          true,
			sentAt:          time.Now(),
			isUserInChatErr: nil,
			addMessageErr:   nil,
			broadcastUsers:  []int{1, 2, 3},
			expectedStatus:  http.StatusOK,
		},
		{
			name:            "User Not In Chat",
			requestBody:     `{"message_id":"msg123","content":"Hello, world!"}`,
			inChat:          false,
			sentAt:          time.Time{},
			isUserInChatErr: nil,
			addMessageErr:   nil,
			broadcastUsers:  []int{},
			expectedStatus:  http.StatusNotFound,
		},
		{
			name:            "Duplicate Message ID",
			requestBody:     `{"message_id":"msg123","content":"Hello, world!"}`,
			inChat:          true,
			sentAt:          time.Time{},
			isUserInChatErr: nil,
			addMessageErr:   error(errPrimaryKey()),
			broadcastUsers:  []int{},
			expectedStatus:  http.StatusConflict,
		},
		{
			name:            "Service Error",
			requestBody:     `{"message_id":"msg123","content":"Hello, world!"}`,
			inChat:          true,
			sentAt:          time.Time{},
			isUserInChatErr: nil,
			addMessageErr:   error(errGeneric()),
			broadcastUsers:  []int{},
			expectedStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("IsUserInChat", userID, chatID).Return(tt.inChat, tt.isUserInChatErr)
			if tt.inChat {
				mockService.On("AddMessage", messageID, chatID, userID, content).Return(tt.sentAt, tt.addMessageErr)

				if tt.addMessageErr == nil {
					mockService.On("GetChatParticipantsForBroadcast", chatID).Return(tt.broadcastUsers, nil)
				}
			}

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Post("/chats/{chatID}/messages", handler.SendMessage)

			// Create request
			req, err := http.NewRequest("POST", "/chats/"+chatID+"/messages", strings.NewReader(tt.requestBody))
			require.NoError(t, err)

			// Set content type
			req.Header.Set("Content-Type", "application/json")

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If success, verify response body contains message details
			if tt.expectedStatus == http.StatusOK {
				var response ChatMessage
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, messageID, response.MessageID)
				assert.Equal(t, chatID, response.ChatID)
				assert.Equal(t, userID, response.SenderID)
				assert.Equal(t, content, response.Content)
			}

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test AddParticipant handler
func TestAddParticipant(t *testing.T) {
	chatID := "123"
	userID := 1
	newUserID := 4

	tests := []struct {
		name            string
		requestBody     string
		inChat          bool
		addErr          error
		isUserInChatErr error
		expectedStatus  int
	}{
		{
			name:            "Success",
			requestBody:     `{"user_id":4}`,
			inChat:          true,
			addErr:          nil,
			isUserInChatErr: nil,
			expectedStatus:  http.StatusCreated,
		},
		{
			name:            "User Not In Chat",
			requestBody:     `{"user_id":4}`,
			inChat:          false,
			addErr:          nil,
			isUserInChatErr: nil,
			expectedStatus:  http.StatusNotFound,
		},
		{
			name:            "Service Error",
			requestBody:     `{"user_id":4}`,
			inChat:          true,
			addErr:          error(errGeneric()),
			isUserInChatErr: nil,
			expectedStatus:  http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("IsUserInChat", userID, chatID).Return(tt.inChat, tt.isUserInChatErr)
			if tt.inChat {
				mockService.On("AddParticipant", chatID, newUserID).Return(tt.addErr)
			}

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Post("/chats/{chatID}/participants", handler.AddParticipant)

			// Create request
			req, err := http.NewRequest("POST", "/chats/"+chatID+"/participants", strings.NewReader(tt.requestBody))
			require.NoError(t, err)

			// Set content type
			req.Header.Set("Content-Type", "application/json")

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test AddReaction handler
func TestAddReaction(t *testing.T) {
	messageID := "msg123"
	userID := 1
	reactionID := "react123"
	reactionCode := "👍"
	chatID := "chat123"

	tests := []struct {
		name            string
		requestBody     string
		addErr          error
		getChatIDErr    error
		expectedStatus  int
		shouldBroadcast bool
	}{
		{
			name:            "Success",
			requestBody:     `{"reaction_id":"react123","reaction_code":"👍"}`,
			addErr:          nil,
			getChatIDErr:    nil,
			expectedStatus:  http.StatusOK,
			shouldBroadcast: true,
		},
		{
			name:            "Duplicate Reaction",
			requestBody:     `{"reaction_id":"react123","reaction_code":"👍"}`,
			addErr:          error(errPrimaryKey()),
			getChatIDErr:    nil,
			expectedStatus:  http.StatusConflict,
			shouldBroadcast: false,
		},
		{
			name:            "Invalid Reaction Code",
			requestBody:     `{"reaction_id":"react123","reaction_code":"👍"}`,
			addErr:          error(errInvalidReaction()),
			getChatIDErr:    nil,
			expectedStatus:  http.StatusBadRequest,
			shouldBroadcast: false,
		},
		{
			name:            "User Not Authorized",
			requestBody:     `{"reaction_id":"react123","reaction_code":"👍"}`,
			addErr:          error(errUserNotAuthorized()),
			getChatIDErr:    nil,
			expectedStatus:  http.StatusNotFound,
			shouldBroadcast: false,
		},
		{
			name:            "Service Error",
			requestBody:     `{"reaction_id":"react123","reaction_code":"👍"}`,
			addErr:          error(errGeneric()),
			getChatIDErr:    nil,
			expectedStatus:  http.StatusInternalServerError,
			shouldBroadcast: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("AddReaction", reactionID, messageID, userID, reactionCode).Return(tt.addErr)
			if tt.shouldBroadcast {
				mockService.On("GetChatIDForMessage", messageID).Return(chatID, tt.getChatIDErr)
				mockService.On("GetChatParticipantsForBroadcast", chatID).Return([]int{1, 2, 3}, nil)
			}

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Post("/messages/{messageID}/reactions", handler.AddReaction)

			// Create request
			req, err := http.NewRequest("POST", "/messages/"+messageID+"/reactions", strings.NewReader(tt.requestBody))
			require.NoError(t, err)

			// Set content type
			req.Header.Set("Content-Type", "application/json")

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// If success, verify reaction ID in response
			if tt.expectedStatus == http.StatusOK {
				var response map[string]string
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, reactionID, response["reaction_id"])
			}

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Test RemoveReaction handler
func TestRemoveReaction(t *testing.T) {
	messageID := "msg123"
	userID := 1
	reactionCode := "👍"
	chatID := "chat123"

	tests := []struct {
		name            string
		removeErr       error
		getChatIDErr    error
		expectedStatus  int
		shouldBroadcast bool
	}{
		{
			name:            "Success",
			removeErr:       nil,
			getChatIDErr:    nil,
			expectedStatus:  http.StatusOK,
			shouldBroadcast: true,
		},
		{
			name:            "Service Error",
			removeErr:       error(errGeneric()),
			getChatIDErr:    nil,
			expectedStatus:  http.StatusInternalServerError,
			shouldBroadcast: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service
			mockService := new(MockMessagingService)

			// Set up expectations
			mockService.On("GetChatIDForMessage", messageID).Return(chatID, tt.getChatIDErr)
			mockService.On("RemoveReaction", messageID, userID, reactionCode).Return(tt.removeErr)
			if tt.shouldBroadcast {
				mockService.On("GetChatParticipantsForBroadcast", chatID).Return([]int{1, 2, 3}, nil)
			}

			// Create handler
			handler := NewHandler(mockService)

			// Create request with chi router to handle URL params
			r := chi.NewRouter()
			r.Delete("/messages/{messageID}/reactions/{reactionCode}", handler.RemoveReaction)

			// Create request
			req, err := http.NewRequest("DELETE", "/messages/"+messageID+"/reactions/"+reactionCode, nil)
			require.NoError(t, err)

			// Set context with user ID
			ctx := context.WithValue(req.Context(), "user_id", userID)
			req = req.WithContext(ctx)

			// Record response
			rr := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Verify expectations were met
			mockService.AssertExpectations(t)
		})
	}
}

// Helper function to create mock errors
func errGeneric() error {
	return errors.New("generic error")
}

func errPrimaryKey() error {
	return errors.New("pq: duplicate key value violates unique constraint")
}

func errUserNotInChat() error {
	return errors.New("user not in chat")
}

func errInvalidReaction() error {
	return errors.New("invalid reaction code")
}

func errUserNotAuthorized() error {
	return errors.New("user not authorized to react to this message")
}

// MockConn implements the WSConn interface for testing
type MockConn struct {
	mock.Mock
}

func (m *MockConn) ReadMessage() (int, []byte, error) {
	args := m.Called()
	return args.Int(0), args.Get(1).([]byte), args.Error(2)
}

func (m *MockConn) WriteMessage(messageType int, data []byte) error {
	args := m.Called(messageType, data)
	return args.Error(0)
}

func (m *MockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestHandleWSConnection(t *testing.T) {
	// Create mock objects
	mockConn := new(MockConn)
	mockService := new(MockMessagingService)

	// Setup handler with mock service
	handler := &Handler{
		service:      mockService,
		clients:      make(map[int]*Client),
		clientsMutex: sync.RWMutex{},
	}

	userID := 123
	chatRooms := map[string]struct{}{
		"chat1": {},
		"chat2": {},
	}

	// Setup expectations
	mockService.On("GetUserChatRooms", userID).Return(chatRooms, nil)

	// ReadMessage will be called in the goroutine, so set up a mock that returns an error
	// to make the goroutine exit quickly
	mockConn.On("ReadMessage").Return(0, []byte{}, errors.New("test exit"))
	mockConn.On("Close").Return(nil)

	// Call the method under test
	handler.handleWSConnection(mockConn, userID)

	// Check if client was added to the map
	client, exists := handler.clients[userID]

	assert.True(t, exists, "Client should be added to the clients map")

	if exists {
		// Check if the client was set up correctly
		assert.Equal(t, userID, client.userID, "Client should have the correct userID")
		assert.Equal(t, chatRooms, client.chatRooms, "Client should have the correct chat rooms")
		assert.Equal(t, mockConn, client.conn, "Client should have the correct connection")
	}

	// Verify that our mock expectations were met
	mockService.AssertExpectations(t)
}

func TestHandleWSConnection_ServiceError(t *testing.T) {
	// Create mock objects
	mockConn := new(MockConn)
	mockService := new(MockMessagingService)

	// Setup handler with mock service
	handler := &Handler{
		service:      mockService,
		clients:      make(map[int]*Client),
		clientsMutex: sync.RWMutex{},
	}

	userID := 123

	// Setup expectations - this time return an error from GetUserChatRooms
	mockService.On("GetUserChatRooms", userID).Return(map[string]struct{}{}, errors.New("service error"))

	// ReadMessage will be called in the goroutine, so set up a mock that returns an error
	mockConn.On("ReadMessage").Return(0, []byte{}, errors.New("test exit"))
	mockConn.On("Close").Return(nil)

	// Call the method under test
	handler.handleWSConnection(mockConn, userID)

	// Check if client was added to the map
	client, exists := handler.clients[userID]

	assert.True(t, exists, "Client should be added to the clients map even if service returns an error")

	if exists {
		// Check if the client was set up correctly
		assert.Equal(t, userID, client.userID, "Client should have the correct userID")
		assert.Empty(t, client.chatRooms, "Client should have empty chat rooms on service error")
		assert.Equal(t, mockConn, client.conn, "Client should have the correct connection")
	}

	// Verify that our mock expectations were met
	mockService.AssertExpectations(t)
}

func TestHandleChatMessage(t *testing.T) {
	tests := []struct {
		name             string
		message          WSMessage
		userInChat       bool
		expectStoreMsg   bool
		expectedSentAt   time.Time
		addMessageError  error
		expectBroadcast  bool
		broadcastUserIDs []int
		broadcastError   error
	}{
		{
			name: "Successfully handle message",
			message: WSMessage{
				Type:   MsgTypeChat,
				ChatID: "chat123",
				Payload: mustMarshalJSON(ChatMessage{
					MessageID: "msg123",
					Content:   "Hello, world!",
				}),
			},
			userInChat:       true,
			expectStoreMsg:   true,
			expectedSentAt:   time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC),
			addMessageError:  nil,
			expectBroadcast:  true,
			broadcastUserIDs: []int{1, 2, 3},
			broadcastError:   nil,
		},
		{
			name: "User not in chat",
			message: WSMessage{
				Type:   MsgTypeChat,
				ChatID: "chat123",
				Payload: mustMarshalJSON(ChatMessage{
					MessageID: "msg123",
					Content:   "Hello, world!",
				}),
			},
			userInChat:      false,
			expectStoreMsg:  false,
			expectBroadcast: false,
		},
		{
			name: "Duplicate message",
			message: WSMessage{
				Type:   MsgTypeChat,
				ChatID: "chat123",
				Payload: mustMarshalJSON(ChatMessage{
					MessageID: "msg123",
					Content:   "Hello, world!",
				}),
			},
			userInChat:      true,
			expectStoreMsg:  true,
			addMessageError: error(errPrimaryKey()),
			expectBroadcast: false,
		},
		{
			name: "Service error",
			message: WSMessage{
				Type:   MsgTypeChat,
				ChatID: "chat123",
				Payload: mustMarshalJSON(ChatMessage{
					MessageID: "msg123",
					Content:   "Hello, world!",
				}),
			},
			userInChat:      true,
			expectStoreMsg:  true,
			addMessageError: error(errGeneric()),
			expectBroadcast: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock service and mock connection
			mockService := new(MockMessagingService)
			mockConn := new(MockConn)

			// Setup handler
			handler := &Handler{
				service:      mockService,
				clients:      make(map[int]*Client),
				clientsMutex: sync.RWMutex{},
			}

			// Create client
			userID := 1
			client := &Client{
				conn:      mockConn,
				userID:    userID,
				chatRooms: make(map[string]struct{}),
			}

			// Setup client's chat rooms
			if tt.userInChat {
				client.chatRooms[tt.message.ChatID] = struct{}{}
			}

			// Parse payload to get message details for expectations
			var chatMsg ChatMessage
			json.Unmarshal(tt.message.Payload, &chatMsg)

			// Setup expectations
			if tt.expectStoreMsg {
				mockService.On("AddMessage",
					chatMsg.MessageID,
					tt.message.ChatID,
					userID,
					chatMsg.Content).Return(tt.expectedSentAt, tt.addMessageError)
			}

			if tt.expectBroadcast && tt.addMessageError == nil {
				mockService.On("GetChatParticipantsForBroadcast", tt.message.ChatID).
					Return(tt.broadcastUserIDs, tt.broadcastError)

				// Setup expectations for all clients in broadcast
				for _, broadcastUserID := range tt.broadcastUserIDs {
					// Create and add client to the handler for each user
					if broadcastUserID != userID {
						mockClientConn := new(MockConn)
						mockClientConn.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)

						handler.clients[broadcastUserID] = &Client{
							conn:      mockClientConn,
							userID:    broadcastUserID,
							chatRooms: map[string]struct{}{tt.message.ChatID: {}},
						}
					} else {
						// The original client is already in the map
						mockConn.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
					}
				}
			}

			// Add client to handler's clients map
			handler.clients[userID] = client

			// Call the method under test
			handler.handleChatMessage(client, tt.message)

			// Verify expectations were met
			mockService.AssertExpectations(t)

			// Also verify expectations on the mock connection if needed
			if tt.expectBroadcast && tt.addMessageError == nil {
				mockConn.AssertExpectations(t)

				// Verify other clients' connections too
				for _, broadcastUserID := range tt.broadcastUserIDs {
					if broadcastUserID != userID {
						mockClientConn := handler.clients[broadcastUserID].conn.(*MockConn)
						mockClientConn.AssertExpectations(t)
					}
				}
			}
		})
	}
}

func TestHandleWSConnectionWithMessageSending(t *testing.T) {
	// Create mock objects
	mockConn := new(MockConn)
	mockService := new(MockMessagingService)

	// Setup handler
	handler := &Handler{
		service:      mockService,
		clients:      make(map[int]*Client),
		clientsMutex: sync.RWMutex{},
	}

	userID := 123
	chatID := "chat123"
	messageID := "msg123"
	content := "Hello, world!"
	sentAt := time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC)

	// ChatRooms that the user belongs to
	chatRooms := map[string]struct{}{
		chatID: {},
	}

	// Setup initial expectations
	mockService.On("GetUserChatRooms", userID).Return(chatRooms, nil)

	// Setup message reading expectations
	// First read should return a chat message
	chatMessage := WSMessage{
		Type:   MsgTypeChat,
		ChatID: chatID,
		Payload: mustMarshalJSON(ChatMessage{
			MessageID: messageID,
			Content:   content,
		}),
	}
	chatMessageBytes, _ := json.Marshal(chatMessage)

	// After successful message, return error to end the loop
	mockConn.On("ReadMessage").Return(1, chatMessageBytes, nil).Once()
	mockConn.On("ReadMessage").Return(0, []byte{}, errGeneric()).Once()
	mockConn.On("Close").Return(nil)

	// Service expectations for the message handler
	mockService.On("AddMessage", messageID, chatID, userID, content).Return(sentAt, nil)
	mockService.On("GetChatParticipantsForBroadcast", chatID).Return([]int{userID, 456}, nil)

	// Write message expectation - this is the broadcast
	mockConn.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)

	// Call the method under test
	handler.handleWSConnection(mockConn, userID)

	// Give goroutine time to process messages
	time.Sleep(100 * time.Millisecond)

	// Verify expectations were met
	mockService.AssertExpectations(t)
	mockConn.AssertExpectations(t)
}

func TestHandleChatMessageBroadcasting(t *testing.T) {
	// Create mock service
	mockService := new(MockMessagingService)

	// Setup handler
	handler := &Handler{
		service:      mockService,
		clients:      make(map[int]*Client),
		clientsMutex: sync.RWMutex{},
	}

	chatID := "chat123"
	messageID := "msg123"
	userID1 := 1
	userID2 := 2
	userID3 := 3 // Not connected
	content := "Hello, world!"
	sentAt := time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC)

	// Create mock connections
	mockConn1 := new(MockConn)
	mockConn2 := new(MockConn)

	// Create clients
	client1 := &Client{
		conn:      mockConn1,
		userID:    userID1,
		chatRooms: map[string]struct{}{chatID: {}},
	}

	client2 := &Client{
		conn:      mockConn2,
		userID:    userID2,
		chatRooms: map[string]struct{}{chatID: {}},
	}

	// Add clients to handler
	handler.clients[userID1] = client1
	handler.clients[userID2] = client2

	// Create message
	message := WSMessage{
		Type:   MsgTypeChat,
		ChatID: chatID,
		Payload: mustMarshalJSON(ChatMessage{
			MessageID: messageID,
			Content:   content,
		}),
	}

	// Setup expectations
	mockService.On("AddMessage", messageID, chatID, userID1, content).Return(sentAt, nil)
	mockService.On("GetChatParticipantsForBroadcast", chatID).Return([]int{userID1, userID2, userID3}, nil)

	// Both connected clients should receive the message
	mockConn1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	mockConn2.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)

	// Call the method under test
	handler.handleChatMessage(client1, message)

	// Verify that both connected clients received the message
	mockConn1.AssertNumberOfCalls(t, "WriteMessage", 1)
	mockConn2.AssertNumberOfCalls(t, "WriteMessage", 1)

	// Verify expectations were met
	mockService.AssertExpectations(t)
	mockConn1.AssertExpectations(t)
	mockConn2.AssertExpectations(t)
}

func TestWSMessageWriteErrorHandling(t *testing.T) {
	// Create mock service
	mockService := new(MockMessagingService)

	// Setup handler
	handler := &Handler{
		service:      mockService,
		clients:      make(map[int]*Client),
		clientsMutex: sync.RWMutex{},
	}

	chatID := "chat123"
	messageID := "msg123"
	userID1 := 1
	userID2 := 2
	content := "Hello, world!"
	sentAt := time.Date(2025, 4, 15, 10, 0, 0, 0, time.UTC)

	// Create mock connections
	mockConn1 := new(MockConn)
	mockConn2 := new(MockConn)

	// Create clients
	client1 := &Client{
		conn:      mockConn1,
		userID:    userID1,
		chatRooms: map[string]struct{}{chatID: {}},
	}

	client2 := &Client{
		conn:      mockConn2,
		userID:    userID2,
		chatRooms: map[string]struct{}{chatID: {}},
	}

	// Add clients to handler
	handler.clients[userID1] = client1
	handler.clients[userID2] = client2

	// Create message
	message := WSMessage{
		Type:   MsgTypeChat,
		ChatID: chatID,
		Payload: mustMarshalJSON(ChatMessage{
			MessageID: messageID,
			Content:   content,
		}),
	}

	// Setup expectations
	mockService.On("AddMessage", messageID, chatID, userID1, content).Return(sentAt, nil)
	mockService.On("GetChatParticipantsForBroadcast", chatID).Return([]int{userID1, userID2}, nil)

	// First client succeeds, second client fails
	mockConn1.On("WriteMessage", mock.Anything, mock.Anything).Return(nil)
	mockConn2.On("WriteMessage", mock.Anything, mock.Anything).Return(errGeneric())

	// Call the method under test
	handler.handleChatMessage(client1, message)

	// Verify expectations were met - should continue despite write error for client2
	mockService.AssertExpectations(t)
	mockConn1.AssertExpectations(t)
	mockConn2.AssertExpectations(t)
}

// Helper function to marshal JSON for tests
func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
