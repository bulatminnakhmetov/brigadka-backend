package media

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *RepositoryImpl) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewRepository(db)
	return db, mock, repo
}

func TestCreateMedia(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	mediaType := "image"
	mediaURL := "https://example.com/image.jpg"
	thumbnailURL := "https://example.com/thumbnail.jpg"
	expectedID := 42

	rows := sqlmock.NewRows([]string{"id"}).AddRow(expectedID)
	mock.ExpectQuery("INSERT INTO media").
		WithArgs(userID, mediaType, mediaURL, thumbnailURL).
		WillReturnRows(rows)

	mediaID, err := repo.CreateMedia(userID, mediaType, mediaURL, thumbnailURL)
	assert.NoError(t, err)
	assert.Equal(t, expectedID, mediaID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateMediaError(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	mediaType := "image"
	mediaURL := "https://example.com/image.jpg"
	thumbnailURL := "https://example.com/thumbnail.jpg"

	mock.ExpectQuery("INSERT INTO media").
		WithArgs(userID, mediaType, mediaURL, thumbnailURL).
		WillReturnError(errors.New("database error"))

	mediaID, err := repo.CreateMedia(userID, mediaType, mediaURL, thumbnailURL)
	assert.Error(t, err)
	assert.Equal(t, 0, mediaID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMedia(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	mediaID := 42

	mock.ExpectExec("DELETE FROM media").
		WithArgs(mediaID, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteMedia(userID, mediaID)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMediaError(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	userID := 1
	mediaID := 42

	mock.ExpectExec("DELETE FROM media").
		WithArgs(mediaID, userID).
		WillReturnError(errors.New("database error"))

	err := repo.DeleteMedia(userID, mediaID)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByID(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	mediaID := 42
	now := time.Now()
	expectedMedia := Media{
		ID:           mediaID,
		UserID:       1,
		Role:         "image",
		URL:          "https://example.com/image.jpg",
		ThumbnailURL: "https://example.com/thumbnail.jpg",
		UploadedAt:   now,
	}

	rows := sqlmock.NewRows([]string{"id", "owner_id", "type", "url", "thumbnail_url", "uploaded_at"}).
		AddRow(expectedMedia.ID, expectedMedia.UserID, expectedMedia.Role, expectedMedia.URL, expectedMedia.ThumbnailURL, expectedMedia.UploadedAt)

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(mediaID).
		WillReturnRows(rows)

	media, err := repo.GetMediaByID(mediaID)
	assert.NoError(t, err)
	assert.Equal(t, expectedMedia, *media)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByIDNotFound(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	mediaID := 42

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(mediaID).
		WillReturnError(sql.ErrNoRows)

	media, err := repo.GetMediaByID(mediaID)
	assert.Error(t, err)
	assert.Equal(t, ErrMediaNotFound, err)
	assert.Nil(t, media)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByIDError(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	mediaID := 42

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(mediaID).
		WillReturnError(errors.New("database error"))

	media, err := repo.GetMediaByID(mediaID)
	assert.Error(t, err)
	assert.Nil(t, media)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByIDs(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	now := time.Now()
	mediaIDs := []int{1, 2}

	expectedMedia1 := Media{
		ID:           1,
		UserID:       1,
		Role:         "image",
		URL:          "https://example.com/image1.jpg",
		ThumbnailURL: "https://example.com/thumbnail1.jpg",
		UploadedAt:   now,
	}

	expectedMedia2 := Media{
		ID:           2,
		UserID:       1,
		Role:         "image",
		URL:          "https://example.com/image2.jpg",
		ThumbnailURL: "https://example.com/thumbnail2.jpg",
		UploadedAt:   now,
	}

	// For first media
	rows1 := sqlmock.NewRows([]string{"id", "owner_id", "type", "url", "thumbnail_url", "uploaded_at"}).
		AddRow(expectedMedia1.ID, expectedMedia1.UserID, expectedMedia1.Role, expectedMedia1.URL, expectedMedia1.ThumbnailURL, expectedMedia1.UploadedAt)

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(1).
		WillReturnRows(rows1)

	// For second media
	rows2 := sqlmock.NewRows([]string{"id", "owner_id", "type", "url", "thumbnail_url", "uploaded_at"}).
		AddRow(expectedMedia2.ID, expectedMedia2.UserID, expectedMedia2.Role, expectedMedia2.URL, expectedMedia2.ThumbnailURL, expectedMedia2.UploadedAt)

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(2).
		WillReturnRows(rows2)

	media, err := repo.GetMediaByIDs(mediaIDs)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(media))
	assert.Equal(t, expectedMedia1, media[0])
	assert.Equal(t, expectedMedia2, media[1])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByIDsEmpty(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	mediaIDs := []int{}

	media, err := repo.GetMediaByIDs(mediaIDs)
	assert.NoError(t, err)
	assert.Nil(t, media)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMediaByIDsWithError(t *testing.T) {
	db, mock, repo := setupMock(t)
	defer db.Close()

	mediaIDs := []int{1, 2}

	// First media found
	now := time.Now()
	expectedMedia1 := Media{
		ID:           1,
		UserID:       1,
		Role:         "image",
		URL:          "https://example.com/image1.jpg",
		ThumbnailURL: "https://example.com/thumbnail1.jpg",
		UploadedAt:   now,
	}

	rows1 := sqlmock.NewRows([]string{"id", "owner_id", "type", "url", "thumbnail_url", "uploaded_at"}).
		AddRow(expectedMedia1.ID, expectedMedia1.UserID, expectedMedia1.Role, expectedMedia1.URL, expectedMedia1.ThumbnailURL, expectedMedia1.UploadedAt)

	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(1).
		WillReturnRows(rows1)

	// Second media returns error
	mock.ExpectQuery("SELECT id, owner_id, type, url, thumbnail_url, uploaded_at FROM media").
		WithArgs(2).
		WillReturnError(sql.ErrNoRows)

	media, err := repo.GetMediaByIDs(mediaIDs)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(media))
	assert.Equal(t, expectedMedia1, media[0])
	assert.NoError(t, mock.ExpectationsWereMet())
}
