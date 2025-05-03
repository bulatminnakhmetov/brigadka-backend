package media

import (
	"mime/multipart"
)

// StorageProvider определяет интерфейс для загрузки и получения файлов
type StorageProvider interface {
	// UploadFile загружает файл в хранилище и возвращает URL
	UploadFile(file multipart.File, fileName string, contentType string) (string, error)

	// DeleteFile удаляет файл из хранилища
	DeleteFile(fileName string) error

	// GetFileURL возвращает URL для доступа к файлу
	GetFileURL(fileName string) string
}
