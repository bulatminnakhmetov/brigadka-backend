package messaging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserChats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)

	// Test successful retrieval
	t.Run("Success", func(t *testing.T) {
		userID := 1
		currentTime := time.Now()

		rows := sqlmock.NewRows([]string{"id", "chat_name", "created_at", "is_group"}).
			AddRow("chat-id-1", "Chat 1", currentTime, true).
			AddRow("chat-id-2", "Chat 2", currentTime, false)

		mock.ExpectQuery("SELECT c.id, c.chat_name, c.created_at").
			WithArgs(userID).
			WillReturnRows(rows)

		chats, err := service.GetUserChats(userID)
		assert.NoError(t, err)
		assert.Len(t, chats, 2)
		assert.Equal(t, "chat-id-1", chats[0].ChatID)
		assert.Equal(t, "Chat 1", chats[0].ChatName)
		assert.True(t, chats[0].IsGroup)
		assert.Equal(t, currentTime, chats[0].CreatedAt)
	})

	// Test database error
	t.Run("DatabaseError", func(t *testing.T) {
		userID := 1
		mock.ExpectQuery("SELECT c.id, c.chat_name, c.created_at").
			WithArgs(userID).
			WillReturnError(errors.New("database error"))

		chats, err := service.GetUserChats(userID)
		assert.Error(t, err)
		assert.Nil(t, chats)
	})
}

func TestGetChatDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)

	t.Run("UserNotInChat", func(t *testing.T) {
		chatID := "chat-id-1"
		userID := 1

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		chat, err := service.GetChatDetails(chatID, userID)
		assert.Error(t, err)
		assert.Equal(t, "user not in chat", err.Error())
		assert.Nil(t, chat)
	})

	t.Run("Success", func(t *testing.T) {
		chatID := "chat-id-1"
		userID := 1
		createdAt := time.Now()

		// User is in chat
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Chat details
		mock.ExpectQuery("SELECT c.id, c.chat_name, c.created_at").
			WithArgs(chatID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "chat_name", "created_at", "is_group"}).
				AddRow(chatID, "Test Chat", createdAt, true))

		// Participants
		mock.ExpectQuery("SELECT user_id FROM chat_participants").
			WithArgs(chatID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
				AddRow(1).
				AddRow(2).
				AddRow(3))

		chat, err := service.GetChatDetails(chatID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, chat)
		assert.Equal(t, chatID, chat.ChatID)
		assert.Equal(t, "Test Chat", chat.ChatName)
		assert.True(t, chat.IsGroup)
		assert.Equal(t, 3, len(chat.Participants))
		assert.Contains(t, chat.Participants, 1)
		assert.Contains(t, chat.Participants, 2)
		assert.Contains(t, chat.Participants, 3)
	})
}

func TestCreateChat(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		chatID := "new-chat-id"
		creatorID := 1
		chatName := "New Chat"
		participants := []int{2, 3}

		mock.ExpectBegin()

		// Insert chat
		mock.ExpectExec("INSERT INTO chats").
			WithArgs(chatID, chatName).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Add creator
		mock.ExpectExec("INSERT INTO chat_participants").
			WithArgs(chatID, creatorID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Add participants
		for _, p := range participants {
			mock.ExpectExec("INSERT INTO chat_participants").
				WithArgs(chatID, p).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		mock.ExpectCommit()

		err := service.CreateChat(ctx, chatID, creatorID, chatName, participants)
		assert.NoError(t, err)
	})

	t.Run("TransactionError", func(t *testing.T) {
		chatID := "new-chat-id"
		creatorID := 1
		chatName := "New Chat"
		participants := []int{2, 3}

		mock.ExpectBegin().WillReturnError(errors.New("tx error"))

		err := service.CreateChat(ctx, chatID, creatorID, chatName, participants)
		assert.Error(t, err)
	})

	t.Run("ErrorInsertingChat", func(t *testing.T) {
		chatID := "new-chat-id"
		creatorID := 1
		chatName := "New Chat"
		participants := []int{2, 3}

		mock.ExpectBegin()

		// Insert chat with error
		mock.ExpectExec("INSERT INTO chats").
			WithArgs(chatID, chatName).
			WillReturnError(errors.New("chat insert error"))

		mock.ExpectRollback()

		err := service.CreateChat(ctx, chatID, creatorID, chatName, participants)
		assert.Error(t, err)
	})
}

func TestAddMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)

	t.Run("Success", func(t *testing.T) {
		messageID := "msg-id-1"
		chatID := "chat-id-1"
		senderID := 1
		content := "Hello, world!"
		sentAt := time.Now()

		mock.ExpectQuery("INSERT INTO messages").
			WithArgs(messageID, chatID, senderID, content).
			WillReturnRows(sqlmock.NewRows([]string{"sent_at"}).AddRow(sentAt))

		result, err := service.AddMessage(messageID, chatID, senderID, content)
		assert.NoError(t, err)
		assert.Equal(t, sentAt, result)
	})

	t.Run("Error", func(t *testing.T) {
		messageID := "msg-id-1"
		chatID := "chat-id-1"
		senderID := 1
		content := "Hello, world!"

		mock.ExpectQuery("INSERT INTO messages").
			WithArgs(messageID, chatID, senderID, content).
			WillReturnError(errors.New("insert error"))

		_, err := service.AddMessage(messageID, chatID, senderID, content)
		assert.Error(t, err)
	})
}

func TestIsUserInChat(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)

	t.Run("UserInChat", func(t *testing.T) {
		userID := 1
		chatID := "chat-id-1"

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		result, err := service.IsUserInChat(userID, chatID)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("UserNotInChat", func(t *testing.T) {
		userID := 1
		chatID := "chat-id-1"

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		result, err := service.IsUserInChat(userID, chatID)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("Error", func(t *testing.T) {
		userID := 1
		chatID := "chat-id-1"

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnError(errors.New("database error"))

		result, err := service.IsUserInChat(userID, chatID)
		assert.Error(t, err)
		assert.False(t, result)
	})
}

func TestGetChatMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db)

	t.Run("UserNotInChat", func(t *testing.T) {
		userID := 1
		chatID := "chat-id-1"
		limit, offset := 10, 0

		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		messages, err := service.GetChatMessages(chatID, userID, limit, offset)
		assert.Error(t, err)
		assert.Equal(t, "user not in chat", err.Error())
		assert.Nil(t, messages)
	})

	t.Run("Success", func(t *testing.T) {
		userID := 1
		chatID := "chat-id-1"
		limit, offset := 10, 0
		sentAt := time.Now()

		// User is in chat
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(chatID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		// Messages
		mock.ExpectQuery("SELECT id, chat_id, sender_id, content, sent_at").
			WithArgs(chatID, limit, offset).
			WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "content", "sent_at"}).
				AddRow("msg-1", chatID, 2, "Hello", sentAt).
				AddRow("msg-2", chatID, 1, "Hi back", sentAt.Add(time.Minute)))

		messages, err := service.GetChatMessages(chatID, userID, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "msg-1", messages[0].MessageID)
		assert.Equal(t, chatID, messages[0].ChatID)
		assert.Equal(t, 2, messages[0].SenderID)
		assert.Equal(t, "Hello", messages[0].Content)
	})
}
