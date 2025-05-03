package media

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"

	mediarepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/media"
	storageMedia "github.com/bulatminnakhmetov/brigadka-backend/internal/storage/media"
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

type ProfileMedia struct {
	Avatar string  `json:"avatar"`
	Videos []Video `json:"videos"`
}

// Константы для ограничений
const (
	MaxFileSize = 10 * 1024 * 1024 // 10 MB
)

// Repository defines the interface for media database operations
type MediaRepository interface {
	CreateMedia(profileID int, mediaType, mediaRole, mediaURL string) (int, error)
	GetMediaByProfileAndRole(profileID int, mediaRole string) ([]mediarepo.Media, error)
	DeleteMediaByID(mediaID int) error
	DeleteMediaByProfileAndRole(profileID int, mediaRole string) error
}

// MediaServiceImpl представляет реализацию сервиса медиа
type MediaServiceImpl struct {
	mediaRepository MediaRepository
	storageProvider storageMedia.StorageProvider
	allowedTypes    map[string]bool // Разрешенные расширения файлов
}

// NewMediaService создает новый экземпляр MediaServiceImpl
func NewMediaService(mediaRepo MediaRepository, storageProvider storageMedia.StorageProvider) *MediaServiceImpl {
	// Разрешенные типы файлов
	allowedTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
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
func (s *MediaServiceImpl) UploadMedia(profileID int, mediaRole string, fileHeader UploadedFile) (*int, error) {
	// Проверяем размер файла
	if fileHeader.GetSize() > MaxFileSize {
		return nil, ErrFileTooBig
	}

	// Проверяем расширение файла
	ext := strings.ToLower(filepath.Ext(fileHeader.GetFilename()))
	if _, allowed := s.allowedTypes[ext]; !allowed {
		return nil, ErrInvalidFileType
	}

	// Открываем файл
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Определяем тип медиа по расширению
	var mediaType string
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		mediaType = "image"
	case ".mp4", ".webm":
		mediaType = "video"
	default:
		mediaType = "other"
	}

	// Загружаем файл в хранилище
	mediaURL, err := s.storageProvider.UploadFile(file, fileHeader.GetFilename(), fileHeader.GetHeader().Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// Если у профиля уже есть медиа с такой ролью, удаляем старое
	// например, у профиля может быть только один аватар
	if mediaRole == "avatar" || mediaRole == "cover" {
		oldMedia, err := s.mediaRepository.GetMediaByProfileAndRole(profileID, mediaRole)
		if err == nil && len(oldMedia) > 0 {
			// Удаляем старое медиа из БД
			err = s.mediaRepository.DeleteMediaByProfileAndRole(profileID, mediaRole)
			if err != nil {
				return nil, fmt.Errorf("failed to delete old media: %w", err)
			}

			// Не удаляем файл из хранилища, так как это может привести к ошибкам, если файл используется где-то еще
			// В будущем можно добавить периодическую очистку неиспользуемых файлов
		}
	}

	// Сохраняем информацию о медиа в БД
	mediaID, err := s.mediaRepository.CreateMedia(profileID, mediaType, mediaRole, mediaURL)
	if err != nil {
		return nil, err
	}

	return &mediaID, nil
}

// DeleteMedia удаляет медиа
func (s *MediaServiceImpl) DeleteMedia(mediaID int) error {
	return s.mediaRepository.DeleteMediaByID(mediaID)
}

func (s *MediaServiceImpl) GetProfileMedia(profileID int) (*ProfileMedia, error) {
	// Получаем медиа для профиля
	mediaList, err := s.mediaRepository.GetMediaByProfileAndRole(profileID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to get profile media: %w", err)
	}

	profileMedia := &ProfileMedia{}

	for _, media := range mediaList {
		switch media.Role {
		case "avatar":
			profileMedia.Avatar = media.URL
		case "video":
			profileMedia.Videos = append(profileMedia.Videos, Video{Url: media.URL})
		}
	}

	return profileMedia, nil
}
