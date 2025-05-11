package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/messaging"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// testUser holds user details and associated profiles
type testUser struct {
	UserID      int
	Token       string
	ProfileID   int
	ProfileType string
}

// MessagingIntegrationTestSuite defines a set of integration tests for messaging
type MessagingIntegrationTestSuite struct {
	suite.Suite
	appUrl      string
	wsConnMutex sync.Mutex
	wsConns     []*websocket.Conn
}

// Sample message for testing
type messagingRequest struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

// Chat creation request
type createChatRequest struct {
	ChatID       string `json:"chat_id"`
	ChatName     string `json:"chat_name"`
	Participants []int  `json:"participants"`
}

// SetupSuite prepares the test environment before running all tests
func (s *MessagingIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Default for local testing
	}
}

// TearDownSuite cleans up after all tests have run
func (s *MessagingIntegrationTestSuite) TearDownSuite() {
	// Close WebSocket connections if any are open
	s.wsConnMutex.Lock()
	defer s.wsConnMutex.Unlock()

	for _, conn := range s.wsConns {
		if conn != nil {
			conn.Close()
		}
	}
}

// Helper function to create a test user
func (s *MessagingIntegrationTestSuite) createTestUser() (int, string, error) {
	// Generate unique email for the user
	testEmail := fmt.Sprintf("messaging_test_user_%d_%d@example.com", os.Getpid(), time.Now().UnixNano())
	testPassword := "TestPassword123"

	// Register test user
	registerData := auth.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
	}

	registerJSON, _ := json.Marshal(registerData)
	registerReq, _ := http.NewRequest("POST", s.appUrl+"/api/auth/register", bytes.NewBuffer(registerJSON))
	registerReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	registerResp, err := client.Do(registerReq)
	if err != nil {
		return 0, "", fmt.Errorf("failed to register test user: %v", err)
	}
	defer registerResp.Body.Close()

	if registerResp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(registerResp.Body)
		return 0, "", fmt.Errorf("failed to register test user. Status: %d, Body: %s", registerResp.StatusCode, string(body))
	}

	var registerResult auth.AuthResponse
	err = json.NewDecoder(registerResp.Body).Decode(&registerResult)
	if err != nil {
		return 0, "", fmt.Errorf("failed to decode register response: %v", err)
	}

	return registerResult.UserID, registerResult.Token, nil
}

// Helper function to create a chat
func (s *MessagingIntegrationTestSuite) createChat(token string, chatID string, chatName string, participants []int) error {
	createChatReq := createChatRequest{
		ChatID:       chatID,
		ChatName:     chatName,
		Participants: participants,
	}

	createChatJSON, _ := json.Marshal(createChatReq)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/chats", bytes.NewBuffer(createChatJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create chat: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create chat. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Helper to create users and a chat for testing
func (s *MessagingIntegrationTestSuite) setupUsersAndChat() ([]testUser, string, error) {
	// Create test users
	testUsers := make([]testUser, 2)
	for i := 0; i < 2; i++ {
		userID, token, err := s.createTestUser()
		if err != nil {
			return nil, "", fmt.Errorf("failed to create test user: %v", err)
		}
		testUsers[i] = testUser{UserID: userID, Token: token}
	}

	// Create a chat between the test users
	chatID := uuid.NewString()
	err := s.createChat(testUsers[0].Token, chatID, "Test Chat", []int{testUsers[0].UserID, testUsers[1].UserID})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create test chat: %v", err)
	}

	return testUsers, chatID, nil
}

func generateMessageID() string {
	return uuid.NewString()
}

func generateReactionId() string {
	return uuid.NewString()
}

// TestCreateChat tests the creation of a new chat
func (s *MessagingIntegrationTestSuite) TestCreateChat() {
	t := s.T()

	// Create test users
	user1ID, user1Token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create first test user")
	user2ID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create second test user")

	// Create a unique chat ID
	chatID := uuid.NewString()
	chatName := "Test Chat Creation"

	// Create chat request
	createChatReq := createChatRequest{
		ChatID:       chatID,
		ChatName:     chatName,
		Participants: []int{user1ID, user2ID},
	}

	createChatJSON, _ := json.Marshal(createChatReq)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/chats", bytes.NewBuffer(createChatJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return status 201 Created")

	// Check response content
	var response map[string]string
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, chatID, response["chat_id"], "Returned chat ID should match")
}

// TestCreateChatUnauthorized tests creating a chat without authentication
func (s *MessagingIntegrationTestSuite) TestCreateChatUnauthorized() {
	t := s.T()

	// Create test users (only used for the request, not for auth)
	user1ID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create first test user")
	user2ID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create second test user")

	chatID := uuid.NewString()
	createChatReq := createChatRequest{
		ChatID:       chatID,
		ChatName:     "Test Unauthorized Chat",
		Participants: []int{user1ID, user2ID},
	}

	createChatJSON, _ := json.Marshal(createChatReq)
	req, _ := http.NewRequest("POST", s.appUrl+"/api/chats", bytes.NewBuffer(createChatJSON))
	req.Header.Set("Content-Type", "application/json")
	// Deliberately not setting Authorization header

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Should return Unauthorized
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "Should return status 401 Unauthorized")
}

// TestSendAndGetMessages tests sending messages and retrieving them
func (s *MessagingIntegrationTestSuite) TestSendAndGetMessages() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	// Send a message to the test chat
	messageID := generateMessageID()
	messageContent := "Hello, this is a test message!"

	sendMsgReq := messagingRequest{
		MessageID: messageID,
		Content:   messageContent,
	}

	sendMsgJSON, _ := json.Marshal(sendMsgReq)
	sendReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/chats/%s/messages", s.appUrl, chatID), bytes.NewBuffer(sendMsgJSON))
	sendReq.Header.Set("Content-Type", "application/json")
	sendReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	sendResp, err := client.Do(sendReq)
	assert.NoError(t, err)
	defer sendResp.Body.Close()

	// Check send response
	assert.Equal(t, http.StatusOK, sendResp.StatusCode, "Should return status 200 OK")

	// Get messages from the chat
	getReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s/messages", s.appUrl, chatID), nil)
	getReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	getResp, err := client.Do(getReq)
	assert.NoError(t, err)
	defer getResp.Body.Close()

	// Check get response
	assert.Equal(t, http.StatusOK, getResp.StatusCode, "Should return status 200 OK")

	// Parse response body
	var messages []map[string]interface{}
	err = json.NewDecoder(getResp.Body).Decode(&messages)
	assert.NoError(t, err)

	// Verify that the sent message is in the response
	messageFound := false
	for _, msg := range messages {
		if msg["message_id"] == messageID {
			assert.Equal(t, messageContent, msg["content"], "Message content should match")
			assert.Equal(t, float64(testUsers[0].UserID), msg["sender_id"], "Sender ID should match")
			messageFound = true
			break
		}
	}
	assert.True(t, messageFound, "The sent message should be retrieved")
}

// TestGetUserChats tests retrieving a user's chats
func (s *MessagingIntegrationTestSuite) TestGetUserChats() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	req, _ := http.NewRequest("GET", s.appUrl+"/api/chats", nil)
	req.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response content
	var chats []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&chats)
	assert.NoError(t, err)

	// Verify that the test chat is in the response
	chatFound := false
	for _, chat := range chats {
		if chat["chat_id"] == chatID {
			chatFound = true
			break
		}
	}
	assert.True(t, chatFound, "The test chat should be retrieved")
}

// TestGetChatDetails tests retrieving chat details
func (s *MessagingIntegrationTestSuite) TestGetChatDetails() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s", s.appUrl, chatID), nil)
	req.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Check response content
	var chatDetails map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&chatDetails)
	assert.NoError(t, err)

	// Verify chat details
	assert.Equal(t, chatID, chatDetails["chat_id"], "Chat ID should match")
	assert.Equal(t, "Test Chat", chatDetails["chat_name"], "Chat name should match")

	// Check that participants include our test users
	participants, ok := chatDetails["participants"].([]interface{})
	assert.True(t, ok, "Participants should be an array")

	participant1Found := false
	participant2Found := false
	for _, p := range participants {
		pid, ok := p.(float64)
		if !ok {
			continue
		}

		if int(pid) == testUsers[0].UserID {
			participant1Found = true
		} else if int(pid) == testUsers[1].UserID {
			participant2Found = true
		}
	}

	assert.True(t, participant1Found, "First participant should be in the chat")
	assert.True(t, participant2Found, "Second participant should be in the chat")
}

// Helper function to send a message and return its ID
func (s *MessagingIntegrationTestSuite) sendTestMessage(token string, chatID string) (string, error) {
	messageID := generateMessageID()
	messageContent := "Test message for reactions"

	sendMsgReq := messagingRequest{
		MessageID: messageID,
		Content:   messageContent,
	}

	sendMsgJSON, _ := json.Marshal(sendMsgReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/chats/%s/messages", s.appUrl, chatID), bytes.NewBuffer(sendMsgJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to send message. Status: %d, Body: %s", resp.StatusCode, string(body))
	}

	return messageID, nil
}

// TestAddReaction tests adding a reaction to a message
func (s *MessagingIntegrationTestSuite) TestAddReaction() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	// Send a test message to react to
	messageID, err := s.sendTestMessage(testUsers[0].Token, chatID)
	assert.NoError(t, err, "Failed to send test message")

	// Add a reaction
	reactionID := generateReactionId()
	reactionReq := struct {
		ReactionID   string `json:"reaction_id"`
		ReactionCode string `json:"reaction_code"`
	}{
		ReactionID:   reactionID,
		ReactionCode: "clap",
	}

	reactionJSON, _ := json.Marshal(reactionReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/messages/%s/reactions", s.appUrl, messageID), bytes.NewBuffer(reactionJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	// Parse the response body
	var response map[string]string
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, reactionID, response["reaction_id"], "Returned reaction ID should match")
}

// TestRemoveReaction tests removing a reaction from a message
func (s *MessagingIntegrationTestSuite) TestRemoveReaction() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	// Send a test message to react to
	messageID, err := s.sendTestMessage(testUsers[0].Token, chatID)
	assert.NoError(t, err, "Failed to send test message")

	// Add a reaction first
	reactionID := generateReactionId()
	reactionReq := struct {
		ReactionID   string `json:"reaction_id"`
		ReactionCode string `json:"reaction_code"`
	}{
		ReactionID:   reactionID,
		ReactionCode: "clap",
	}

	reactionJSON, _ := json.Marshal(reactionReq)
	addReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/messages/%s/reactions", s.appUrl, messageID), bytes.NewBuffer(reactionJSON))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	addResp, err := client.Do(addReq)
	assert.NoError(t, err)
	addResp.Body.Close()

	// Now remove the reaction
	removeReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/messages/%s/reactions/%s", s.appUrl, messageID, "clap"), nil)
	removeReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	removeResp, err := client.Do(removeReq)
	assert.NoError(t, err)
	defer removeResp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusOK, removeResp.StatusCode, "Should return status 200 OK")
}

// TestAddParticipant tests adding a participant to a chat
func (s *MessagingIntegrationTestSuite) TestAddParticipant() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	// Create a new user to add to the chat
	newUserID, _, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create test user")

	// Add the user to the chat
	addParticipantReq := struct {
		UserID int `json:"user_id"`
	}{
		UserID: newUserID,
	}

	addParticipantJSON, _ := json.Marshal(addParticipantReq)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/chats/%s/participants", s.appUrl, chatID), bytes.NewBuffer(addParticipantJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Check response status
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "Should return status 201 Created")

	// Verify the user was added by checking chat details
	verifyReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s", s.appUrl, chatID), nil)
	verifyReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

	verifyResp, err := client.Do(verifyReq)
	assert.NoError(t, err)
	defer verifyResp.Body.Close()

	var chatDetails map[string]interface{}
	err = json.NewDecoder(verifyResp.Body).Decode(&chatDetails)
	assert.NoError(t, err)

	participants, ok := chatDetails["participants"].([]interface{})
	assert.True(t, ok, "Participants should be an array")

	newUserFound := false
	for _, p := range participants {
		pid, ok := p.(float64)
		if !ok {
			continue
		}

		if int(pid) == newUserID {
			newUserFound = true
			break
		}
	}

	assert.True(t, newUserFound, "New participant should be in the chat")
}

// // TestRemoveParticipant tests removing a participant from a chat
// func (s *MessagingIntegrationTestSuite) TestRemoveParticipant() {
// 	t := s.T()

// 	// Setup test users and chat
// 	testUsers, chatID, err := s.setupUsersAndChat()
// 	assert.NoError(t, err, "Failed to setup users and chat")

// 	// Create a new user to add and then remove from the chat
// 	newUserID, _, err := s.createTestUser()
// 	assert.NoError(t, err, "Failed to create test user")

// 	// Add the user to the chat
// 	addParticipantReq := struct {
// 		UserID int `json:"user_id"`
// 	}{
// 		UserID: newUserID,
// 	}

// 	addParticipantJSON, _ := json.Marshal(addParticipantReq)
// 	addReq, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/chats/%s/participants", s.appUrl, chatID), bytes.NewBuffer(addParticipantJSON))
// 	addReq.Header.Set("Content-Type", "application/json")
// 	addReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

// 	client := &http.Client{}
// 	addResp, err := client.Do(addReq)
// 	assert.NoError(t, err)
// 	addResp.Body.Close()

// 	// Now remove the user from the chat
// 	removeReq, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/chats/%s/participants/%d", s.appUrl, chatID, newUserID), nil)
// 	removeReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

// 	removeResp, err := client.Do(removeReq)
// 	assert.NoError(t, err)
// 	defer removeResp.Body.Close()

// 	// Check response status
// 	assert.Equal(t, http.StatusOK, removeResp.StatusCode, "Should return status 200 OK")

// 	// Verify the user was removed by checking chat details
// 	verifyReq, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s", s.appUrl, chatID), nil)
// 	verifyReq.Header.Set("Authorization", "Bearer "+testUsers[0].Token)

// 	verifyResp, err := client.Do(verifyReq)
// 	assert.NoError(t, err)
// 	defer verifyResp.Body.Close()

// 	var chatDetails map[string]interface{}
// 	err = json.NewDecoder(verifyResp.Body).Decode(&chatDetails)
// 	assert.NoError(t, err)

// 	participants, ok := chatDetails["participants"].([]interface{})
// 	assert.True(t, ok, "Participants should be an array")

// 	userStillInChat := false
// 	for _, p := range participants {
// 		pid, ok := p.(float64)
// 		if !ok {
// 			continue
// 		}

// 		if int(pid) == newUserID {
// 			userStillInChat = true
// 			break
// 		}
// 	}

// 	assert.False(t, userStillInChat, "Removed user should not be in the chat")
// }

// TestWebSocketMessaging tests sending and receiving messages via WebSocket
func (s *MessagingIntegrationTestSuite) TestWebSocketMessaging() {
	t := s.T()

	// Setup test users and chat
	testUsers, chatID, err := s.setupUsersAndChat()
	assert.NoError(t, err, "Failed to setup users and chat")

	// Convert HTTP URL to WebSocket URL
	wsURL := fmt.Sprintf("ws%s/api/ws/chat", s.appUrl[4:])

	// Connect sender
	headerSender := http.Header{}
	headerSender.Add("Authorization", "Bearer "+testUsers[0].Token)
	connSender, _, err := websocket.DefaultDialer.Dial(wsURL, headerSender)
	assert.NoError(t, err, "Sender should connect to WebSocket")

	if err == nil && connSender != nil {
		s.wsConnMutex.Lock()
		s.wsConns = append(s.wsConns, connSender)
		s.wsConnMutex.Unlock()

		// Connect receiver
		headerReceiver := http.Header{}
		headerReceiver.Add("Authorization", "Bearer "+testUsers[1].Token)
		connReceiver, _, err := websocket.DefaultDialer.Dial(wsURL, headerReceiver)
		assert.NoError(t, err, "Receiver should connect to WebSocket")

		if err == nil && connReceiver != nil {
			s.wsConnMutex.Lock()
			s.wsConns = append(s.wsConns, connReceiver)
			s.wsConnMutex.Unlock()

			// Prepare test message
			wsMessageID := generateMessageID()
			messageContent := "Hello via WebSocket!"

			// Create ChatMessage with embedded BaseMessage
			chatMsg := messaging.ChatMessage{
				BaseMessage: messaging.BaseMessage{
					Type:   messaging.MsgTypeChatMessage,
					ChatID: chatID,
				},
				MessageID: wsMessageID,
				Content:   messageContent,
			}

			chatMsgJSON, _ := json.Marshal(chatMsg)

			// Set up receiver to listen for the message
			receiverGotMessage := make(chan bool, 1)
			go func() {
				for {
					_, msg, err := connReceiver.ReadMessage()
					if err != nil {
						t.Logf("Receiver error reading message: %v", err)
						receiverGotMessage <- false
						return
					}

					var response messaging.BaseMessage
					if err := json.Unmarshal(msg, &response); err != nil {
						continue
					}

					if response.Type == messaging.MsgTypeChatMessage {
						var chatResponse messaging.ChatMessage
						if err := json.Unmarshal(msg, &chatResponse); err == nil {
							if chatResponse.MessageID == wsMessageID &&
								chatResponse.Content == messageContent {
								receiverGotMessage <- true
								return
							}
						}
					}
				}
			}()

			// Set up sender to listen for the message echo
			senderGotMessage := make(chan bool, 1)
			go func() {
				for {
					_, msg, err := connSender.ReadMessage()
					if err != nil {
						t.Logf("Sender error reading message: %v", err)
						senderGotMessage <- false
						return
					}

					var response messaging.BaseMessage
					if err := json.Unmarshal(msg, &response); err != nil {
						continue
					}

					if response.Type == messaging.MsgTypeChatMessage {
						var chatResponse messaging.ChatMessage
						if err := json.Unmarshal(msg, &chatResponse); err == nil {
							if chatResponse.MessageID == wsMessageID &&
								chatResponse.Content == messageContent {
								senderGotMessage <- true
								return
							}
						}
					}
				}
			}()

			// Send the message
			err = connSender.WriteMessage(websocket.TextMessage, chatMsgJSON)
			assert.NoError(t, err, "Should send message via WebSocket")

			// Wait for receiver to get the message (with timeout)
			select {
			case received := <-receiverGotMessage:
				assert.True(t, received, "Receiver should receive the message")
			case <-time.After(5 * time.Second):
				t.Error("Timed out waiting for receiver to get the message")
			}

			// Wait for sender to get the message echo (with timeout)
			select {
			case received := <-senderGotMessage:
				assert.True(t, received, "Sender should receive the message echo")
			case <-time.After(5 * time.Second):
				t.Error("Timed out waiting for sender to get the message echo")
			}
		}
	}
}

// TestGetOrCreateDirectChat tests creating or finding a direct chat between two users
func (s *MessagingIntegrationTestSuite) TestGetOrCreateDirectChat() {
	t := s.T()

	// Create two test users
	user1ID, user1Token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create first test user")
	user2ID, user2Token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create second test user")

	// Prepare request body
	reqBody := map[string]int{"user_id": user2ID}
	reqJSON, _ := json.Marshal(reqBody)

	// User1 requests to create/get direct chat with User2
	req, _ := http.NewRequest("POST", s.appUrl+"/api/chats/direct", bytes.NewBuffer(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	var response map[string]string
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	chatID, ok := response["chat_id"]
	assert.True(t, ok, "Response should contain chat_id")
	assert.NotEmpty(t, chatID, "chat_id should not be empty")

	// User2 requests to get/create direct chat with User1 (should return the same chat)
	req2Body := map[string]int{"user_id": user1ID}
	req2JSON, _ := json.Marshal(req2Body)
	req2, _ := http.NewRequest("POST", s.appUrl+"/api/chats/direct", bytes.NewBuffer(req2JSON))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+user2Token)

	resp2, err := client.Do(req2)
	assert.NoError(t, err)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode, "Should return status 200 OK")

	var response2 map[string]string
	err = json.NewDecoder(resp2.Body).Decode(&response2)
	assert.NoError(t, err)
	chatID2, ok := response2["chat_id"]
	assert.True(t, ok, "Response should contain chat_id")
	assert.Equal(t, chatID, chatID2, "Both users should get the same direct chat ID")

	// Negative test: user tries to create direct chat with themselves
	selfReqBody := map[string]int{"user_id": user1ID}
	selfReqJSON, _ := json.Marshal(selfReqBody)
	selfReq, _ := http.NewRequest("POST", s.appUrl+"/api/chats/direct", bytes.NewBuffer(selfReqJSON))
	selfReq.Header.Set("Content-Type", "application/json")
	selfReq.Header.Set("Authorization", "Bearer "+user1Token)

	selfResp, err := client.Do(selfReq)
	assert.NoError(t, err)
	defer selfResp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, selfResp.StatusCode, "Should not allow creating direct chat with self")
}

// TestDirectChatNameDisplay tests that direct chat names display the name of the other participant
func (s *MessagingIntegrationTestSuite) TestDirectChatNameDisplay() {
	t := s.T()

	// Create two test users and set up profiles for them
	user1ID, user1Token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create first test user")

	user2ID, user2Token, err := s.createTestUser()
	assert.NoError(t, err, "Failed to create second test user")

	// Create profiles for both users with different names
	user1Name := "Alice Test"
	user2Name := "Bob Test"

	// Create profile for user1
	user1Profile := map[string]interface{}{
		"user_id":          user1ID,
		"full_name":        user1Name,
		"birthday":         "1990-01-01",
		"gender":           "female",
		"city_id":          1,
		"bio":              "Test bio 1",
		"goal":             "career",
		"improv_styles":    []string{"longform"},
		"looking_for_team": true,
	}

	profileJSON1, _ := json.Marshal(user1Profile)
	req1, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(profileJSON1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+user1Token)

	client := &http.Client{}
	resp1, err := client.Do(req1)
	assert.NoError(t, err)
	if resp1.Body != nil {
		defer resp1.Body.Close()
	}

	// Create profile for user2
	user2Profile := map[string]interface{}{
		"user_id":          user2ID,
		"full_name":        user2Name,
		"birthday":         "1992-02-02",
		"gender":           "male",
		"city_id":          1,
		"bio":              "Test bio 2",
		"goal":             "hobby",
		"improv_styles":    []string{"shortform"},
		"looking_for_team": false,
	}

	profileJSON2, _ := json.Marshal(user2Profile)
	req2, _ := http.NewRequest("POST", s.appUrl+"/api/profiles", bytes.NewBuffer(profileJSON2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+user2Token)

	resp2, err := client.Do(req2)
	assert.NoError(t, err)
	if resp2.Body != nil {
		defer resp2.Body.Close()
	}

	// Create or get direct chat between user1 and user2
	directChatReq := map[string]int{"user_id": user2ID}
	directChatJSON, _ := json.Marshal(directChatReq)

	req, _ := http.NewRequest("POST", s.appUrl+"/api/chats/direct", bytes.NewBuffer(directChatJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user1Token)

	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return status 200 OK")

	var chatResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&chatResponse)
	assert.NoError(t, err)
	chatID := chatResponse["chat_id"]
	assert.NotEmpty(t, chatID, "ChatID should not be empty")

	// Test 1: User1 should see User2's name as the chat name
	getChatReq1, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s", s.appUrl, chatID), nil)
	getChatReq1.Header.Set("Authorization", "Bearer "+user1Token)

	getChatResp1, err := client.Do(getChatReq1)
	assert.NoError(t, err)
	defer getChatResp1.Body.Close()
	assert.Equal(t, http.StatusOK, getChatResp1.StatusCode, "Should return status 200 OK")

	var chat1 map[string]interface{}
	err = json.NewDecoder(getChatResp1.Body).Decode(&chat1)
	assert.NoError(t, err)
	assert.Equal(t, user2Name, chat1["chat_name"], "User1 should see User2's name as the chat name")
	assert.Equal(t, false, chat1["is_group"], "Direct chat should not be a group chat")

	// Test 2: User2 should see User1's name as the chat name
	getChatReq2, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/chats/%s", s.appUrl, chatID), nil)
	getChatReq2.Header.Set("Authorization", "Bearer "+user2Token)

	getChatResp2, err := client.Do(getChatReq2)
	assert.NoError(t, err)
	defer getChatResp2.Body.Close()
	assert.Equal(t, http.StatusOK, getChatResp2.StatusCode, "Should return status 200 OK")

	var chat2 map[string]interface{}
	err = json.NewDecoder(getChatResp2.Body).Decode(&chat2)
	assert.NoError(t, err)
	assert.Equal(t, user1Name, chat2["chat_name"], "User2 should see User1's name as the chat name")
	assert.Equal(t, false, chat2["is_group"], "Direct chat should not be a group chat")

	// Also test the chat list to verify consistent naming in the list view
	getChatsReq1, _ := http.NewRequest("GET", s.appUrl+"/api/chats", nil)
	getChatsReq1.Header.Set("Authorization", "Bearer "+user1Token)

	getChatsResp1, err := client.Do(getChatsReq1)
	assert.NoError(t, err)
	defer getChatsResp1.Body.Close()
	assert.Equal(t, http.StatusOK, getChatsResp1.StatusCode, "Should return status 200 OK")

	var chatsList1 []map[string]interface{}
	err = json.NewDecoder(getChatsResp1.Body).Decode(&chatsList1)
	assert.NoError(t, err)

	// Find our test chat in the list
	foundChatInList := false
	for _, c := range chatsList1 {
		if c["chat_id"] == chatID {
			foundChatInList = true
			assert.Equal(t, user2Name, c["chat_name"], "User1's chat list should show User2's name")
			assert.Equal(t, false, c["is_group"], "Direct chat in list should not be a group chat")
			break
		}
	}
	assert.True(t, foundChatInList, "Should find the direct chat in user1's chat list")
}

// TestMessagingIntegration runs the messaging integration test suite
func TestMessagingIntegration(t *testing.T) {
	// Skip tests if SKIP_INTEGRATION_TESTS environment variable is set
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(MessagingIntegrationTestSuite))
}
