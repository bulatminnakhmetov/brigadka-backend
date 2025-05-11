package messaging

import (
	"encoding/json"
	"log"
	"time"

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
	MsgTypeJoinChat       = "join_chat"
	MsgTypeLeaveChat      = "leave_chat"
	MsgTypeReaction       = "reaction"
	MsgTypeRemoveReaction = "remove_reaction"
	MsgTypeTyping         = "typing"
	MsgTypeReadReceipt    = "read_receipt"
)

func (h *Handler) handleWSConnection(conn WSConn, userID int) {
	// Create new client
	client := &Client{
		conn:      conn,
		userID:    userID,
		chatRooms: make(map[string]struct{}),
	}

	// Add client to clients map
	h.clientsMutex.Lock()
	h.clients[userID] = client
	h.clientsMutex.Unlock()

	// Get user's chats and add them to chatRooms
	chatRooms, err := h.service.GetUserChatRooms(userID)
	if err != nil {
		log.Printf("Error fetching user chats: %v", err)
	} else {
		client.chatRooms = chatRooms
	}

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

		// Handle message based on type
		switch baseMsg.Type {
		case MsgTypeChatMessage:
			var chatMsg ChatMessage
			if err := json.Unmarshal(data, &chatMsg); err != nil {
				log.Printf("Error parsing chat message: %v", err)
				continue
			}
			h.handleChatMessage(client, chatMsg)
		case MsgTypeJoinChat:
			var joinMsg JoinMessage
			if err := json.Unmarshal(data, &joinMsg); err != nil {
				log.Printf("Error parsing join message: %v", err)
				continue
			}
			h.handleJoinChat(client, joinMsg)
		case MsgTypeLeaveChat:
			var leaveMsg LeaveMessage
			if err := json.Unmarshal(data, &leaveMsg); err != nil {
				log.Printf("Error parsing leave message: %v", err)
				continue
			}
			h.handleLeaveChat(client, leaveMsg)
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
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Store message using the service
	sentAt, err := h.service.AddMessage(msg.MessageID, msg.ChatID, client.userID, msg.Content)
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

	// Broadcast message to all participants in the chat
	h.broadcastToChat(msg.ChatID, msgData)
}

// handleJoinChat handles a client joining a chat
func (h *Handler) handleJoinChat(client *Client, msg JoinMessage) {
	// Check if user is already in the chat
	if _, ok := client.chatRooms[msg.ChatID]; ok {
		log.Printf("User %d already in chat %s", client.userID, msg.ChatID)
		return
	}

	// Check if user is authorized to join this chat
	inChat, err := h.service.IsUserInChat(client.userID, msg.ChatID)
	if err != nil {
		log.Printf("Error checking if user is in chat: %v", err)
		return
	}

	if !inChat {
		log.Printf("User %d not authorized to join chat %s", client.userID, msg.ChatID)
		return
	}

	// Add chat to client's list of active chat rooms
	client.chatRooms[msg.ChatID] = struct{}{}

	// Prepare join notification
	notification := JoinMessage{
		BaseMessage: BaseMessage{
			Type:   MsgTypeJoinChat,
			ChatID: msg.ChatID,
		},
		UserID:   client.userID,
		JoinedAt: time.Now(),
	}

	// Marshal notification
	msgData, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling join notification: %v", err)
		return
	}

	// Broadcast to other participants that this user has joined
	h.broadcastToChat(msg.ChatID, msgData)

	// Send confirmation to the client
	if err := client.conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
		log.Printf("Error sending join confirmation to user %d: %v", client.userID, err)
	}
}

// handleLeaveChat handles a client leaving a chat
func (h *Handler) handleLeaveChat(client *Client, msg LeaveMessage) {
	// Check if user is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Remove chat from client's list of active chat rooms
	delete(client.chatRooms, msg.ChatID)

	// Prepare leave notification
	notification := LeaveMessage{
		BaseMessage: BaseMessage{
			Type:   MsgTypeLeaveChat,
			ChatID: msg.ChatID,
		},
		UserID: client.userID,
		LeftAt: time.Now(),
	}

	// Marshal notification
	msgData, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling leave notification: %v", err)
		return
	}

	// Broadcast to other participants that this user has left
	h.broadcastToChat(msg.ChatID, msgData)
}

// handleReaction handles client adding a reaction via WebSocket
func (h *Handler) handleReaction(client *Client, msg ReactionMessage) {
	// Add reaction using service
	err := h.service.AddReaction(msg.ReactionID, msg.MessageID, client.userID, msg.ReactionCode)
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
	chatID, err := h.service.GetChatIDForMessage(msg.MessageID)
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
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Store typing indicator (optional, could use a cache/Redis for this)
	if err := h.service.StoreTypingIndicator(client.userID, msg.ChatID); err != nil {
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
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Store read receipt
	if err := h.service.StoreReadReceipt(client.userID, msg.ChatID, msg.MessageID); err != nil {
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
	participants, err := h.service.GetChatParticipantsForBroadcast(chatID)
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
	participants, err := h.service.GetChatParticipants(chatID)
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
