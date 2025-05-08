package media

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient определяет интерфейс для работы с S3-совместимым хранилищем
type MinioClient interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (info minio.UploadInfo, err error)
	RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error
}

// S3StorageProvider представляет провайдер хранилища для S3-совместимых сервисов (включая Backblaze B2)
type S3StorageProvider struct {
	client         MinioClient // Заменили *minio.Client на интерфейс
	bucketName     string
	cdnDomain      string // Домен Cloudflare CDN
	endpoint       string // S3-совместимый эндпоинт
	uploadPath     string // Путь для загрузки в бакете
	contentType    map[string]string
	publicEndpoint string // Публичный эндпоинт для Android-эмуляторов
}

// NewS3StorageProvider создает новый экземпляр S3StorageProvider для работы с Backblaze B2
func NewS3StorageProvider(accessKeyID, secretAccessKey, endpoint, bucketName, cdnDomain, uploadPath string, publicEndpoint string) (*S3StorageProvider, error) {
	// Инициализируем клиент MinIO для работы с S3-совместимым API
	// Для Backblaze B2 используем endpoint: s3.us-west-004.backblazeb2.com (или другой регион)
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: true, // Используем HTTPS
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	// Проверяем, существует ли бакет
	exists, err := client.BucketExists(context.Background(), bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket '%s' does not exist", bucketName)
	}

	// Карта известных MIME-типов
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mp3":  "audio/mpeg",
	}

	return &S3StorageProvider{
		client:         client,
		bucketName:     bucketName,
		cdnDomain:      cdnDomain,
		endpoint:       endpoint,
		uploadPath:     uploadPath,
		contentType:    contentTypes,
		publicEndpoint: publicEndpoint,
	}, nil
}

// UploadFile загружает файл в хранилище
func (s *S3StorageProvider) UploadFile(file multipart.File, fileName string) (string, error) {
	ctx := context.Background()

	// Генерируем уникальное имя файла, используя UUID
	extension := filepath.Ext(fileName)
	uniqueFileName := fmt.Sprintf("%s/%s%s", s.uploadPath, uuid.New().String(), extension)

	// Определяем тип контента
	contentType := ""
	if knownType, ok := s.contentType[extension]; ok {
		contentType = knownType
	} else {
		contentType = "application/octet-stream"
	}

	// Опции для загрузки файла
	options := minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "public, max-age=31536000", // 1 год кэширования
	}

	// Загружаем файл в бакет
	_, err := s.client.PutObject(ctx, s.bucketName, uniqueFileName, file, -1, options)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// Возвращаем URL через Cloudflare CDN
	return s.GetFileURL(uniqueFileName), nil
}

// DeleteFile удаляет файл из хранилища
func (s *S3StorageProvider) DeleteFile(fileName string) error {
	ctx := context.Background()

	// Удаляем объект из бакета
	err := s.client.RemoveObject(ctx, s.bucketName, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetFileURL возвращает URL для доступа к файлу через Cloudflare CDN
func (s *S3StorageProvider) GetFileURL(fileName string) string {
	// Если указан CDN домен, используем его
	if s.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", s.cdnDomain, fileName)
	}

	// Если указан публичный эндпоинт (для Android-эмуляторов), используем его
	if s.publicEndpoint != "" {
		return fmt.Sprintf("https://%s/%s/%s", s.publicEndpoint, s.bucketName, fileName)
	}

	// Иначе используем прямую ссылку на S3-совместимое хранилище
	// Обратите внимание: для Backblaze B2 URL может иметь другой формат
	return fmt.Sprintf("https://%s/%s/%s", s.endpoint, s.bucketName, fileName)
}
