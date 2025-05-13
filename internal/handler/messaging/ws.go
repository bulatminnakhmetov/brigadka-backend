package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/push"
	"github.com/gorilla/websocket"
)

// BaseMessage defines the common fields for all WebSocket messages
type BaseMessage struct {
	Type   string `json:"type"`
	ChatID string `json:"chat_id,omitempty"`
}

// ChatMessage represents a message sent in a chat
type ChatMessage struct {
	BaseMessage
	MessageID string    `json:"message_id"`
	SenderID  int       `json:"sender_id"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sent_at,omitempty"`
}

// JoinMessage represents a user joining a chat
type JoinMessage struct {
	BaseMessage
	UserID   int       `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

// LeaveMessage represents a user leaving a chat
type LeaveMessage struct {
	BaseMessage
	UserID int       `json:"user_id"`
	LeftAt time.Time `json:"left_at"`
}

// ReactionMessage represents a reaction to a message
type ReactionMessage struct {
	BaseMessage
	ReactionID   string    `json:"reaction_id"`
	MessageID    string    `json:"message_id"`
	UserID       int       `json:"user_id"`
	ReactionCode string    `json:"reaction_code"`
	ReactedAt    time.Time `json:"reacted_at,omitempty"`
}

// ReactionMessage represents a reaction to a message
type ReactionRemovedMessage struct {
	BaseMessage
	ReactionID   string    `json:"reaction_id"`
	MessageID    string    `json:"message_id"`
	UserID       int       `json:"user_id"`
	ReactionCode string    `json:"reaction_code"`
	RemovedAt    time.Time `json:"reacted_at,omitempty"`
}

// TypingMessage represents a typing indicator
type TypingMessage struct {
	BaseMessage
	UserID    int       `json:"user_id"`
	IsTyping  bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}

// ReadReceiptMessage represents a read receipt notification
type ReadReceiptMessage struct {
	BaseMessage
	UserID    int       `json:"user_id"`
	MessageID string    `json:"message_id"`
	ReadAt    time.Time `json:"read_at"`
}

// Message type constants
const (
	MsgTypeChatMessage    = "chat_message"
	MsgTypeReaction       = "reaction"
	MsgTypeRemoveReaction = "remove_reaction"
	MsgTypeTyping         = "typing"
	MsgTypeReadReceipt    = "read_receipt"
)

func (h *Handler) handleWSConnection(conn WSConn, userID int) {
	// Create new client
	client := &Client{
		conn:   conn,
		userID: userID,
	}

	// Add client to clients map
	h.clientsMutex.Lock()
	h.clients[userID] = client
	h.clientsMutex.Unlock()

	// Handle WebSocket connection
	go h.handleClient(client)
}

// handleClient handles messages from a specific client
func (h *Handler) handleClient(client *Client) {
	defer func() {
		client.conn.Close()
		h.clientsMutex.Lock()
		delete(h.clients, client.userID)
		h.clientsMutex.Unlock()
	}()

	for {
		// Read message from client
		_, data, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message to get the type
		var baseMsg BaseMessage
		if err := json.Unmarshal(data, &baseMsg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		isUserInChat, err := h.messagineService.IsUserInChat(client.userID, baseMsg.ChatID)
		if err != nil {
			log.Printf("Error checking if user is in chat: %v", err)
			continue
		}

		if !isUserInChat {
			log.Printf("User %d not in chat %s", client.userID, baseMsg.ChatID)
			continue
		}

		// Handle message based on type
		switch baseMsg.Type {
		case MsgTypeChatMessage:
			var chatMsg ChatMessage
			if err := json.Unmarshal(data, &chatMsg); err != nil {
				log.Printf("Error parsing chat message: %v", err)
				continue
			}
			h.handleChatMessage(client, chatMsg)
		case MsgTypeReaction:
			var reactionMsg ReactionMessage
			if err := json.Unmarshal(data, &reactionMsg); err != nil {
				log.Printf("Error parsing reaction message: %v", err)
				continue
			}
			h.handleReaction(client, reactionMsg)
		case MsgTypeTyping:
			var typingMsg TypingMessage
			if err := json.Unmarshal(data, &typingMsg); err != nil {
				log.Printf("Error parsing typing message: %v", err)
				continue
			}
			h.handleTypingIndicator(client, typingMsg)
		case MsgTypeReadReceipt:
			var readReceiptMsg ReadReceiptMessage
			if err := json.Unmarshal(data, &readReceiptMsg); err != nil {
				log.Printf("Error parsing read receipt message: %v", err)
				continue
			}
			h.handleReadReceipt(client, readReceiptMsg)
		default:
			log.Printf("Unknown message type: %s", baseMsg.Type)
		}
	}
}

// handleChatMessage handles a chat message from a client
func (h *Handler) handleChatMessage(client *Client, msg ChatMessage) {
	// Store message using the service
	sentAt, err := h.messagineService.AddMessage(msg.MessageID, msg.ChatID, client.userID, msg.Content)
	if err != nil {
		// Check if it's a duplicate message (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			log.Printf("Duplicate message detected (ID: %s), ignoring", msg.MessageID)
			return
		}
		log.Printf("Error storing message: %v", err)
		return
	}

	// Update the sent time and sender ID in the message
	msg.SentAt = sentAt
	msg.SenderID = client.userID

	// Marshal message to JSON
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling chat message: %v", err)
		return
	}

	// Get all participants in the chat
	participants, err := h.messagineService.GetChatParticipantsForBroadcast(msg.ChatID)
	if err != nil {
		log.Printf("Error fetching chat participants: %v", err)
		return
	}

	// Track which participants are offline to send push notifications
	offlineParticipants := make([]int, 0)

	// Send message to all online participants
	h.clientsMutex.RLock()
	for _, userID := range participants {
		// Skip the sender
		if userID == client.userID {
			continue
		}

		if client, ok := h.clients[userID]; ok {
			// Participant is online, send via WebSocket
			if err := client.conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
				log.Printf("Error sending message to user %d: %v", userID, err)
			}
		} else {
			// Participant is offline, add to list for push notification
			offlineParticipants = append(offlineParticipants, userID)
		}
	}
	h.clientsMutex.RUnlock()

	// Send push notifications to offline participants
	if len(offlineParticipants) > 0 {
		h.sendChatPushNotifications(client.userID, msg, offlineParticipants)
	}
}

// sendChatPushNotifications sends push notifications to offline participants
func (h *Handler) sendChatPushNotifications(senderID int, msg ChatMessage, recipients []int) {
	// Get sender profile to include name in notification
	senderProfile, err := h.profileService.GetProfile(senderID)
	if err != nil {
		log.Printf("Error fetching sender profile for push notification: %v", err)
		return
	}

	// Get chat details to include chat name
	chatDetails, err := h.messagineService.GetChat(msg.ChatID, senderID)
	if err != nil {
		log.Printf("Error fetching chat details for push notification: %v", err)
		return
	}

	// Create notification title based on chat type
	title := senderProfile.FullName
	if chatDetails.IsGroup && chatDetails.ChatName != nil {
		title = fmt.Sprintf("%s in %s", senderProfile.FullName, *chatDetails.ChatName)
	}

	// Create notification payload
	payload := push.NotificationPayload{
		Title: title,
		Body:  msg.Content,
		Sound: "default",
		Badge: 1,
	}

	// If sender has avatar, include it
	if senderProfile.Avatar != nil {
		payload.ImageURL = senderProfile.Avatar.URL
	}

	// Send notifications to each offline recipient
	for _, recipientID := range recipients {
		go func(userID int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := h.pushService.SendNotification(ctx, userID, payload); err != nil {
				log.Printf("Error sending push notification to user %d: %v", userID, err)
			}
		}(recipientID)
	}
}

// handleReaction handles client adding a reaction via WebSocket
func (h *Handler) handleReaction(client *Client, msg ReactionMessage) {
	// Add reaction using service
	err := h.messagineService.AddReaction(msg.ReactionID, msg.MessageID, client.userID, msg.ReactionCode)
	if err != nil {
		// Check if it's a duplicate reaction (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			log.Printf("Duplicate reaction detected (ID: %s), ignoring", msg.ReactionID)
			return
		}
		log.Printf("Error adding reaction: %v", err)
		return
	}

	// Get chat ID for the message
	chatID, err := h.messagineService.GetChatIDForMessage(msg.MessageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		return
	}

	// Update reaction with user ID and current time
	msg.UserID = client.userID
	msg.ReactedAt = time.Now()
	msg.ChatID = chatID

	// Marshal message
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling reaction: %v", err)
		return
	}

	// Broadcast reaction to all participants in the chat
	h.broadcastToChat(chatID, msgData)
}

// handleTypingIndicator handles typing indicators from clients
func (h *Handler) handleTypingIndicator(client *Client, msg TypingMessage) {
	// Store typing indicator (optional, could use a cache/Redis for this)
	if err := h.messagineService.StoreTypingIndicator(client.userID, msg.ChatID); err != nil {
		log.Printf("Error storing typing indicator: %v", err)
		// Continue anyway as it's not critical
	}

	// Update with user ID and current time
	msg.UserID = client.userID
	msg.Timestamp = time.Now()

	// Marshal message
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling typing notification: %v", err)
		return
	}

	// Broadcast to other participants (excluding the sender)
	h.broadcastToChatExcept(msg.ChatID, msgData, client.userID)
}

// handleReadReceipt handles read receipts from clients
func (h *Handler) handleReadReceipt(client *Client, msg ReadReceiptMessage) {
	// Store read receipt
	if err := h.messagineService.StoreReadReceipt(client.userID, msg.ChatID, msg.MessageID); err != nil {
		log.Printf("Error storing read receipt: %v", err)
		return
	}

	// Update with user ID and current time
	msg.UserID = client.userID
	msg.ReadAt = time.Now()

	// Marshal message
	msgData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling read receipt notification: %v", err)
		return
	}

	// Broadcast read receipt to other participants
	h.broadcastToChatExcept(msg.ChatID, msgData, client.userID)
}

// broadcastToChat sends a message to all clients in a chat
func (h *Handler) broadcastToChat(chatID string, message []byte) {
	// Get all participants in the chat
	participants, err := h.messagineService.GetChatParticipantsForBroadcast(chatID)
	if err != nil {
		log.Printf("Error fetching chat participants: %v", err)
		return
	}

	// Send message to all online participants
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()

	for _, userID := range participants {
		if client, ok := h.clients[userID]; ok {
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error sending message to user %d: %v", userID, err)
			}
		}
	}
}

// broadcastToChatExcept sends a message to all clients in a chat except the specified user
func (h *Handler) broadcastToChatExcept(chatID string, message []byte, exceptUserID int) {
	participants, err := h.messagineService.GetChatParticipants(chatID)
	if err != nil {
		log.Printf("Error fetching chat participants: %v", err)
		return
	}

	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()

	for _, userID := range participants {
		if userID == exceptUserID {
			continue // Skip the excluded user
		}

		if client, ok := h.clients[userID]; ok {
			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Error sending message to user %d: %v", userID, err)
			}
		}
	}
}
