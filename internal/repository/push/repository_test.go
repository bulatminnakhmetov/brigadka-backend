package push

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *postgresRepository) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	repo := NewPostgresRepository(db).(*postgresRepository)
	return db, mock, repo
}

func TestSaveToken(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	token := PushToken{
		UserID:   1,
		Token:    "test-token",
		Platform: "ios",
		DeviceID: "device-123",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO push_tokens (user_id, token, platform, device_id, last_seen_at)
        VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (token) DO UPDATE 
        SET user_id = $1, platform = $3, device_id = $4, last_seen_at = NOW(), updated_at = NOW()
        RETURNING id`)).
		WithArgs(token.UserID, token.Token, token.Platform, token.DeviceID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	id, err := repo.SaveToken(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, 1, id)
}

func TestGetUserTokens(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, user_id, token, platform, device_id, last_seen_at, created_at, updated_at
        FROM push_tokens
        WHERE user_id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "user_id", "token", "platform", "device_id", "last_seen_at", "created_at", "updated_at",
		}).AddRow(1, 1, "token1", "ios", "device1", time.Now(), time.Now(), time.Now()).
			AddRow(2, 1, "token2", "android", "device2", time.Now(), time.Now(), time.Now()))

	tokens, err := repo.GetUserTokens(context.Background(), 1)
	assert.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.Equal(t, "token1", tokens[0].Token)
	assert.Equal(t, "token2", tokens[1].Token)
}

func TestIsTokenExists(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT EXISTS(SELECT 1 FROM push_tokens WHERE token = $1 AND user_id = $2)`)).
		WithArgs("test-token", 1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.IsTokenExists(context.Background(), "test-token", 1)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestDeleteToken(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
        DELETE FROM push_tokens WHERE token = $1 AND user_id = $2`)).
		WithArgs("test-token", 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteToken(context.Background(), 1, "test-token")
	assert.NoError(t, err)
}

func TestUpdateLastSeen(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(`
        UPDATE push_tokens 
        SET last_seen_at = NOW(), updated_at = NOW() 
        WHERE token = $1`)).
		WithArgs("test-token").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateLastSeen(context.Background(), "test-token")
	assert.NoError(t, err)
}

func TestSaveToken_Error(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	token := PushToken{
		UserID:   1,
		Token:    "test-token",
		Platform: "ios",
		DeviceID: "device-123",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO push_tokens (user_id, token, platform, device_id, last_seen_at)
        VALUES ($1, $2, $3, $4, NOW())
        ON CONFLICT (token) DO UPDATE 
        SET user_id = $1, platform = $3, device_id = $4, last_seen_at = NOW(), updated_at = NOW()
        RETURNING id`)).
		WithArgs(token.UserID, token.Token, token.Platform, token.DeviceID).
		WillReturnError(errors.New("database error"))

	id, err := repo.SaveToken(context.Background(), token)
	assert.Error(t, err)
	assert.Equal(t, 0, id)
}
