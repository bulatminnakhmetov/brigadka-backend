package auth

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetUserByEmail(t *testing.T) {
	// Создаем mock базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр репозитория с mock БД
	repo := NewPostgresUserRepository(db)

	t.Run("Success case", func(t *testing.T) {
		// Тестовые данные
		email := "test@example.com"
		expectedUser := &User{
			ID:           1,
			Email:        email,
			PasswordHash: "hashed_password",
			FullName:     "Test User",
			Gender:       "male",
			Age:          30,
			CityID:       1,
		}

		// Настраиваем mock
		rows := sqlmock.NewRows([]string{"user_id", "email", "password_hash", "full_name", "gender", "age", "city_id"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.PasswordHash, expectedUser.FullName, expectedUser.Gender, expectedUser.Age, expectedUser.CityID)

		mock.ExpectQuery("SELECT id, email, password_hash, full_name, gender, age, city_id FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		user, err := repo.GetUserByEmail(email)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, expectedUser.ID, user.ID)
		assert.Equal(t, expectedUser.Email, user.Email)
		assert.Equal(t, expectedUser.PasswordHash, user.PasswordHash)
		assert.Equal(t, expectedUser.FullName, user.FullName)
		assert.Equal(t, expectedUser.Gender, user.Gender)
		assert.Equal(t, expectedUser.Age, user.Age)
		assert.Equal(t, expectedUser.CityID, user.CityID)

		// Проверяем, что все ожидаемые запросы были выполнены
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User not found", func(t *testing.T) {
		email := "nonexistent@example.com"

		mock.ExpectQuery("SELECT id, email, password_hash, full_name, gender, age, city_id FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnError(sql.ErrNoRows)

		user, err := repo.GetUserByEmail(email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "user not found", err.Error())
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		email := "test@example.com"

		mock.ExpectQuery("SELECT id, email, password_hash, full_name, gender, age, city_id FROM users WHERE email = \\$1").
			WithArgs(email).
			WillReturnError(sql.ErrConnDone)

		user, err := repo.GetUserByEmail(email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCreateUser(t *testing.T) {
	// Создаем mock базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр репозитория с mock БД
	repo := NewPostgresUserRepository(db)

	t.Run("Success case", func(t *testing.T) {
		// Тестовые данные
		user := &User{
			Email:        "new@example.com",
			PasswordHash: "hashed_password",
			FullName:     "New User",
			Gender:       "female",
			Age:          25,
			CityID:       2,
		}

		// Настраиваем mock
		mock.ExpectQuery("INSERT INTO users \\(email, password_hash, full_name, gender, age, city_id\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\) RETURNING id").
			WithArgs(user.Email, user.PasswordHash, user.FullName, user.Gender, user.Age, user.CityID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(1))

		// Вызываем тестируемый метод
		err := repo.CreateUser(user)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.Equal(t, 1, user.ID) // ID должен быть установлен после создания
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		user := &User{
			Email:        "new@example.com",
			PasswordHash: "hashed_password",
			FullName:     "New User",
			Gender:       "female",
			Age:          25,
			CityID:       2,
		}

		mock.ExpectQuery("INSERT INTO users \\(email, password_hash, full_name, gender, age, city_id\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\) RETURNING id").
			WithArgs(user.Email, user.PasswordHash, user.FullName, user.Gender, user.Age, user.CityID).
			WillReturnError(sql.ErrConnDone)

		err := repo.CreateUser(user)

		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetUserByID(t *testing.T) {
	// Создаем моки для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	repo := NewPostgresUserRepository(db)

	t.Run("successful user retrieval", func(t *testing.T) {
		// Определяем ожидаемые данные
		expectedUser := &User{
			ID:           1,
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			FullName:     "Test User",
			Gender:       "male",
			Age:          30,
			CityID:       100,
		}

		// Настраиваем мок для имитации успешного запроса
		rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "full_name", "gender", "age", "city_id"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.PasswordHash, expectedUser.FullName, expectedUser.Gender, expectedUser.Age, expectedUser.CityID)

		mock.ExpectQuery("^SELECT (.+) FROM users WHERE id = \\$1$").
			WithArgs(1).
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		user, err := repo.GetUserByID(1)

		// Проверяем результат
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)

		// Проверяем, что все ожидания были выполнены
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		// Настраиваем мок для имитации отсутствия пользователя
		mock.ExpectQuery("^SELECT (.+) FROM users WHERE id = \\$1$").
			WithArgs(999).
			WillReturnError(sql.ErrNoRows)

		// Вызываем тестируемый метод
		user, err := repo.GetUserByID(999)

		// Проверяем результат
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "user not found", err.Error())

		// Проверяем, что все ожидания были выполнены
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		// Настраиваем мок для имитации ошибки базы данных
		dbError := errors.New("database connection failed")
		mock.ExpectQuery("^SELECT (.+) FROM users WHERE id = \\$1$").
			WithArgs(1).
			WillReturnError(dbError)

		// Вызываем тестируемый метод
		user, err := repo.GetUserByID(1)

		// Проверяем результат
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, dbError, err)

		// Проверяем, что все ожидания были выполнены
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}
