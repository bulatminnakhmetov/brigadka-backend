package messaging

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type ReadReceiptNotification struct {
	UserID    int       `json:"user_id"`
	ChatID    string    `json:"chat_id"`
	MessageID string    `json:"message_id"`
	ReadAt    time.Time `json:"read_at"`
}

type JoinNotification struct {
	UserID   int       `json:"user_id"`
	ChatID   string    `json:"chat_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type LeaveNotification struct {
	UserID int       `json:"user_id"`
	ChatID string    `json:"chat_id"`
	LeftAt time.Time `json:"left_at"`
}

type TypingNotification struct {
	UserID    int       `json:"user_id"`
	ChatID    string    `json:"chat_id"`
	IsTyping  bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}

type ReadReceiptPayload struct {
	MessageID string `json:"message_id"`
}

// handleJoinChat handles a client joining a chat
func (h *Handler) handleJoinChat(client *Client, msg WSMessage) {
	// Check if user is already in the chat
	if _, ok := client.chatRooms[msg.ChatID]; ok {
		log.Printf("User %d already in chat %s", client.userID, msg.ChatID)
		return
	}

	// Check if user is authorized to join this chat
	inChat, err := h.repo.IsUserInChat(client.userID, msg.ChatID)
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

	notification := JoinNotification{
		UserID:   client.userID,
		ChatID:   msg.ChatID,
		JoinedAt: time.Now(),
	}

	// Marshal notification
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling join notification: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeJoin,
		ChatID:  msg.ChatID,
		Payload: notificationBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
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
func (h *Handler) handleLeaveChat(client *Client, msg WSMessage) {
	// Check if user is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Remove chat from client's list of active chat rooms
	delete(client.chatRooms, msg.ChatID)

	// Prepare leave notification

	notification := LeaveNotification{
		UserID: client.userID,
		ChatID: msg.ChatID,
		LeftAt: time.Now(),
	}

	// Marshal notification
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling leave notification: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeLeave,
		ChatID:  msg.ChatID,
		Payload: notificationBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast to other participants that this user has left
	h.broadcastToChat(msg.ChatID, msgData)
}

// handleReaction handles client adding a reaction via WebSocket
func (h *Handler) handleReaction(client *Client, msg WSMessage) {
	// Parse payload to get reaction details
	var req Reaction
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("Error parsing reaction request: %v", err)
		return
	}

	// Add reaction using service
	err := h.repo.AddReaction(req.ReactionID, req.MessageID, client.userID, req.ReactionCode)
	if err != nil {
		// Check if it's a duplicate reaction (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			log.Printf("Duplicate reaction detected (ID: %s), ignoring", req.ReactionID)
			return
		}
		log.Printf("Error adding reaction: %v", err)
		return
	}

	// Get chat ID for the message
	chatID, err := h.repo.GetChatIDForMessage(req.MessageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		return
	}

	// Create reaction notification
	reaction := Reaction{
		ReactionID:   req.ReactionID,
		MessageID:    req.MessageID,
		UserID:       client.userID,
		ReactionCode: req.ReactionCode,
		ReactedAt:    time.Now(),
	}

	// Marshal reaction
	reactionBytes, err := json.Marshal(reaction)
	if err != nil {
		log.Printf("Error marshaling reaction: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeReaction,
		ChatID:  chatID,
		Payload: reactionBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast reaction to all participants in the chat
	h.broadcastToChat(chatID, msgData)
}

// handleTypingIndicator handles typing indicators from clients
func (h *Handler) handleTypingIndicator(client *Client, msg WSMessage) {
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Store typing indicator (optional, could use a cache/Redis for this)
	if err := h.repo.StoreTypingIndicator(client.userID, msg.ChatID); err != nil {
		log.Printf("Error storing typing indicator: %v", err)
		// Continue anyway as it's not critical
	}

	// Prepare typing notification

	// Parse payload to get typing status
	var isTyping bool
	if err := json.Unmarshal(msg.Payload, &isTyping); err != nil {
		// Default to true if payload parsing fails
		isTyping = true
	}

	notification := TypingNotification{
		UserID:    client.userID,
		ChatID:    msg.ChatID,
		IsTyping:  isTyping,
		Timestamp: time.Now(),
	}

	// Marshal notification
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling typing notification: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeTyping,
		ChatID:  msg.ChatID,
		Payload: notificationBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast to other participants (excluding the sender)
	h.broadcastToChatExcept(msg.ChatID, msgData, client.userID)
}

// broadcastToChatExcept sends a message to all clients in a chat except the specified user
func (h *Handler) broadcastToChatExcept(chatID string, message []byte, exceptUserID int) {
	participants, err := h.repo.GetChatParticipants(chatID)
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

// handleReadReceipt handles read receipts from clients
func (h *Handler) handleReadReceipt(client *Client, msg WSMessage) {
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Parse read receipt details
	var req ReadReceiptPayload
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("Error parsing read receipt request: %v", err)
		return
	}

	// Store read receipt
	if err := h.repo.StoreReadReceipt(client.userID, msg.ChatID, req.MessageID); err != nil {
		log.Printf("Error storing read receipt: %v", err)
		return
	}

	// Prepare read receipt notification

	notification := ReadReceiptNotification{
		UserID:    client.userID,
		ChatID:    msg.ChatID,
		MessageID: req.MessageID,
		ReadAt:    time.Now(),
	}

	// Marshal notification
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling read receipt notification: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeReadReceipt,
		ChatID:  msg.ChatID,
		Payload: notificationBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast read receipt to other participants
	h.broadcastToChatExcept(msg.ChatID, msgData, client.userID)
}

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
	chatRooms, err := h.repo.GetUserChatRooms(userID)
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

		// Parse message
		var msg WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// Handle message based on type
		switch msg.Type {
		case MsgTypeChat:
			h.handleChatMessage(client, msg)
		case MsgTypeJoin:
			h.handleJoinChat(client, msg)
		case MsgTypeLeave:
			h.handleLeaveChat(client, msg)
		case MsgTypeReaction:
			h.handleReaction(client, msg)
		case MsgTypeTyping:
			h.handleTypingIndicator(client, msg)
		case MsgTypeReadReceipt:
			h.handleReadReceipt(client, msg)
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

// handleChatMessage handles a chat message from a client
func (h *Handler) handleChatMessage(client *Client, msg WSMessage) {
	// Parse payload to get message details
	var chatMsg ChatMessage
	if err := json.Unmarshal(msg.Payload, &chatMsg); err != nil {
		log.Printf("Error parsing chat message: %v", err)
		return
	}

	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Store message using the service
	sentAt, err := h.repo.AddMessage(chatMsg.MessageID, msg.ChatID, client.userID, chatMsg.Content)
	if err != nil {
		// Check if it's a duplicate message (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			log.Printf("Duplicate message detected (ID: %s), ignoring", chatMsg.MessageID)
			return
		}
		log.Printf("Error storing message: %v", err)
		return
	}

	// Update the sent time in the message
	chatMsg.SentAt = sentAt
	chatMsg.SenderID = client.userID
	chatMsg.ChatID = msg.ChatID

	// Marshal message to JSON
	msgBytes, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Error marshaling chat message: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:    MsgTypeChat,
		ChatID:  msg.ChatID,
		Payload: msgBytes,
	}

	// Marshal WebSocket message
	msgData, err := json.Marshal(wsMsg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	// Broadcast message to all participants in the chat
	h.broadcastToChat(msg.ChatID, msgData)
}

// broadcastToChat sends a message to all clients in a chat
func (h *Handler) broadcastToChat(chatID string, message []byte) {
	// Get all participants in the chat
	participants, err := h.repo.GetChatParticipantsForBroadcast(chatID)
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
