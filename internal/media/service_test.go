package media

import (
	"bytes"
	"database/sql"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockStorageProvider - мок для интерфейса StorageProvider
type MockStorageProvider struct {
	mock.Mock
}

func (m *MockStorageProvider) UploadFile(file multipart.File, fileName string, contentType string) (string, error) {
	args := m.Called(file, fileName, contentType)
	return args.String(0), args.Error(1)
}

func (m *MockStorageProvider) DeleteFile(fileName string) error {
	args := m.Called(fileName)
	return args.Error(0)
}

func (m *MockStorageProvider) GetFileURL(fileName string) string {
	args := m.Called(fileName)
	return args.String(0)
}

// MockUploadedFile implements UploadedFile interface for testing
type MockUploadedFile struct {
	filename   string
	size       int64
	header     textproto.MIMEHeader
	shouldFail bool
}

func NewMockUploadedFile(filename string, size int64, contentType string) *MockUploadedFile {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Type", contentType)
	return &MockUploadedFile{
		filename: filename,
		size:     size,
		header:   header,
	}
}

// multipartFileMock implements multipart.File for tests
type multipartFileMock struct {
	*bytes.Reader
}

func (f multipartFileMock) Read(p []byte) (int, error) {
	return 0, io.EOF
}
func (f multipartFileMock) Close() error {
	return nil
}
func (f multipartFileMock) ReadAt(p []byte, off int64) (int, error) {
	return 0, io.EOF
}
func (f multipartFileMock) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (m *MockUploadedFile) Open() (multipart.File, error) {
	if m.shouldFail {
		return nil, errors.New("failed to open file")
	}
	// Return a bytes.Reader wrapped as multipart.File (which implements io.Reader, io.ReaderAt, io.Seeker, io.Closer)
	return multipartFileMock{}, nil
}

func (m *MockUploadedFile) GetFilename() string {
	return m.filename
}

func (m *MockUploadedFile) GetSize() int64 {
	return m.size
}

func (m *MockUploadedFile) GetHeader() textproto.MIMEHeader {
	return m.header
}

// TestNewMediaService проверяет создание сервиса
func TestNewMediaService(t *testing.T) {
	// Создаем мок базы данных
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем мок StorageProvider
	mockStorage := new(MockStorageProvider)

	// Создаем экземпляр сервиса
	service := NewMediaService(db, mockStorage)

	// Проверяем, что сервис создан успешно
	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, mockStorage, service.storageProvider)
	assert.NotEmpty(t, service.allowedTypes)
	assert.True(t, service.allowedTypes[".jpg"])
}

// TestUploadMedia проверяет функцию загрузки медиа файла
func TestUploadMedia(t *testing.T) {
	t.Run("successful upload", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		// 1. Проверка существующих медиа с такой ролью
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE profile_id = (.+) AND role = (.+)").
			WithArgs(1, "avatar").
			WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "type", "role", "url", "uploaded_at"}))

		// 2. Вставка новой записи
		dbMock.ExpectQuery("INSERT INTO media (.+) VALUES (.+) RETURNING id").
			WithArgs(1, "image", "avatar", "https://cdn.example.com/media/12345.jpg").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		// 3. Получение созданной записи
		mediaCreatedAt := time.Now()
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE id = (.+)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).AddRow(1, 1, "image", "avatar", "https://cdn.example.com/media/12345.jpg", mediaCreatedAt))

		// Создаем тестовый файл
		fileHeader := NewMockUploadedFile("test.jpg", 5*1024, "image/jpeg")

		// Настраиваем ожидаемые вызовы хранилища
		mockStorage.On("UploadFile", mock.Anything, "test.jpg", "image/jpeg").
			Return("https://cdn.example.com/media/12345.jpg", nil)

		// Вызываем функцию загрузки
		media, err := service.UploadMedia(1, "avatar", fileHeader)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, media)
		assert.Equal(t, 1, media.ID)
		assert.Equal(t, 1, media.ProfileID)
		assert.Equal(t, "image", media.Type)
		assert.Equal(t, "avatar", media.Role)
		assert.Equal(t, "https://cdn.example.com/media/12345.jpg", media.URL)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
		mockStorage.AssertExpectations(t)
	})

	t.Run("upload with replacing existing media", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		// 1. Проверка существующих медиа с такой ролью - возвращаем одно существующее
		mediaCreatedAt := time.Now().Add(-24 * time.Hour)
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE profile_id = (.+) AND role = (.+)").
			WithArgs(1, "avatar").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).AddRow(5, 1, "image", "avatar", "https://cdn.example.com/media/old.jpg", mediaCreatedAt))

		// 2. Удаление старой записи
		dbMock.ExpectExec("DELETE FROM media WHERE profile_id = (.+) AND role = (.+)").
			WithArgs(1, "avatar").
			WillReturnResult(sqlmock.NewResult(0, 1))

		// 3. Вставка новой записи
		dbMock.ExpectQuery("INSERT INTO media (.+) VALUES (.+) RETURNING id").
			WithArgs(1, "image", "avatar", "https://cdn.example.com/media/12345.jpg").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(6))

		// 4. Получение созданной записи
		newMediaCreatedAt := time.Now()
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE id = (.+)").
			WithArgs(6).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).AddRow(6, 1, "image", "avatar", "https://cdn.example.com/media/12345.jpg", newMediaCreatedAt))

		// Создаем тестовый файл
		fileHeader := NewMockUploadedFile("test.jpg", 5*1024, "image/jpeg")

		// Настраиваем ожидаемые вызовы хранилища
		mockStorage.On("UploadFile", mock.Anything, "test.jpg", "image/jpeg").
			Return("https://cdn.example.com/media/12345.jpg", nil)

		// Вызываем функцию загрузки
		media, err := service.UploadMedia(1, "avatar", fileHeader)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, media)
		assert.Equal(t, 6, media.ID)
		assert.Equal(t, 1, media.ProfileID)
		assert.Equal(t, "https://cdn.example.com/media/12345.jpg", media.URL)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
		mockStorage.AssertExpectations(t)
	})

	t.Run("file too big", func(t *testing.T) {
		// Создаем мок базы данных
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Создаем тестовый файл, превышающий лимит
		fileHeader := NewMockUploadedFile("test.jpg", 15*1024*1024, "image/jpeg")

		// Вызываем функцию загрузки
		media, err := service.UploadMedia(1, "avatar", fileHeader)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrFileTooBig, err)
		assert.Nil(t, media)

		// Хранилище не должно вызываться
		mockStorage.AssertNotCalled(t, "UploadFile")
	})

	t.Run("invalid file type", func(t *testing.T) {
		// Создаем мок базы данных
		db, _, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Создаем тестовый файл неподдерживаемого типа
		fileHeader := NewMockUploadedFile("test.exe", 5*1024, "application/octet-stream")

		// Вызываем функцию загрузки
		media, err := service.UploadMedia(1, "avatar", fileHeader)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidFileType, err)
		assert.Nil(t, media)

		// Хранилище не должно вызываться
		mockStorage.AssertNotCalled(t, "UploadFile")
	})

	t.Run("storage error", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Создаем тестовый файл
		fileHeader := NewMockUploadedFile("test.jpg", 5*1024, "image/jpeg")

		// Настраиваем ожидаемые вызовы хранилища с возвратом ошибки
		expectedError := errors.New("storage upload failed")
		mockStorage.On("UploadFile", mock.Anything, "test.jpg", "image/jpeg").
			Return("", expectedError)

		// Вызываем функцию загрузки
		media, err := service.UploadMedia(1, "avatar", fileHeader)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upload file")
		assert.Nil(t, media)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
		mockStorage.AssertExpectations(t)
	})
}

// TestGetMedia проверяет функцию получения медиа по ID
func TestGetMedia(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		mediaCreatedAt := time.Now()
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE id = (.+)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).AddRow(1, 2, "image", "avatar", "https://cdn.example.com/media/12345.jpg", mediaCreatedAt))

		// Вызываем функцию получения
		media, err := service.GetMedia(1)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, media)
		assert.Equal(t, 1, media.ID)
		assert.Equal(t, 2, media.ProfileID)
		assert.Equal(t, "image", media.Type)
		assert.Equal(t, "avatar", media.Role)
		assert.Equal(t, "https://cdn.example.com/media/12345.jpg", media.URL)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE id = (.+)").
			WithArgs(1).
			WillReturnError(sql.ErrNoRows)

		// Вызываем функцию получения
		media, err := service.GetMedia(1)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrMediaNotFound, err)
		assert.Nil(t, media)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

// TestGetMediaByProfile проверяет функцию получения медиа по профилю
func TestGetMediaByProfile(t *testing.T) {
	t.Run("with specific role", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		mediaCreatedAt := time.Now()
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE profile_id = (.+) AND role = (.+)").
			WithArgs(1, "avatar").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).AddRow(1, 1, "image", "avatar", "https://cdn.example.com/media/avatar.jpg", mediaCreatedAt))

		// Вызываем функцию получения
		mediaList, err := service.GetMediaByProfile(1, "avatar")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, mediaList)
		assert.Len(t, mediaList, 1)
		assert.Equal(t, 1, mediaList[0].ID)
		assert.Equal(t, "avatar", mediaList[0].Role)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("all roles", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		mediaCreatedAt := time.Now()
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE profile_id = (.+)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}).
				AddRow(1, 1, "image", "avatar", "https://cdn.example.com/media/avatar.jpg", mediaCreatedAt).
				AddRow(2, 1, "image", "gallery", "https://cdn.example.com/media/gallery1.jpg", mediaCreatedAt).
				AddRow(3, 1, "image", "gallery", "https://cdn.example.com/media/gallery2.jpg", mediaCreatedAt))

		// Вызываем функцию получения
		mediaList, err := service.GetMediaByProfile(1, "")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, mediaList)
		assert.Len(t, mediaList, 3)
		assert.Equal(t, "avatar", mediaList[0].Role)
		assert.Equal(t, "gallery", mediaList[1].Role)
		assert.Equal(t, "gallery", mediaList[2].Role)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		dbMock.ExpectQuery("SELECT (.+) FROM media WHERE profile_id = (.+)").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "profile_id", "type", "role", "url", "uploaded_at",
			}))

		// Вызываем функцию получения
		mediaList, err := service.GetMediaByProfile(1, "")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.Empty(t, mediaList)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

// TestDeleteMedia проверяет функцию удаления медиа
func TestDeleteMedia(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы
		dbMock.ExpectExec("DELETE FROM media WHERE id = (.+)").
			WithArgs(1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Вызываем функцию удаления
		err = service.DeleteMedia(1)

		// Проверяем результаты
		assert.NoError(t, err)

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("db error", func(t *testing.T) {
		// Создаем мок базы данных
		db, dbMock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}
		defer db.Close()

		// Создаем мок StorageProvider
		mockStorage := new(MockStorageProvider)

		// Создаем сервис
		service := NewMediaService(db, mockStorage)

		// Настраиваем ожидаемые запросы с ошибкой
		dbError := errors.New("db error")
		dbMock.ExpectExec("DELETE FROM media WHERE id = (.+)").
			WithArgs(1).
			WillReturnError(dbError)

		// Вызываем функцию удаления
		err = service.DeleteMedia(1)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete media from DB")

		// Проверяем, что все мокированные вызовы были сделаны
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}
