package messaging

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *MessagingRepositoryImpl) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewRepository(db)
	return db, mock, repo
}

func TestGetUserChats(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	mockTime := time.Now()

	chatRows := sqlmock.NewRows([]string{"id", "chat_name", "created_at", "is_group"}).
		AddRow("chat1", nil, mockTime, false).
		AddRow("chat2", sql.NullString{String: "Group Chat", Valid: true}, mockTime, true)

	mock.ExpectQuery(`SELECT c.id, c.chat_name, c.created_at, c.is_group FROM chats c JOIN chat_participants cp ON c.id = cp.chat_id WHERE cp.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(chatRows)

	// For the direct chat, expect query for participants
	participantRows := sqlmock.NewRows([]string{"user_id"}).
		AddRow(1).
		AddRow(2)

	mock.ExpectQuery(`SELECT user_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs("chat1").
		WillReturnRows(participantRows)

	// Group chat doesn't query for participants

	chats, err := repo.GetUserChats(userID)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(chats))
	assert.Equal(t, "chat1", chats[0].ChatID)
	assert.Equal(t, false, chats[0].IsGroup)
	assert.Nil(t, chats[0].ChatName)
	assert.Equal(t, []int{1, 2}, chats[0].Participants)

	assert.Equal(t, "chat2", chats[1].ChatID)
	assert.Equal(t, true, chats[1].IsGroup)
	assert.NotNil(t, chats[1].ChatName)
	assert.Equal(t, "Group Chat", *chats[1].ChatName)
	assert.Empty(t, chats[1].Participants) // Group chats don't load participants

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserChatsEmpty(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1

	emptyRows := sqlmock.NewRows([]string{"id", "chat_name", "created_at", "is_group"})

	mock.ExpectQuery(`SELECT c.id, c.chat_name, c.created_at, c.is_group FROM chats c JOIN chat_participants cp ON c.id = cp.chat_id WHERE cp.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(emptyRows)

	chats, err := repo.GetUserChats(userID)

	assert.NoError(t, err)
	assert.Empty(t, chats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserChatsError(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	expectedErr := errors.New("database error")

	mock.ExpectQuery(`SELECT c.id, c.chat_name, c.created_at, c.is_group FROM chats c JOIN chat_participants cp ON c.id = cp.chat_id WHERE cp.user_id = \$1`).
		WithArgs(userID).
		WillReturnError(expectedErr)

	chats, err := repo.GetUserChats(userID)

	assert.Error(t, err)
	assert.Nil(t, chats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChat(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1
	mockTime := time.Now()
	chatName := "Test Chat"

	// Check if user is in chat
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2`).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Get chat details
	mock.ExpectQuery(`SELECT c.id, c.chat_name, c.created_at, c.is_group FROM chats c WHERE c.id = \$1`).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_name", "created_at", "is_group"}).
			AddRow(chatID, chatName, mockTime, true))

	// Get participants
	mock.ExpectQuery(`SELECT user_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(1).
			AddRow(2).
			AddRow(3))

	chat, err := repo.GetChat(chatID, userID)

	assert.NoError(t, err)
	assert.NotNil(t, chat)
	assert.Equal(t, chatID, chat.ChatID)
	assert.NotNil(t, chat.ChatName)
	assert.Equal(t, chatName, *chat.ChatName)
	assert.Equal(t, true, chat.IsGroup)
	assert.Equal(t, []int{1, 2, 3}, chat.Participants)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChatUserNotInChat(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1

	// Check if user is in chat - returns 0 count
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2`).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	chat, err := repo.GetChat(chatID, userID)

	assert.Error(t, err)
	assert.Nil(t, chat)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateChat(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	ctx := context.Background()
	chatID := "chat1"
	creatorID := 1
	chatName := "New Group"
	participants := []int{1, 2, 3}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO chats \(id, chat_name, is_group\) VALUES \(\$1, \$2, true\)`).
		WithArgs(chatID, chatName).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(chatID, creatorID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Skip creator as already added
	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(chatID, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(chatID, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	err := repo.CreateChat(ctx, chatID, creatorID, chatName, participants)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateDirectChat_Existing(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	ctx := context.Background()
	userID1 := 1
	userID2 := 2
	existingChatID := "chat123"

	// Check for existing chat
	mock.ExpectQuery(`SELECT c.id FROM chats c JOIN chat_participants cp1 ON c.id = cp1.chat_id JOIN chat_participants cp2 ON c.id = cp2.chat_id WHERE c.is_group = false AND cp1.user_id = \$1 AND cp2.user_id = \$2`).
		WithArgs(userID1, userID2).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(existingChatID))

	chatID, err := repo.GetOrCreateDirectChat(ctx, userID1, userID2)

	assert.NoError(t, err)
	assert.Equal(t, existingChatID, chatID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrCreateDirectChat_New(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	ctx := context.Background()
	userID1 := 1
	userID2 := 2

	// No existing chat
	mock.ExpectQuery(`SELECT c.id FROM chats c JOIN chat_participants cp1 ON c.id = cp1.chat_id JOIN chat_participants cp2 ON c.id = cp2.chat_id WHERE c.is_group = false AND cp1.user_id = \$1 AND cp2.user_id = \$2`).
		WithArgs(userID1, userID2).
		WillReturnError(sql.ErrNoRows)

	// Create new chat
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO chats \(id, is_group\) VALUES \(\$1, false\)`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	chatID, err := repo.GetOrCreateDirectChat(ctx, userID1, userID2)

	assert.NoError(t, err)
	assert.NotEmpty(t, chatID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddMessage(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	messageID := "msg1"
	chatID := "chat1"
	senderID := 1
	content := "Hello world"
	mockTime := time.Now()

	mock.ExpectQuery(`INSERT INTO messages \(id, chat_id, sender_id, content\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING sent_at`).
		WithArgs(messageID, chatID, senderID, content).
		WillReturnRows(sqlmock.NewRows([]string{"sent_at"}).AddRow(mockTime))

	sentAt, err := repo.AddMessage(messageID, chatID, senderID, content)

	assert.NoError(t, err)
	assert.Equal(t, mockTime, sentAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChatParticipants(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"

	mock.ExpectQuery(`SELECT user_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(1).
			AddRow(2).
			AddRow(3))

	participants, err := repo.GetChatParticipants(chatID)

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, participants)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserInChat(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2`).
		WithArgs(chatID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	isInChat, err := repo.IsUserInChat(userID, chatID)

	assert.NoError(t, err)
	assert.True(t, isInChat)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddParticipant(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1

	mock.ExpectExec(`INSERT INTO chat_participants \(chat_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(chatID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.AddParticipant(chatID, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRemoveParticipant(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1

	mock.ExpectExec(`DELETE FROM chat_participants WHERE chat_id = \$1 AND user_id = \$2`).
		WithArgs(chatID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveParticipant(chatID, userID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddReaction(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	reactionID := "react1"
	messageID := "msg1"
	userID := 1
	reactionCode := "👍"

	// Check if reaction code exists
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reaction_catalog WHERE reaction_code = \$1`).
		WithArgs(reactionCode).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Add reaction
	mock.ExpectExec(`INSERT INTO message_reactions \(id, message_id, user_id, reaction_code\) VALUES \(\$1, \$2, \$3, \$4\)`).
		WithArgs(reactionID, messageID, userID, reactionCode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.AddReaction(reactionID, messageID, userID, reactionCode)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddReactionInvalidCode(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	reactionID := "react1"
	messageID := "msg1"
	userID := 1
	reactionCode := "invalid"

	// Check if reaction code exists - not found
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM reaction_catalog WHERE reaction_code = \$1`).
		WithArgs(reactionCode).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := repo.AddReaction(reactionID, messageID, userID, reactionCode)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRemoveReaction(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	messageID := "msg1"
	userID := 1
	reactionCode := "👍"

	mock.ExpectExec(`DELETE FROM message_reactions WHERE message_id = \$1 AND user_id = \$2 AND reaction_code = \$3`).
		WithArgs(messageID, userID, reactionCode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.RemoveReaction(messageID, userID, reactionCode)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChatIDForMessage(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	messageID := "msg1"
	chatID := "chat1"

	mock.ExpectQuery(`SELECT chat_id FROM messages WHERE id = \$1`).
		WithArgs(messageID).
		WillReturnRows(sqlmock.NewRows([]string{"chat_id"}).AddRow(chatID))

	result, err := repo.GetChatIDForMessage(messageID)

	assert.NoError(t, err)
	assert.Equal(t, chatID, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChatMessages(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"
	userID := 1
	limit := 10
	offset := 0
	mockTime := time.Now()

	mock.ExpectQuery(`SELECT id, chat_id, sender_id, content, sent_at FROM messages WHERE chat_id = \$1 ORDER BY sent_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs(chatID, limit, offset).
		WillReturnRows(sqlmock.NewRows([]string{"id", "chat_id", "sender_id", "content", "sent_at"}).
			AddRow("msg1", chatID, userID, "Hello", mockTime).
			AddRow("msg2", chatID, userID+1, "Hi there", mockTime.Add(-1*time.Minute)))

	messages, err := repo.GetChatMessages(chatID, userID, limit, offset)

	assert.NoError(t, err)
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, "msg1", messages[0].MessageID)
	assert.Equal(t, chatID, messages[0].ChatID)
	assert.Equal(t, userID, messages[0].SenderID)
	assert.Equal(t, "Hello", messages[0].Content)
	assert.Equal(t, mockTime, messages[0].SentAt)

	assert.Equal(t, "msg2", messages[1].MessageID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreReadReceipt(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	chatID := "chat1"
	messageID := "msg1"
	seq := int64(42)

	// Get message sequence
	mock.ExpectQuery(`SELECT seq FROM messages WHERE id = \$1`).
		WithArgs(messageID).
		WillReturnRows(sqlmock.NewRows([]string{"seq"}).AddRow(seq))

	// Store read receipt
	mock.ExpectExec(`INSERT INTO message_read_receipts \(user_id, chat_id, last_read_seq, read_at\) VALUES \(\$1, \$2, \$3, NOW\(\)\) ON CONFLICT \(user_id, chat_id\) DO UPDATE SET last_read_seq = \$3, read_at = NOW\(\)`).
		WithArgs(userID, chatID, seq).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.StoreReadReceipt(userID, chatID, messageID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserChatRooms(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1

	mock.ExpectQuery(`SELECT chat_id FROM chat_participants WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"chat_id"}).
			AddRow("chat1").
			AddRow("chat2").
			AddRow("chat3"))

	chatRooms, err := repo.GetUserChatRooms(userID)

	assert.NoError(t, err)
	assert.Equal(t, 3, len(chatRooms))
	_, ok1 := chatRooms["chat1"]
	_, ok2 := chatRooms["chat2"]
	_, ok3 := chatRooms["chat3"]
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetChatParticipantsForBroadcast(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	chatID := "chat1"

	mock.ExpectQuery(`SELECT user_id FROM chat_participants WHERE chat_id = \$1`).
		WithArgs(chatID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).
			AddRow(1).
			AddRow(2).
			AddRow(3))

	participants, err := repo.GetChatParticipantsForBroadcast(chatID)

	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, participants)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewRepository(t *testing.T) {
	db, _, _ := sqlmock.New()
	defer db.Close()

	repo := NewRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}
