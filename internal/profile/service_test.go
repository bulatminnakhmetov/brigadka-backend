package profile

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateProfile(t *testing.T) {
	// Создаем mock базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		userID := 1
		description := "Test description"
		activityType := "sports"
		now := time.Now()

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования типа активности
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(activityType).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на создание профиля
		mock.ExpectQuery("INSERT INTO profiles").
			WithArgs(userID, description, activityType).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "user_id", "description", "activity_type", "created_at"}).
				AddRow(1, userID, description, activityType, now))

		// Вызываем тестируемый метод
		profile, err := service.CreateProfile(userID, description, activityType)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, 1, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, activityType, profile.ActivityType)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("User not found", func(t *testing.T) {
		userID := 999
		description := "Test description"
		activityType := "sports"

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateProfile(userID, description, activityType)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid activity type", func(t *testing.T) {
		userID := 1
		description := "Test description"
		activityType := "invalid_type"

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования типа активности
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(activityType).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateProfile(userID, description, activityType)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidActivityType, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile already exists", func(t *testing.T) {
		userID := 1
		description := "Test description"
		activityType := "sports"

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования типа активности
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(activityType).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Вызываем тестируемый метод
		profile, err := service.CreateProfile(userID, description, activityType)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrProfileAlreadyExists, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := 1
		description := "Test description"
		activityType := "sports"

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования типа активности
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(activityType).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на создание профиля с ошибкой
		mock.ExpectQuery("INSERT INTO profiles").
			WithArgs(userID, description, activityType).
			WillReturnError(sql.ErrConnDone)

		// Вызываем тестируемый метод
		profile, err := service.CreateProfile(userID, description, activityType)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
