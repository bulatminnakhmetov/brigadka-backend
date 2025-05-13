package messaging

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	apierrors "github.com/bulatminnakhmetov/brigadka-backend/internal/errors"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/messaging"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/service/push"
)

type PushService interface {
	SendNotification(ctx context.Context, userID int, payload push.NotificationPayload) error
}

type ProfileService interface {
	GetProfile(userID int) (*profile.Profile, error)
}

type Handler struct {
	messagineService messaging.Service
	profileService   ProfileService
	pushService      PushService
	upgrader         websocket.Upgrader
	clients          map[int]*Client // Map of userID to client connection
	clientsMutex     sync.RWMutex
}

// CreateChatRequest представляет запрос на создание чата
type CreateChatRequest struct {
	ChatID       string `json:"chat_id"`
	ChatName     string `json:"chat_name"`
	Participants []int  `json:"participants"`
}

// AddParticipantRequest представляет запрос на добавление участника в чат
type AddParticipantRequest struct {
	UserID int `json:"user_id"`
}

// AddReactionRequest представляет запрос на добавление реакции к сообщению
type AddReactionRequest struct {
	ReactionID   string `json:"reaction_id"`
	ReactionCode string `json:"reaction_code"`
}

// SendMessageRequest представляет запрос на отправку сообщения
type SendMessageRequest struct {
	MessageID string `json:"message_id"`
	Content   string `json:"content"`
}

type GetOrCreateDirectChatRequest struct {
	UserID int `json:"user_id"`
}

type ChatIDResponse struct {
	ChatID string `json:"chat_id"`
}

type AddReactionResponse struct {
	ReactionID string `json:"reaction_id"`
}

// WSConn is an interface for websocket.Conn to allow mocking in tests.
type WSConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

type Client struct {
	conn   WSConn
	userID int
}

func NewHandler(messagineService messaging.Service, profileService ProfileService, pushService PushService) *Handler {
	return &Handler{
		messagineService: messagineService,
		profileService:   profileService,
		pushService:      pushService,
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

// Reaction structure
type Reaction struct {
	ReactionID   string    `json:"reaction_id"`
	MessageID    string    `json:"message_id"`
	UserID       int       `json:"user_id"`
	ReactionCode string    `json:"reaction_code"`
	ReactedAt    time.Time `json:"reacted_at"`
}

// @Summary      Веб-сокет для чата
// @Description  Устанавливает WebSocket соединение для обмена сообщениями в реальном времени
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      101 {object} string "WebSocket connection established"
// @Failure      401 {string} string "Unauthorized"
// @Router       /ws/chat [get]
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from context (assuming auth middleware sets this)
	userID, ok := r.Context().Value("user_id").(int)
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

	h.handleWSConnection(conn, userID)
}

// @Summary      Создать новый чат
// @Description  Создает новый чат с указанными участниками
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Param        request body CreateChatRequest true "Данные для создания чата"
// @Security     BearerAuth
// @Success      201 {object} ChatIDResponse "Чат успешно создан"
// @Failure      400 {string} string "Некорректный запрос"
// @Failure      401 {string} string "Unauthorized"
// @Failure      409 {string} string "Чат с таким ID уже существует"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats [post]
func (h *Handler) CreateChat(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var req CreateChatRequest
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
	err := h.messagineService.CreateChat(r.Context(), req.ChatID, userID, req.ChatName, req.Participants)
	if err != nil {
		// Check if it's a duplicate chat (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, apierrors.ErrorChatAlreadyExistsWithThisID, http.StatusConflict)
			return
		}
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error creating chat: %v", err)
		return
	}

	response := ChatIDResponse{
		ChatID: req.ChatID,
	}

	// Return created chat
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary      Получить или создать личный чат
// @Description  Находит существующий личный чат между двумя пользователями или создает новый
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Param        request body GetOrCreateDirectChatRequest true "ID второго пользователя"
// @Security     BearerAuth
// @Success      200 {object} ChatIDResponse "ID чата"
// @Failure      400 {string} string "Некорректный запрос или попытка создать чат с самим собой"
// @Failure      401 {string} string "Unauthorized"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/direct [post]
// GetOrCreateDirectChat finds an existing direct chat or creates a new one
func (h *Handler) GetOrCreateDirectChat(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (current user)
	currentUserID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request to get the other user's ID
	var req GetOrCreateDirectChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get or create the direct chat
	chatID, err := h.messagineService.GetOrCreateDirectChat(r.Context(), currentUserID, req.UserID)
	if err != nil {
		if err.Error() == apierrors.ErrorCannotCreateChatWithSelf {
			http.Error(w, apierrors.ErrorCannotCreateChatWithSelf, http.StatusBadRequest)
			return
		}

		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error getting/creating direct chat: %v", err)
		return
	}

	response := ChatIDResponse{
		ChatID: chatID,
	}

	// Return the chat ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// @Summary      Получить чаты пользователя
// @Description  Возвращает все чаты, в которых участвует пользователь
// @Tags         messaging
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} messaging.Chat "Список чатов пользователя"
// @Failure      401 {string} string "Unauthorized"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats [get]
func (h *Handler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's chats using the service
	chats, err := h.messagineService.GetUserChats(userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error fetching chats: %v", err)
		return
	}

	// Return chats
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chats)
}

// @Summary      Получить детали чата
// @Description  Возвращает информацию о чате и его участниках
// @Tags         messaging
// @Produce      json
// @Param        chatID path string true "ID чата"
// @Security     BearerAuth
// @Success      200 {object} messaging.Chat "Детали чата"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Чат не найден"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/{chatID} [get]
func (h *Handler) GetChat(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Get chat details from the service
	chat, err := h.messagineService.GetChat(chatID, userID)
	if err != nil {
		if err.Error() == apierrors.ErrorUserNotInChat {
			http.Error(w, "Chat not found", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error fetching chat details: %v", err)
		}
		return
	}

	// Return chat details
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chat)
}

// @Summary      Получить сообщения чата
// @Description  Возвращает сообщения чата с поддержкой пагинации
// @Tags         messaging
// @Produce      json
// @Param        chatID path string true "ID чата"
// @Param        limit query int false "Максимальное количество сообщений (по умолчанию 50)"
// @Param        offset query int false "Смещение (по умолчанию 0)"
// @Security     BearerAuth
// @Success      200 {array} messaging.ChatMessage "Сообщения чата"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Чат не найден"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/{chatID}/messages [get]
func (h *Handler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
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
	messages, err := h.messagineService.GetChatMessages(chatID, userID, limit, offset)
	if err != nil {
		if err.Error() == apierrors.ErrorUserNotInChat {
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

// @Summary      Добавить участника в чат
// @Description  Добавляет нового участника в существующий чат
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Param        chatID path string true "ID чата"
// @Param        request body AddParticipantRequest true "Данные пользователя для добавления"
// @Security     BearerAuth
// @Success      201 {string} string "Участник успешно добавлен"
// @Failure      400 {string} string "Некорректный запрос"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Чат не найден"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/{chatID}/participants [post]
func (h *Handler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Parse request body
	var req AddParticipantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if the current user is in the chat (only participants can add others)
	inChat, err := h.messagineService.IsUserInChat(userID, chatID)
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
	if err := h.messagineService.AddParticipant(chatID, req.UserID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error adding participant: %v", err)
		return
	}

	// Return success
	w.WriteHeader(http.StatusCreated)
}

// @Summary      Удалить участника из чата
// @Description  Удаляет участника из чата (пользователь может удалить только себя)
// @Tags         messaging
// @Produce      json
// @Param        chatID path string true "ID чата"
// @Param        userID path int true "ID пользователя для удаления"
// @Security     BearerAuth
// @Success      200 {string} string "Участник успешно удален"
// @Failure      400 {string} string "Некорректный запрос"
// @Failure      401 {string} string "Unauthorized"
// @Failure      403 {string} string "Нет прав на удаление этого пользователя"
// @Failure      404 {string} string "Чат не найден"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/{chatID}/participants/{userID} [delete]
func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID and target user ID from URL
	chatID := chi.URLParam(r, "chatID")
	targetUserID, err := parseInt(chi.URLParam(r, "userID"))

	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Check if the current user is in the chat
	inChat, err := h.messagineService.IsUserInChat(userID, chatID)
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
		http.Error(w, "Not authorized to remove this user", http.StatusForbidden)
		return
	}

	// Remove participant
	if err := h.messagineService.RemoveParticipant(chatID, targetUserID); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error removing participant: %v", err)
		return
	}

	// Return success
	w.WriteHeader(http.StatusOK)
}

// @Summary      Добавить реакцию к сообщению
// @Description  Добавляет эмоциональную реакцию к сообщению
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Param        messageID path string true "ID сообщения"
// @Param        request body AddReactionRequest true "Данные реакции"
// @Security     BearerAuth
// @Success      200 {object} AddReactionResponse "Реакция успешно добавлена"
// @Failure      400 {string} string "Некорректный запрос"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Сообщение не найдено или нет прав для реакции"
// @Failure      409 {string} string "Реакция с таким ID уже существует"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /messages/{messageID}/reactions [post]
func (h *Handler) AddReaction(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get message ID from URL
	messageID := chi.URLParam(r, "messageID")

	// Parse request body
	var req AddReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Add reaction using service
	err := h.messagineService.AddReaction(req.ReactionID, messageID, userID, req.ReactionCode)
	if err != nil {
		// Check if it's a duplicate reaction (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, apierrors.ErrorReactionAlreadyExists, http.StatusConflict)
			return
		}

		// Other errors
		if err.Error() == apierrors.ErrorInvalidReactionCode {
			http.Error(w, apierrors.ErrorInvalidReactionCode, http.StatusBadRequest)
		} else if err.Error() == apierrors.ErrorNotAuthorizedToReact {
			http.Error(w, "Message not found or not authorized", http.StatusNotFound)
		} else {
			http.Error(w, "Server error", http.StatusInternalServerError)
			log.Printf("Error adding reaction: %v", err)
		}
		return
	}

	// Get chat ID for the message for broadcasting
	chatID, err := h.messagineService.GetChatIDForMessage(messageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		// Continue to return success even if we can't broadcast
	} else {
		// Broadcast reaction to chat participants
		msgData, _ := json.Marshal(ReactionMessage{
			BaseMessage: BaseMessage{
				Type:   MsgTypeReaction,
				ChatID: chatID,
			},
			ReactionID:   req.ReactionID,
			MessageID:    messageID,
			UserID:       userID,
			ReactionCode: req.ReactionCode,
			ReactedAt:    time.Now(),
		})

		h.broadcastToChat(chatID, msgData)
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AddReactionResponse{ReactionID: req.ReactionID})
}

// @Summary      Удалить реакцию с сообщения
// @Description  Удаляет эмоциональную реакцию с сообщения
// @Tags         messaging
// @Produce      json
// @Param        messageID path string true "ID сообщения"
// @Param        reactionCode path string true "Код реакции для удаления"
// @Security     BearerAuth
// @Success      200 {object} map[string]string "Реакция успешно удалена"
// @Failure      401 {string} string "Unauthorized"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /messages/{messageID}/reactions/{reactionCode} [delete]
func (h *Handler) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get message ID and reaction code from URL
	messageID := chi.URLParam(r, "messageID")
	reactionCode := chi.URLParam(r, "reactionCode")

	// Get chat ID for the message for broadcasting
	chatID, err := h.messagineService.GetChatIDForMessage(messageID)
	if err != nil {
		log.Printf("Error getting chat ID for message: %v", err)
		// We'll continue even if we can't broadcast
	}

	// Remove reaction
	err = h.messagineService.RemoveReaction(messageID, userID, reactionCode)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error removing reaction: %v", err)
		return
	}

	// Broadcast reaction removal if we have a chat ID
	if chatID != "" {
		msgData, _ := json.Marshal(ReactionRemovedMessage{
			BaseMessage: BaseMessage{
				Type:   MsgTypeRemoveReaction,
				ChatID: chatID,
			},
			MessageID:    messageID,
			UserID:       userID,
			ReactionCode: reactionCode,
			RemovedAt:    time.Now(),
		})

		h.broadcastToChat(chatID, msgData)
	}

	// Return success
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// @Summary      Отправить сообщение
// @Description  Отправляет новое сообщение в чат
// @Tags         messaging
// @Accept       json
// @Produce      json
// @Param        chatID path string true "ID чата"
// @Param        request body SendMessageRequest true "Данные сообщения"
// @Security     BearerAuth
// @Success      200 {object} ChatMessage "Сообщение успешно отправлено"
// @Failure      400 {string} string "Некорректный запрос"
// @Failure      401 {string} string "Unauthorized"
// @Failure      404 {string} string "Чат не найден"
// @Failure      409 {string} string "Сообщение с таким ID уже существует"
// @Failure      500 {string} string "Ошибка сервера"
// @Router       /chats/{chatID}/messages [post]
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get chat ID from URL
	chatID := chi.URLParam(r, "chatID")

	// Parse request body
	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store message
	sentAt, err := h.messagineService.AddMessage(req.MessageID, chatID, userID, req.Content)
	if err != nil {
		// Check if it's a duplicate message (UUID constraint violation)
		if isPrimaryKeyViolation(err) {
			http.Error(w, apierrors.ErrorMessageAlreadyExists, http.StatusConflict)
			return
		}

		// Check for user not in chat
		if err.Error() == apierrors.ErrorUserNotInChat {
			http.Error(w, "Chat not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Server error", http.StatusInternalServerError)
		log.Printf("Error storing message: %v", err)
		return
	}

	// Marshal message for broadcasting
	wsMsg := ChatMessage{
		BaseMessage: BaseMessage{
			Type:   MsgTypeChatMessage,
			ChatID: chatID,
		},
		MessageID: req.MessageID,
		SenderID:  userID,
		Content:   req.Content,
		SentAt:    sentAt,
	}

	msgData, _ := json.Marshal(wsMsg)

	// Broadcast message to all participants in the chat
	h.broadcastToChat(chatID, msgData)

	// Return success with message details
	w.Header().Set("Content-Type", "application/json")
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
