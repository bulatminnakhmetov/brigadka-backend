package user

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *PostgresUserRepository) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	repo := NewPostgresUserRepository(db)
	return db, mock, repo
}

func TestGetUserByEmail_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash
        FROM users 
        WHERE email = $1
    `)).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash"}).
			AddRow(1, "test@example.com", "hashed_password"))

	user, err := repo.GetUserByEmail("test@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash
        FROM users 
        WHERE email = $1
    `)).
		WithArgs("notfound@example.com").
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByEmail("notfound@example.com")
	assert.Nil(t, user)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestCreateUser_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO users (email, password_hash)
        VALUES ($1, $2)
        RETURNING id
    `)).
		WithArgs("newuser@example.com", "hashed_password").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	user := &User{
		Email:        "newuser@example.com",
		PasswordHash: "hashed_password",
	}

	err := repo.CreateUser(user)
	assert.NoError(t, err)
	assert.Equal(t, 1, user.ID)
}

func TestCreateUser_Error(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        INSERT INTO users (email, password_hash)
        VALUES ($1, $2)
        RETURNING id
    `)).
		WithArgs("newuser@example.com", "hashed_password").
		WillReturnError(errors.New("database error"))

	user := &User{
		Email:        "newuser@example.com",
		PasswordHash: "hashed_password",
	}

	err := repo.CreateUser(user)
	assert.Error(t, err)
	assert.Equal(t, 0, user.ID)
}

func TestGetUserByID_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash
        FROM users 
        WHERE id = $1
    `)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash"}).
			AddRow(1, "test@example.com", "hashed_password"))

	user, err := repo.GetUserByID(1)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "hashed_password", user.PasswordHash)
}

func TestGetUserByID_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT id, email, password_hash
        FROM users 
        WHERE id = $1
    `)).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByID(999)
	assert.Nil(t, user)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestBeginTx_Success(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()

	tx, err := repo.BeginTx()
	assert.NoError(t, err)
	assert.NotNil(t, tx)
}
