package messaging

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type Handler struct {
	db           *sql.DB
	service      *ServiceImpl
	upgrader     websocket.Upgrader
	clients      map[int]*Client // Map of userID to client connection
	clientsMutex sync.RWMutex
}

type Client struct {
	conn      *websocket.Conn
	userID    int
	chatRooms map[string]struct{} // Set of chatIDs the client is in
}

func NewHandler(service Service) *Handler {
	return &Handler{
		db:      db,
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin check
			},
		},
		clients: make(map[int]*Client),
	}
}

// RegisterRoutes registers all chat-related routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/chats", h.CreateChat)
	r.Get("/chats", h.GetUserChats)
	r.Get("/chats/{chatID}", h.GetChatDetails)
	r.Get("/chats/{chatID}/messages", h.GetChatMessages)
	r.Post("/chats/{chatID}/messages", h.SendMessage)
	r.Post("/chats/{chatID}/participants", h.AddParticipant)
	r.Delete("/chats/{chatID}/participants/{userID}", h.RemoveParticipant)
	r.Post("/messages/{messageID}/reactions", h.AddReaction)
	r.Delete("/messages/{messageID}/reactions/{reactionCode}", h.RemoveReaction)
}

// Message types for WebSocket communication
const (
	MsgTypeChat        = "chat_message"
	MsgTypeJoin        = "join_chat"
	MsgTypeLeave       = "leave_chat"
	MsgTypeReaction    = "reaction"
	MsgTypeTyping      = "typing"
	MsgTypeReadReceipt = "read_receipt"
)

// WebSocket message structure
type WSMessage struct {
	Type       string          `json:"type"`
	ChatID     string          `json:"chat_id,omitempty"`
	MessageID  string          `json:"message_id,omitempty"`
	Content    string          `json:"content,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	ReactionID string          `json:"reaction_id,omitempty"`
}

// Chat message structure
type ChatMessage struct {
	MessageID string    `json:"message_id"`
	ChatID    string    `json:"chat_id"`
	SenderID  int       `json:"sender_id"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sent_at"`
}

// Reaction structure
type Reaction struct {
	ReactionID   string    `json:"reaction_id"`
	MessageID    string    `json:"message_id"`
	UserID       int       `json:"user_id"`
	ReactionCode string    `json:"reaction_code"`
	ReactedAt    time.Time `json:"reacted_at"`
}

// Chat structure
type Chat struct {
	ChatID       string    `json:"chat_id"`
	ChatName     string    `json:"chat_name"`
	CreatedAt    time.Time `json:"created_at"`
	IsGroup      bool      `json:"is_group"`
	Participants []int     `json:"participants"`
}

// HandleWebSocket upgrades HTTP connection to WebSocket and handles chat messages
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (assuming auth middleware sets this)
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Upgrade connection to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		return
	}

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
	rows, err := h.db.Query("SELECT chat_id FROM chat_participants WHERE user_id = $1", userID)
	if err != nil {
		log.Printf("Error fetching user chats: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var chatID string
			if err := rows.Scan(&chatID); err != nil {
				log.Printf("Error scanning chat ID: %v", err)
				continue
			}
			client.chatRooms[chatID] = struct{}{}
		}
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

	// Create chat message
	chatMsg := ChatMessage{
		MessageID: msg.MessageID,
		ChatID:    msg.ChatID,
		SenderID:  client.userID,
		Content:   msg.Content,
		SentAt:    sentAt,
	}

	// Marshal message to JSON
	msgBytes, err := json.Marshal(chatMsg)
	if err != nil {
		log.Printf("Error marshaling chat message: %v", err)
		return
	}

	// Create WebSocket message
	wsMsg := WSMessage{
		Type:      MsgTypeChat,
		ChatID:    msg.ChatID,
		MessageID: msg.MessageID,
		Payload:   msgBytes,
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
	rows, err := h.db.Query("SELECT user_id FROM chat_participants WHERE chat_id = $1", chatID)
	if err != nil {
		log.Printf("Error fetching chat participants: %v", err)
		return
	}
	defer rows.Close()

	var participants []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			log.Printf("Error scanning participant ID: %v", err)
			continue
		}
		participants = append(participants, userID)
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

// Handler for creating a new chat (1:1 or group)
func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	type createChatRequest struct {
		ChatID       string `json:"chat_id"`
		ChatName     string `json:"chat_name"`
		Participants []int  `json:"participants"`
	}

	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req createChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Participants) == 0 {
		http.Error(w, "At least one participant is required", http.StatusBadRequest)
		return
	}

	// Create chat using the service
	err := h.service.CreateChat(r.Context(), req.ChatID, userID, req.ChatName, req.Participants)
	if err != nil {
		// Check if it's a duplicate chat (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, "Chat already exists with this ID", http.StatusConflict)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error creating chat: %v", err)
		return
	}

	// Return created chat
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"chat_id": req.ChatID})
}

// Handler for getting user's chats
func (h *Handler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's chats
	rows, err := h.db.Query(`
        SELECT c.id, c.chat_name, c.created_at,
               (SELECT COUNT(*) > 2 FROM chat_participants WHERE chat_id = c.id) AS is_group
        FROM chats c
        JOIN chat_participants cp ON c.id = cp.chat_id
        WHERE cp.user_id = $1
        ORDER BY c.created_at DESC
    `, userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error fetching chats: %v", err)
		return
	}
	defer rows.Close()

	// Parse results
	var chats []Chat
	for rows.Next() {
		var chat Chat
		if err := rows.Scan(&chat.ChatID, &chat.ChatName, &chat.CreatedAt, &chat.IsGroup); err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error scanning chat: %v", err)
			return
		}
		chats = append(chats, chat)
	}

	// Return chats
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

// Handler for getting chat details
func (h *Handler) GetChatDetails(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Check if user is a participant in the chat
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM chat_participants WHERE chat_id = $1 AND user_id = $2", chatID, userID).Scan(&count)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error checking chat participation: %v", err)
		return
	}
	if count == 0 {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Get chat details
	var chat Chat
	err = h.db.QueryRow(`
        SELECT c.id, c.chat_name, c.created_at,
               (SELECT COUNT(*) > 2 FROM chat_participants WHERE chat_id = c.id) AS is_group
        FROM chats c WHERE c.id = $1
    `, chatID).Scan(&chat.ChatID, &chat.ChatName, &chat.CreatedAt, &chat.IsGroup)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error fetching chat details: %v", err)
		return
	}

	// Get chat participants
	rows, err := h.db.Query("SELECT user_id FROM chat_participants WHERE chat_id = $1", chatID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error fetching chat participants: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var participantID int
		if err := rows.Scan(&participantID); err != nil {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error scanning participant: %v", err)
			return
		}
		chat.Participants = append(chat.Participants, participantID)
	}

	// Return chat details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// GetChatMessages retrieves messages for a chat with pagination
func (h *Handler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Get pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 50 // Default
	offset := 0 // Default

	// Parse limit and offset
	if limitStr != "" {
		if val, err := parseInt(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	if offsetStr != "" {
		if val, err := parseInt(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	// Get messages
	messages, err := h.service.GetChatMessages(chatID, userID, limit, offset)
	if err != nil {
		if err.Error() == "user not in chat" {
			http.Error(w, "Chat not found", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error fetching messages: %v", err)
		}
		return
	}

	// Return messages
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// AddParticipant adds a participant to a chat
func (h *Handler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Parse request body
	var req struct {
		UserID int `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if the current user is in the chat (only participants can add others)
	inChat, err := h.service.IsUserInChat(userID, chatID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error checking chat participation: %v", err)
		return
	}
	if !inChat {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Add new participant
	if err := h.service.AddParticipant(chatID, req.UserID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error adding participant: %v", err)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}

// RemoveParticipant removes a participant from a chat
func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID and target user ID from URL
	chatID := chi.URLParam(r, "chatID")
	targetUserID, err := parseInt(vars["userID"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if the current user is in the chat
	inChat, err := h.service.IsUserInChat(userID, chatID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error checking chat participation: %v", err)
		return
	}
	if !inChat {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Allow users to remove themselves, or check if target is the current user
	if userID != targetUserID {
		// In a real app, check if user has permission to remove others (admin/creator)
		// For simplicity, we'll allow any participant to remove others
	}

	// Remove participant
	if err := h.service.RemoveParticipant(chatID, targetUserID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error removing participant: %v", err)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// AddReaction adds a reaction to a message
func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get message ID from URL
	messageID := chi.URLParam(r, "messageID")

	// Parse request body
	var req struct {
		ReactionID   string `json:"reaction_id"`
		ReactionCode string `json:"reaction_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Add reaction using service
	err := h.service.AddReaction(req.ReactionID, messageID, userID, req.ReactionCode)
	if err != nil {
		// Check if it's a duplicate reaction (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, "Reaction already exists with this ID", http.StatusConflict)
			return
		}

		// Other errors
		if err.Error() == "invalid reaction code" {
			http.Error(w, "Invalid reaction code", http.StatusBadRequest)
		} else if err.Error() == "user not authorized to react to this message" {
			http.Error(w, "Message not found or not authorized", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error adding reaction: %v", err)
		}
		return
	}

	// Get chat ID for the message for broadcasting
	chatID, err := h.service.GetChatIDForMessage(messageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		// Continue to return success even if we can't broadcast
	} else {
		// Broadcast reaction to chat participants
		reaction := Reaction{
			ReactionID:   req.ReactionID,
			MessageID:    messageID,
			UserID:       userID,
			ReactionCode: req.ReactionCode,
			ReactedAt:    time.Now(),
		}

		reactionData, _ := json.Marshal(reaction)
		msgData, _ := json.Marshal(WSMessage{
			Type:       MsgTypeReaction,
			ChatID:     chatID,
			MessageID:  messageID,
			ReactionID: req.ReactionID,
			Payload:    reactionData,
		})

		h.broadcastToChat(chatID, msgData)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"reaction_id": req.ReactionID,
	})
}

// RemoveReaction removes a reaction from a message
func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get message ID and reaction code from URL
	messageID := chi.URLParam(r, "messageID")
	reactionCode := chi.URLParam(r, "reactionCode")

	// Get chat ID for the message for broadcasting
	chatID, err := h.service.GetChatIDForMessage(messageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		// We'll continue even if we can't broadcast
	}

	// Remove reaction
	err = h.service.RemoveReaction(messageID, userID, reactionCode)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error removing reaction: %v", err)
		return
	}

	// Broadcast reaction removal if we have a chat ID
	if chatID != "" {
		type ReactionRemoval struct {
			MessageID    string    `json:"message_id"`
			UserID       int       `json:"user_id"`
			ReactionCode string    `json:"reaction_code"`
			RemovedAt    time.Time `json:"removed_at"`
		}

		removal := ReactionRemoval{
			MessageID:    messageID,
			UserID:       userID,
			ReactionCode: reactionCode,
			RemovedAt:    time.Now(),
		}

		removalBytes, _ := json.Marshal(removal)
		msgData, _ := json.Marshal(WSMessage{
			Type:      "reaction_removed",
			ChatID:    chatID,
			MessageID: messageID,
			Payload:   removalBytes,
		})

		h.broadcastToChat(chatID, msgData)
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// handleJoinChat handles a client joining a chat
func (h *Handler) handleJoinChat(client *Client, msg WSMessage) {
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
	type JoinNotification struct {
		UserID   int       `json:"user_id"`
		ChatID   string    `json:"chat_id"`
		JoinedAt time.Time `json:"joined_at"`
	}

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
	type LeaveNotification struct {
		UserID int       `json:"user_id"`
		ChatID string    `json:"chat_id"`
		LeftAt time.Time `json:"left_at"`
	}

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
	type ReactionRequest struct {
		MessageID    string `json:"message_id"`
		ReactionID   string `json:"reaction_id"`
		ReactionCode string `json:"reaction_code"`
	}

	var req ReactionRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("Error parsing reaction request: %v", err)
		return
	}

	// If ReactionID isn't in the payload, use the one from the message
	if req.ReactionID == "" {
		req.ReactionID = msg.ReactionID
	}

	// Add reaction using service
	err := h.service.AddReaction(req.ReactionID, req.MessageID, client.userID, req.ReactionCode)
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
	chatID, err := h.service.GetChatIDForMessage(req.MessageID)
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
		Type:       MsgTypeReaction,
		ChatID:     chatID,
		MessageID:  req.MessageID,
		ReactionID: req.ReactionID,
		Payload:    reactionBytes,
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
	if err := h.service.StoreTypingIndicator(client.userID, msg.ChatID); err != nil {
		log.Printf("Error storing typing indicator: %v", err)
		// Continue anyway as it's not critical
	}

	// Prepare typing notification
	type TypingNotification struct {
		UserID    int       `json:"user_id"`
		ChatID    string    `json:"chat_id"`
		IsTyping  bool      `json:"is_typing"`
		Timestamp time.Time `json:"timestamp"`
	}

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

// handleReadReceipt handles read receipts from clients
func (h *Handler) handleReadReceipt(client *Client, msg WSMessage) {
	// Check if client is in the chat
	if _, ok := client.chatRooms[msg.ChatID]; !ok {
		log.Printf("User %d not in chat %s", client.userID, msg.ChatID)
		return
	}

	// Parse read receipt details
	type ReadReceiptRequest struct {
		MessageID string `json:"message_id"`
	}

	var req ReadReceiptRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		log.Printf("Error parsing read receipt request: %v", err)
		return
	}

	// Store read receipt
	if err := h.service.StoreReadReceipt(client.userID, msg.ChatID, req.MessageID); err != nil {
		log.Printf("Error storing read receipt: %v", err)
		return
	}

	// Prepare read receipt notification
	type ReadReceiptNotification struct {
		UserID    int       `json:"user_id"`
		ChatID    string    `json:"chat_id"`
		MessageID string    `json:"message_id"`
		ReadAt    time.Time `json:"read_at"`
	}

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
		Type:      MsgTypeReadReceipt,
		ChatID:    msg.ChatID,
		MessageID: req.MessageID,
		Payload:   notificationBytes,
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

// SendMessage handles HTTP requests to send a message
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Parse request body
	var req struct {
		MessageID string `json:"message_id"`
		Content   string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if user is a participant in the chat
	inChat, err := h.service.IsUserInChat(userID, chatID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error checking chat participation: %v", err)
		return
	}
	if !inChat {
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	// Store message
	sentAt, err := h.service.AddMessage(req.MessageID, chatID, userID, req.Content)
	if err != nil {
		// Check if it's a duplicate message (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, "Message with this ID already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error storing message: %v", err)
		return
	}

	// Create chat message
	chatMsg := ChatMessage{
		MessageID: req.MessageID,
		ChatID:    chatID,
		SenderID:  userID,
		Content:   req.Content,
		SentAt:    sentAt,
	}

	// Marshal message for broadcasting
	msgBytes, _ := json.Marshal(chatMsg)
	wsMsg := WSMessage{
		Type:      MsgTypeChat,
		ChatID:    chatID,
		MessageID: req.MessageID,
		Payload:   msgBytes,
	}

	msgData, _ := json.Marshal(wsMsg)

	// Broadcast message to all participants in the chat
	h.broadcastToChat(chatID, msgData)

	// Return success with message details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatMsg)
}

// Helper function to parse int from string
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// Helper function to check if error is a primary key violation
func isPrimaryKeyViolation(err error) bool {
	// This implementation will depend on the specific database driver
	// For PostgreSQL, the error message contains "duplicate key value violates unique constraint"
	if err == nil {
		return false
	}

	errMsg := err.Error()
	return (errMsg != "" &&
		(errMsg == "pq: duplicate key value violates unique constraint" ||
			errMsg == "UNIQUE constraint failed" ||
			errMsg == "Duplicate entry" ||
			errMsg == "duplicate key value violates unique constraint"))
}
