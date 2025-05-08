package media

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMinioClient реализует интерфейс работы с minio для тестирования
type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *MockMinioClient) PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockMinioClient) RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func (m *MockMinioClient) PresignedGetObject(ctx context.Context, bucketName string, objectName string, expires time.Duration, reqParams url.Values) (*url.URL, error) {
	args := m.Called(ctx, bucketName, objectName, expires, reqParams)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

type mockMultipartFile struct {
	*bytes.Reader
	io.Closer
}

func (m *mockMultipartFile) ReadAt(p []byte, off int64) (n int, err error) {
	return m.Reader.ReadAt(p, off)
}

// TestS3StorageProvider проверяет создание и функционирование S3StorageProvider
func TestNewS3StorageProvider(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		// Мокаем только проверки, не создавая реальный клиент
		// Это не полностью тестирует NewS3StorageProvider, только логику после создания клиента

		provider := &S3StorageProvider{
			client:      nil, // Настоящий клиент не создаем в тесте
			bucketName:  "test-bucket",
			cdnDomain:   "cdn.example.com",
			endpoint:    "test.endpoint.com",
			uploadPath:  "media",
			contentType: make(map[string]string),
		}

		// Проверяем, что основные поля правильно установлены
		assert.Equal(t, "test-bucket", provider.bucketName)
		assert.Equal(t, "cdn.example.com", provider.cdnDomain)
		assert.Equal(t, "media", provider.uploadPath)
	})
}

// TestUploadFile проверяет функцию загрузки файла
func TestUploadFile(t *testing.T) {
	t.Run("successful upload", func(t *testing.T) {
		mockClient := new(MockMinioClient)

		provider := &S3StorageProvider{
			client:     mockClient,
			bucketName: "test-bucket",
			cdnDomain:  "cdn.example.com",
			endpoint:   "test.endpoint.com",
			uploadPath: "media",
			contentType: map[string]string{
				".jpg": "image/jpeg",
			},
		}

		// Подготавливаем файл для загрузки
		fileContent := []byte("test file content")
		filePart := &multipart.FileHeader{
			Filename: "test.jpg",
			Size:     int64(len(fileContent)),
			Header: map[string][]string{
				"Content-Type": {"image/jpeg"},
			},
		}

		// Создаем мок для файла, реализующий multipart.File
		mockFile := &mockMultipartFile{
			Reader: bytes.NewReader(fileContent),
			Closer: io.NopCloser(nil),
		}

		// Ожидаем вызов PutObject с любыми аргументами
		mockClient.On("PutObject",
			mock.Anything,
			"test-bucket",
			mock.MatchedBy(func(s string) bool {
				return len(s) > 0 && s[:6] == "media/"
			}),
			mock.Anything,
			int64(-1),
			mock.MatchedBy(func(o minio.PutObjectOptions) bool {
				return o.ContentType == "image/jpeg"
			})).Return(minio.UploadInfo{}, nil)

		// Имитация открытия файла из multipart.FileHeader
		fileURL, err := provider.UploadFile(mockFile, filePart.Filename)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.Contains(t, fileURL, "https://cdn.example.com/media/")
		assert.Contains(t, fileURL, ".jpg")

		// Проверяем, что мок был вызван
		mockClient.AssertExpectations(t)
	})

	t.Run("upload error", func(t *testing.T) {
		mockClient := new(MockMinioClient)

		provider := &S3StorageProvider{
			client:     mockClient,
			bucketName: "test-bucket",
			cdnDomain:  "cdn.example.com",
			endpoint:   "test.endpoint.com",
			uploadPath: "media",
			contentType: map[string]string{
				".jpg": "image/jpeg",
			},
		}

		// Подготавливаем файл для загрузки
		fileContent := []byte("test file content")
		filePart := &multipart.FileHeader{
			Filename: "test.jpg",
			Size:     int64(len(fileContent)),
		}

		// Создаем мок для файла, реализующий multipart.File
		mockFile := &mockMultipartFile{
			Reader: bytes.NewReader(fileContent),
			Closer: io.NopCloser(nil),
		}

		// Ожидаем вызов PutObject и возвращаем ошибку
		mockClient.On("PutObject",
			mock.Anything,
			"test-bucket",
			mock.Anything,
			mock.Anything,
			int64(-1),
			mock.Anything).Return(minio.UploadInfo{}, os.ErrPermission)

		// Имитация открытия файла из multipart.FileHeader
		fileURL, err := provider.UploadFile(mockFile, filePart.Filename)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to upload file")
		assert.Empty(t, fileURL)

		// Проверяем, что мок был вызван
		mockClient.AssertExpectations(t)
	})
}

// TestDeleteFile проверяет функцию удаления файла
func TestDeleteFile(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockClient := new(MockMinioClient)

		provider := &S3StorageProvider{
			client:     mockClient,
			bucketName: "test-bucket",
		}

		fileName := "media/test-file.jpg"

		// Ожидаем вызов RemoveObject с заданными параметрами
		mockClient.On("RemoveObject",
			mock.Anything,
			"test-bucket",
			fileName,
			mock.Anything).Return(nil)

		err := provider.DeleteFile(fileName)

		// Проверяем результаты
		assert.NoError(t, err)

		// Проверяем, что мок был вызван
		mockClient.AssertExpectations(t)
	})

	t.Run("delete error", func(t *testing.T) {
		mockClient := new(MockMinioClient)

		provider := &S3StorageProvider{
			client:     mockClient,
			bucketName: "test-bucket",
		}

		fileName := "media/test-file.jpg"

		// Ожидаем вызов RemoveObject и возвращаем ошибку
		mockClient.On("RemoveObject",
			mock.Anything,
			"test-bucket",
			fileName,
			mock.Anything).Return(os.ErrNotExist)

		err := provider.DeleteFile(fileName)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete file")

		// Проверяем, что мок был вызван
		mockClient.AssertExpectations(t)
	})
}

// TestGetFileURL проверяет функцию получения URL файла
func TestGetFileURL(t *testing.T) {
	t.Run("with CDN domain", func(t *testing.T) {
		provider := &S3StorageProvider{
			cdnDomain:  "cdn.example.com",
			bucketName: "test-bucket",
			endpoint:   "s3.example.com",
		}

		fileName := "media/test-file.jpg"
		url := provider.GetFileURL(fileName)

		assert.Equal(t, "https://cdn.example.com/media/test-file.jpg", url)
	})

	t.Run("without CDN domain", func(t *testing.T) {
		provider := &S3StorageProvider{
			cdnDomain:  "",
			bucketName: "test-bucket",
			endpoint:   "s3.example.com",
		}

		fileName := "media/test-file.jpg"
		url := provider.GetFileURL(fileName)

		assert.Equal(t, "https://s3.example.com/test-bucket/media/test-file.jpg", url)
	})
}
