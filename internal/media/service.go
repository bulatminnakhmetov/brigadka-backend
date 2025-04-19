package media

import (
	"database/sql"
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

// Константы для ограничений
const (
	MaxFileSize = 10 * 1024 * 1024 // 10 MB
)

// MediaServiceImpl представляет реализацию сервиса медиа
type MediaServiceImpl struct {
	db              *sql.DB
	storageProvider StorageProvider
	allowedTypes    map[string]bool // Разрешенные расширения файлов
}

// NewMediaService создает новый экземпляр MediaServiceImpl
func NewMediaService(db *sql.DB, storageProvider StorageProvider) *MediaServiceImpl {
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
		db:              db,
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
func (s *MediaServiceImpl) UploadMedia(profileID int, mediaRole string, fileHeader UploadedFile) (*Media, error) {
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
		oldMedia, err := s.GetMediaByProfile(profileID, mediaRole)
		if err == nil && len(oldMedia) > 0 {
			// Удаляем старое медиа из БД
			_, err = s.db.Exec("DELETE FROM media WHERE profile_id = $1 AND role = $2", profileID, mediaRole)
			if err != nil {
				return nil, fmt.Errorf("failed to delete old media: %w", err)
			}

			// Не удаляем файл из хранилища, так как это может привести к ошибкам, если файл используется где-то еще
			// В будущем можно добавить периодическую очистку неиспользуемых файлов
		}
	}

	// Сохраняем информацию о медиа в БД
	var mediaID int
	err = s.db.QueryRow(
		"INSERT INTO media (profile_id, type, role, url) VALUES ($1, $2, $3, $4) RETURNING id",
		profileID, mediaType, mediaRole, mediaURL,
	).Scan(&mediaID)

	if err != nil {
		return nil, fmt.Errorf("failed to save media info: %w", err)
	}

	// Получаем созданную запись
	return s.GetMedia(mediaID)
}

// GetMedia получает медиа по ID
func (s *MediaServiceImpl) GetMedia(mediaID int) (*Media, error) {
	var media Media

	err := s.db.QueryRow(
		"SELECT id, profile_id, type, role, url, uploaded_at FROM media WHERE id = $1",
		mediaID,
	).Scan(
		&media.ID,
		&media.ProfileID,
		&media.Type,
		&media.Role,
		&media.URL,
		&media.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrMediaNotFound
		}
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	return &media, nil
}

// GetMediaByProfile получает все медиа для профиля
func (s *MediaServiceImpl) GetMediaByProfile(profileID int, mediaRole string) ([]Media, error) {
	var query string
	var args []interface{}

	if mediaRole == "" {
		// Если роль не указана, возвращаем все медиа для профиля
		query = "SELECT id, profile_id, type, role, url, uploaded_at FROM media WHERE profile_id = $1"
		args = []interface{}{profileID}
	} else {
		// Если роль указана, фильтруем по ней
		query = "SELECT id, profile_id, type, role, url, uploaded_at FROM media WHERE profile_id = $1 AND role = $2"
		args = []interface{}{profileID, mediaRole}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}
	defer rows.Close()

	var mediaList []Media
	for rows.Next() {
		var media Media
		err := rows.Scan(
			&media.ID,
			&media.ProfileID,
			&media.Type,
			&media.Role,
			&media.URL,
			&media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media: %w", err)
		}
		mediaList = append(mediaList, media)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media rows: %w", err)
	}

	return mediaList, nil
}

// DeleteMedia удаляет медиа
func (s *MediaServiceImpl) DeleteMedia(mediaID int) error {
	// Удаляем запись из БД
	_, err := s.db.Exec("DELETE FROM media WHERE id = $1", mediaID)
	if err != nil {
		return fmt.Errorf("failed to delete media from DB: %w", err)
	}

	// Имя файла в хранилище обычно последняя часть URL
	// Но может потребоваться более сложная логика в зависимости от формирования URL
	// Файл не удаляем из хранилища, только из БД

	return nil
}
