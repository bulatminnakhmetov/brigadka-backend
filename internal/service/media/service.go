package media

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
)

// Определение ошибок
var (
	ErrMediaNotFound   = errors.New("media not found")
	ErrInvalidFileType = errors.New("invalid file type")
	ErrFileTooBig      = errors.New("file too big")
)

type Video struct {
	Id           int    `json:"id"`
	Url          string `json:"url"`
	ThumbnailUrl string `json:"thumbnail_url"`
}

type Image struct {
	Id  int    `json:"id"`
	Url string `json:"url"`
}

type ProfileMedia struct {
	Avatar Image   `json:"avatar"`
	Videos []Video `json:"videos"`
}

// Константы для ограничений
const (
	MaxFileSize = 10 * 1024 * 1024 // 10 MB
)

// Repository defines the interface for media database operations
type MediaRepository interface {
	CreateMedia(userID int, mediaType, mediaURL string) (int, error)
	DeleteMedia(userID, mediaID int) error
}

// StorageProvider определяет интерфейс для загрузки и получения файлов
type StorageProvider interface {
	UploadFile(file multipart.File, fileName string, contentType string) (string, error)
	DeleteFile(fileName string) error
	GetFileURL(fileName string) string
}

// MediaServiceImpl представляет реализацию сервиса медиа
type MediaServiceImpl struct {
	mediaRepository MediaRepository
	storageProvider StorageProvider
	allowedTypes    map[string]bool // Разрешенные расширения
}

// NewMediaService создает новый экземпляр MediaServiceImpl
func NewMediaService(mediaRepo MediaRepository, storageProvider StorageProvider) *MediaServiceImpl {
	// Разрешенные типы файлов
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".mp4":  true,
	}

	return &MediaServiceImpl{
		mediaRepository: mediaRepo,
		storageProvider: storageProvider,
		allowedTypes:    allowedTypes,
	}
}

type FileHeaderWrapper struct {
	*multipart.FileHeader
}

func (w *FileHeaderWrapper) Open() (multipart.File, error) {
	return w.FileHeader.Open()
}

func (w *FileHeaderWrapper) GetFilename() string {
	return w.Filename
}

func (w *FileHeaderWrapper) GetSize() int64 {
	return w.Size
}

func (w *FileHeaderWrapper) GetHeader() textproto.MIMEHeader {
	return w.Header
}

type UploadedFile interface {
	Open() (multipart.File, error)
	GetFilename() string
	GetSize() int64
	GetHeader() textproto.MIMEHeader
}

// UploadMedia загружает новый медиафайл
func (s *MediaServiceImpl) UploadMedia(userID int, fileHeader UploadedFile) (*int, error) {
	// Проверяем размер файла
	if fileHeader.GetSize() > MaxFileSize {
		return nil, ErrFileTooBig
	}

	// Открываем файл
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.GetFilename()))
	if _, allowed := s.allowedTypes[ext]; !allowed {
		return nil, ErrInvalidFileType
	}

	// Определяем тип медиа по расширению
	var mediaType string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mediaType = "image"
	case ".mp4", ".webm":
		mediaType = "video"
	default:
		return nil, ErrInvalidFileType
	}

	// Загружаем файл в хранилище
	mediaURL, err := s.storageProvider.UploadFile(file, fileHeader.GetFilename(), fileHeader.GetHeader().Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Сохраняем информацию о медиа в БД
	mediaID, err := s.mediaRepository.CreateMedia(userID, mediaType, mediaURL)
	if err != nil {
		return nil, err
	}

	return &mediaID, nil
}
