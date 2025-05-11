package messaging

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Chat message structure
type ChatMessage struct {
	MessageID string    `json:"message_id"`
	ChatID    string    `json:"chat_id"`
	SenderID  int       `json:"sender_id"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sent_at"`
}

// Chat structure
type Chat struct {
	ChatID       string    `json:"chat_id"`
	ChatName     string    `json:"chat_name"`
	CreatedAt    time.Time `json:"created_at"`
	IsGroup      bool      `json:"is_group"`
	Participants []int     `json:"participants"`
}

type MessagingService interface {
	GetUserChats(userID int) ([]Chat, error)
	GetChatDetails(chatID string, userID int) (*Chat, error)
	CreateChat(ctx context.Context, chatID string, creatorID int, chatName string, participants []int) error
	AddMessage(messageID string, chatID string, senderID int, content string) (time.Time, error)
	GetChatParticipants(chatID string) ([]int, error)
	IsUserInChat(userID int, chatID string) (bool, error)
	AddParticipant(chatID string, userID int) error
	RemoveParticipant(chatID string, userID int) error
	AddReaction(reactionID string, messageID string, userID int, reactionCode string) error
	RemoveReaction(messageID string, userID int, reactionCode string) error
	GetChatIDForMessage(messageID string) (string, error)
	GetChatMessages(chatID string, userID int, limit, offset int) ([]ChatMessage, error)
	StoreTypingIndicator(userID int, chatID string) error
	StoreReadReceipt(userID int, chatID string, messageID string) error
	GetUserChatRooms(userID int) (map[string]struct{}, error)
	GetChatParticipantsForBroadcast(chatID string) ([]int, error)
	GetOrCreateDirectChat(ctx context.Context, userID1 int, userID2 int) (string, error)
}

// MessagingServiceImpl encapsulates database operations for messaging
type MessagingServiceImpl struct {
	db *sql.DB
}

// NewService creates a new messaging service
func NewService(db *sql.DB) *MessagingServiceImpl {
	return &MessagingServiceImpl{
		db: db,
	}
}

// GetUserChats retrieves all chats for a user
func (s *MessagingServiceImpl) GetUserChats(userID int) ([]Chat, error) {
	rows, err := s.db.Query(`
        SELECT c.id, c.chat_name, c.created_at, c.is_group
        FROM chats c
        JOIN chat_participants cp ON c.id = cp.chat_id
        WHERE cp.user_id = $1
        ORDER BY c.created_at DESC
    `, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []Chat
	for rows.Next() {
		var chat Chat
		if err := rows.Scan(&chat.ChatID, &chat.ChatName, &chat.CreatedAt, &chat.IsGroup); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, nil
}

// GetChatDetails retrieves details for a specific chat
func (s *MessagingServiceImpl) GetChatDetails(chatID string, userID int) (*Chat, error) {
	// Check if user is a participant in the chat
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM chat_participants WHERE chat_id = $1 AND user_id = $2", chatID, userID).Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("user not in chat")
	}

	// Get chat details
	var chat Chat
	err = s.db.QueryRow(`
        SELECT c.id, c.chat_name, c.created_at, c.is_group
        FROM chats c WHERE c.id = $1
    `, chatID).Scan(&chat.ChatID, &chat.ChatName, &chat.CreatedAt, &chat.IsGroup)
	if err != nil {
		return nil, err
	}

	// Get chat participants
	rows, err := s.db.Query("SELECT user_id FROM chat_participants WHERE chat_id = $1", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var participantID int
		if err := rows.Scan(&participantID); err != nil {
			return nil, err
		}
		chat.Participants = append(chat.Participants, participantID)
	}

	return &chat, nil
}

// CreateChat creates a new chat with the specified participants
func (s *MessagingServiceImpl) CreateChat(ctx context.Context, chatID string, creatorID int, chatName string, participants []int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create chat
	_, err = tx.Exec("INSERT INTO chats (id, chat_name, is_group) VALUES ($1, $2, true)", chatID, chatName)
	if err != nil {
		return err
	}

	// Add creator as a participant
	_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)", chatID, creatorID)
	if err != nil {
		return err
	}

	// Add participants
	for _, participantID := range participants {
		if participantID == creatorID {
			continue // Creator already added
		}
		_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)", chatID, participantID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// GetOrCreateDirectChat finds an existing direct chat between two users or creates a new one
func (s *MessagingServiceImpl) GetOrCreateDirectChat(ctx context.Context, userID1 int, userID2 int) (string, error) {
	// First try to find an existing direct chat
	var chatID string
	err := s.db.QueryRow(`
        SELECT c.id FROM chats c
        JOIN chat_participants cp1 ON c.id = cp1.chat_id
        JOIN chat_participants cp2 ON c.id = cp2.chat_id
        WHERE c.is_group = false
        AND cp1.user_id = $1 AND cp2.user_id = $2
    `, userID1, userID2).Scan(&chatID)

	// If found, return it
	if err == nil {
		return chatID, nil
	}

	// If error is not "no rows", return the error
	if err != sql.ErrNoRows {
		return "", err
	}

	// Otherwise create a new direct chat
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Generate a new UUID for the chat
	chatID = uuid.New().String()

	// Create chat
	_, err = tx.Exec("INSERT INTO chats (id, is_group) VALUES ($1, false)", chatID)
	if err != nil {
		return "", err
	}

	// Add both users as participants
	_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)", chatID, userID1)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)", chatID, userID2)
	if err != nil {
		return "", err
	}

	if err = tx.Commit(); err != nil {
		return "", err
	}

	return chatID, nil
}

// AddMessage adds a message to the database and returns the sent time
func (s *MessagingServiceImpl) AddMessage(messageID string, chatID string, senderID int, content string) (time.Time, error) {
	var sentAt time.Time
	err := s.db.QueryRow(
		"INSERT INTO messages (id, chat_id, sender_id, content) VALUES ($1, $2, $3, $4) RETURNING sent_at",
		messageID, chatID, senderID, content,
	).Scan(&sentAt)
	if err != nil {
		return time.Time{}, err
	}
	return sentAt, nil
}

// GetChatParticipants retrieves all participants in a chat
func (s *MessagingServiceImpl) GetChatParticipants(chatID string) ([]int, error) {
	rows, err := s.db.Query("SELECT user_id FROM chat_participants WHERE chat_id = $1", chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []int
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		participants = append(participants, userID)
	}
	return participants, nil
}

// IsUserInChat checks if a user is a participant in a chat
func (s *MessagingServiceImpl) IsUserInChat(userID int, chatID string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM chat_participants WHERE chat_id = $1 AND user_id = $2", chatID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddParticipant adds a user to a chat
func (s *MessagingServiceImpl) AddParticipant(chatID string, userID int) error {
	_, err := s.db.Exec("INSERT INTO chat_participants (chat_id, user_id) VALUES ($1, $2)", chatID, userID)
	return err
}

// RemoveParticipant removes a user from a chat
func (s *MessagingServiceImpl) RemoveParticipant(chatID string, userID int) error {
	_, err := s.db.Exec("DELETE FROM chat_participants WHERE chat_id = $1 AND user_id = $2", chatID, userID)
	return err
}

// AddReaction adds a reaction to a message
func (s *MessagingServiceImpl) AddReaction(reactionID string, messageID string, userID int, reactionCode string) error {
	// Check if reaction code exists
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM reaction_catalog WHERE reaction_code = $1", reactionCode).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("invalid reaction code")
	}

	// Check if user can react to the message (is participant in the chat)
	err = s.db.QueryRow(`
        SELECT COUNT(*) FROM chat_participants cp
        JOIN messages m ON cp.chat_id = m.chat_id
        WHERE m.id = $1 AND cp.user_id = $2
    `, messageID, userID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("user not authorized to react to this message")
	}

	// Add reaction - will fail with constraint error if duplicate
	_, err = s.db.Exec(`
        INSERT INTO message_reactions (id, message_id, user_id, reaction_code)
        VALUES ($1, $2, $3, $4)
    `, reactionID, messageID, userID, reactionCode)

	return err
}

// RemoveReaction removes a reaction from a message
func (s *MessagingServiceImpl) RemoveReaction(messageID string, userID int, reactionCode string) error {
	_, err := s.db.Exec(
		"DELETE FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction_code = $3",
		messageID, userID, reactionCode,
	)
	return err
}

// GetChatIDForMessage retrieves the chat ID for a message
func (s *MessagingServiceImpl) GetChatIDForMessage(messageID string) (string, error) {
	var chatID string
	err := s.db.QueryRow("SELECT chat_id FROM messages WHERE id = $1", messageID).Scan(&chatID)
	return chatID, err
}

// GetChatMessages retrieves messages for a chat with pagination
func (s *MessagingServiceImpl) GetChatMessages(chatID string, userID int, limit, offset int) ([]ChatMessage, error) {
	// Check if user is in chat
	inChat, err := s.IsUserInChat(userID, chatID)
	if err != nil {
		return nil, err
	}
	if !inChat {
		return nil, errors.New("user not in chat")
	}

	// Get messages
	rows, err := s.db.Query(`
        SELECT id, chat_id, sender_id, content, sent_at
        FROM messages
        WHERE chat_id = $1
        ORDER BY sent_at DESC
        LIMIT $2 OFFSET $3
    `, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.MessageID, &msg.ChatID, &msg.SenderID, &msg.Content, &msg.SentAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// StoreTypingIndicator records that a user is typing in a chat
// This could use a cache/Redis instead of DB for better performance
func (s *MessagingServiceImpl) StoreTypingIndicator(userID int, chatID string) error {
	// Implementation would depend on how you want to track typing indicators
	// This is a simple example that could be replaced with Redis
	return nil
}

// StoreReadReceipt records that a user has read messages up to a certain point
func (s *MessagingServiceImpl) StoreReadReceipt(userID int, chatID string, messageID string) error {
	// First, get the sequence number for the message
	var seq int64
	err := s.db.QueryRow("SELECT seq FROM messages WHERE id = $1", messageID).Scan(&seq)
	if err != nil {
		return err
	}

	// Now update the read receipt with the sequence number
	_, err = s.db.Exec(`
        INSERT INTO message_read_receipts (user_id, chat_id, last_read_seq, read_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (user_id, chat_id) DO UPDATE 
        SET last_read_seq = $3, read_at = NOW()
    `, userID, chatID, seq)
	return err
}

// GetUserChatRooms retrieves all chat IDs a user is part of
func (s *MessagingServiceImpl) GetUserChatRooms(userID int) (map[string]struct{}, error) {
	rows, err := s.db.Query("SELECT chat_id FROM chat_participants WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chatRooms := make(map[string]struct{})
	for rows.Next() {
		var chatID string
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatRooms[chatID] = struct{}{}
	}

	return chatRooms, nil
}

// GetChatParticipantsForBroadcast retrieves all participants of a chat for broadcasting
func (s *MessagingServiceImpl) GetChatParticipantsForBroadcast(chatID string) ([]int, error) {
	return s.GetChatParticipants(chatID)
}
