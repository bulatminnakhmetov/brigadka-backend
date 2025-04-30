package profile

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateImprovProfile(t *testing.T) {
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
		goal := "Hobby"
		styles := []string{"Short Form", "Long Form"}
		lookingForTeam := true
		now := time.Now()

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на проверку существования цели импровизации
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запросы на проверку существования стилей импровизации
		for _, style := range styles {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(style).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Ожидаем начало транзакции
		mock.ExpectBegin()

		// Ожидаем запрос на создание базового профиля
		mock.ExpectQuery("INSERT INTO profiles").
			WithArgs(userID, description, ActivityTypeImprov).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, now))

		// Ожидаем запрос на создание профиля импровизации
		mock.ExpectExec("INSERT INTO improv_profiles").
			WithArgs(1, goal, lookingForTeam).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Ожидаем запросы на добавление стилей импровизации
		for _, style := range styles {
			mock.ExpectExec("INSERT INTO improv_profile_styles").
				WithArgs(1, style).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Ожидаем завершение транзакции
		mock.ExpectCommit()

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, 1, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, ActivityTypeImprov, profile.ActivityType)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("User not found", func(t *testing.T) {
		userID := 999
		description := "Test description"
		goal := "Hobby"
		styles := []string{"Short Form"}
		lookingForTeam := true

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile already exists", func(t *testing.T) {
		userID := 1
		description := "Test description"
		goal := "Hobby"
		styles := []string{"Short Form"}
		lookingForTeam := true

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrProfileAlreadyExists, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid improv goal", func(t *testing.T) {
		userID := 1
		description := "Test description"
		goal := "InvalidGoal"
		styles := []string{"Short Form"}
		lookingForTeam := true

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на проверку существования цели импровизации
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImprovGoal, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid improv style", func(t *testing.T) {
		userID := 1
		description := "Test description"
		goal := "Hobby"
		styles := []string{"Short Form", "InvalidStyle"}
		lookingForTeam := true

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на проверку существования цели импровизации
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Проверка первого стиля - успешно
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(styles[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Проверка второго стиля - не найден
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(styles[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImprovStyle, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Database error", func(t *testing.T) {
		userID := 1
		description := "Test description"
		goal := "Hobby"
		styles := []string{"Short Form"}
		lookingForTeam := true

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запрос на проверку существования цели импровизации
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запросы на проверку существования стилей импровизации
		for _, style := range styles {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(style).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Ожидаем начало транзакции
		mock.ExpectBegin()

		// Ожидаем запрос на создание базового профиля с ошибкой
		mock.ExpectQuery("INSERT INTO profiles").
			WithArgs(userID, description, ActivityTypeImprov).
			WillReturnError(sql.ErrConnDone)

		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Вызываем тестируемый метод
		profile, err := service.CreateImprovProfile(userID, description, goal, styles, lookingForTeam)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestCreateMusicProfile(t *testing.T) {
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
		genres := []string{"rock", "jazz"}
		instruments := []string{"guitar", "piano"}
		now := time.Now()

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Ожидаем запросы на проверку существования жанров
		for _, genre := range genres {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(genre).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Ожидаем запросы на проверку существования инструментов
		for _, instrument := range instruments {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(instrument).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Ожидаем начало транзакции
		mock.ExpectBegin()

		// Ожидаем запрос на создание базового профиля
		mock.ExpectQuery("INSERT INTO profiles").
			WithArgs(userID, description, ActivityTypeMusic).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).
				AddRow(1, now))

		// Ожидаем запросы на добавление жанров
		for _, genre := range genres {
			mock.ExpectExec("INSERT INTO music_profile_genres").
				WithArgs(1, genre).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Ожидаем запросы на добавление инструментов инструментов
		for _, instrument := range instruments {
			mock.ExpectExec("INSERT INTO music_profile_instruments").
				WithArgs(1, instrument).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Ожидаем завершение транзакции
		mock.ExpectCommit()

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, 1, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, ActivityTypeMusic, profile.ActivityType)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("User not found", func(t *testing.T) {
		userID := 999
		description := "Test description"
		genres := []string{"rock"}
		instruments := []string{"guitar"}

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile already exists", func(t *testing.T) {
		userID := 1
		description := "Test description"
		genres := []string{"rock"}
		instruments := []string{"guitar"}

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrProfileAlreadyExists, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Empty instruments list", func(t *testing.T) {
		userID := 1
		description := "Test description"
		genres := []string{"rock"}
		var instruments []string // пустой список

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrEmptyInstruments, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid music genre", func(t *testing.T) {
		userID := 1
		description := "Test description"
		genres := []string{"rock", "invalid_genre"}
		instruments := []string{"guitar"}

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Проверка первого жанра - успешно
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Проверка второго жанра - не найден
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidMusicGenre, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid instrument", func(t *testing.T) {
		userID := 1
		description := "Test description"
		genres := []string{"rock"}
		instruments := []string{"guitar", "invalid_instrument"}

		// Ожидаем запрос на проверку существования пользователя
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Ожидаем запрос на проверку существования профиля
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Проверка жанра
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Проверка первого инструмента - успешно
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(instruments[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Проверка второго инструмента - не найден
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(instruments[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Вызываем тестируемый метод
		profile, err := service.CreateMusicProfile(userID, description, genres, instruments)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInstrument, err)
		assert.Nil(t, profile)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetImprovProfile(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Create a service instance with the mock DB
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Test Improv Description"
		activityType := ActivityTypeImprov
		now := time.Now()
		goal := "Hobby"
		styles := []string{"Short Form", "Long Form"}
		lookingForTeam := true

		// Expect query to get base profile
		mock.ExpectQuery("SELECT id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "description", "activity_type", "created_at"}).
				AddRow(profileID, userID, description, activityType, now))

		// Expect query to get improv goal and looking_for_team flag
		mock.ExpectQuery("SELECT goal, looking_for_team").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"goal", "looking_for_team"}).AddRow(goal, lookingForTeam))

		// Expect query to get improv styles
		mock.ExpectQuery("SELECT style").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"style"}).
				AddRow(styles[0]).
				AddRow(styles[1]))

		// Call the method being tested
		profile, err := service.GetImprovProfile(profileID)

		// Check results
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, profileID, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, activityType, profile.ActivityType)
		assert.Equal(t, goal, profile.Goal)
		assert.ElementsMatch(t, styles, profile.Styles)
		assert.Equal(t, lookingForTeam, profile.LookingForTeam)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile not found", func(t *testing.T) {
		profileID := 999

		// Expect query to get base profile with error
		mock.ExpectQuery("SELECT id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Call the method being tested
		profile, err := service.GetImprovProfile(profileID)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetMusicProfile(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Create a service instance with the mock DB
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		profileID := 2
		userID := 2
		description := "Test Music Description"
		activityType := ActivityTypeMusic
		now := time.Now()
		genres := []string{"rock", "jazz"}
		instruments := []string{"guitar", "piano"}

		// Expect query to get base profile
		mock.ExpectQuery("SELECT id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "description", "activity_type", "created_at"}).
				AddRow(profileID, userID, description, activityType, now))

		// Expect query to get music genres
		mock.ExpectQuery("SELECT genre_code").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"genre_code"}).
				AddRow(genres[0]).
				AddRow(genres[1]))

		// Expect query to get music instruments
		mock.ExpectQuery("SELECT instrument_code").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"instrument_code"}).
				AddRow(instruments[0]).
				AddRow(instruments[1]))

		// Call the method being tested
		profile, err := service.GetMusicProfile(profileID)

		// Check results
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, profileID, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, activityType, profile.ActivityType)
		assert.ElementsMatch(t, genres, profile.Genres)
		assert.ElementsMatch(t, instruments, profile.Instruments)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
	t.Run("Profile not found", func(t *testing.T) {
		profileID := 999

		// Expect query to get base profile with error
		mock.ExpectQuery("SELECT id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Call the method being tested
		profile, err := service.GetMusicProfile(profileID)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestUpdateImprovProfile(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Create a service instance with the mock DB
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated description"
		goal := "Career"                              // Updated goal
		styles := []string{"Long Form", "Short Form"} // Updated styles
		lookingForTeam := false                       // Updated flag
		now := time.Now()

		// Expect query to check if profile exists and get user_id
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeImprov))

		// Expect query to check if goal exists
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect queries to check if styles exist
		for _, style := range styles {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(style).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Expect transaction to begin
		mock.ExpectBegin()

		// Expect query to update the base profile
		mock.ExpectExec("UPDATE profiles SET description").
			WithArgs(description, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect query to update the improv profile
		mock.ExpectExec("UPDATE improv_profiles SET goal").
			WithArgs(goal, lookingForTeam, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect query to delete old styles
		mock.ExpectExec("DELETE FROM improv_profile_styles").
			WithArgs(profileID).
			WillReturnResult(sqlmock.NewResult(0, 2)) // Assuming 2 old styles

		// Expect queries to add new styles
		for _, style := range styles {
			mock.ExpectExec("INSERT INTO improv_profile_styles").
				WithArgs(profileID, style).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Expect query to get created_at time
		mock.ExpectQuery("SELECT created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

		// Expect transaction to commit
		mock.ExpectCommit()

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, profileID, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, ActivityTypeImprov, profile.ActivityType)
		assert.Equal(t, goal, profile.Goal)
		assert.Equal(t, styles, profile.Styles)
		assert.Equal(t, lookingForTeam, profile.LookingForTeam)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile not found", func(t *testing.T) {
		profileID := 999
		description := "Updated description"
		goal := "Career"
		styles := []string{"Long Form"}
		lookingForTeam := false

		// Expect query to check if profile exists returning no rows
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Wrong activity type", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated description"
		goal := "Career"
		styles := []string{"Long Form"}
		lookingForTeam := false

		// Expect query to check if profile exists with wrong activity type
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeMusic)) // Wrong type

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidActivityType, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid improv goal", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated description"
		goal := "InvalidGoal"
		styles := []string{"Long Form"}
		lookingForTeam := false

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeImprov))

		// Expect query to check if goal exists returning false
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImprovGoal, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid improv style", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated description"
		goal := "Career"
		styles := []string{"Long Form", "InvalidStyle"}
		lookingForTeam := false

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeImprov))

		// Expect query to check if goal exists
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check first style (valid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(styles[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check second style (invalid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(styles[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidImprovStyle, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Database transaction error", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated description"
		goal := "Career"
		styles := []string{"Long Form"}
		lookingForTeam := false

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeImprov))

		// Expect query to check if goal exists
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(goal).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check style
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(styles[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect transaction to begin
		mock.ExpectBegin()

		// Expect query to update base profile with error
		mock.ExpectExec("UPDATE profiles SET description").
			WithArgs(description, profileID).
			WillReturnError(sql.ErrConnDone)

		// Expect transaction to rollback
		mock.ExpectRollback()

		// Call the method being tested
		profile, err := service.UpdateImprovProfile(profileID, description, goal, styles, lookingForTeam)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestUpdateMusicProfile(t *testing.T) {
	// Create a mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Create a service instance with the mock DB
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated music profile"
		genres := []string{"rock", "jazz", "blues"} // Updated genres
		instruments := []string{"guitar", "piano"}  // Updated instruments
		now := time.Now()

		// Expect query to check if profile exists and get user_id
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeMusic))

		// Expect queries to check if genres exist
		for _, genre := range genres {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(genre).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Expect queries to check if instruments exist
		for _, instrument := range instruments {
			mock.ExpectQuery("SELECT EXISTS").
				WithArgs(instrument).
				WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		}

		// Expect transaction to begin
		mock.ExpectBegin()

		// Expect query to update the base profile
		mock.ExpectExec("UPDATE profiles SET description").
			WithArgs(description, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect query to delete old genres
		mock.ExpectExec("DELETE FROM music_profile_genres").
			WithArgs(profileID).
			WillReturnResult(sqlmock.NewResult(0, 2)) // Assuming 2 old genres

		// Expect queries to add new genres
		for _, genre := range genres {
			mock.ExpectExec("INSERT INTO music_profile_genres").
				WithArgs(profileID, genre).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Expect query to delete old instruments
		mock.ExpectExec("DELETE FROM music_profile_instruments").
			WithArgs(profileID).
			WillReturnResult(sqlmock.NewResult(0, 1)) // Assuming 1 old instrument

		// Expect queries to add new instruments
		for _, instrument := range instruments {
			mock.ExpectExec("INSERT INTO music_profile_instruments").
				WithArgs(profileID, instrument).
				WillReturnResult(sqlmock.NewResult(0, 1))
		}

		// Expect query to get created_at time
		mock.ExpectQuery("SELECT created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(now))

		// Expect transaction to commit
		mock.ExpectCommit()

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.NoError(t, err)
		assert.NotNil(t, profile)
		assert.Equal(t, profileID, profile.ProfileID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, description, profile.Description)
		assert.Equal(t, ActivityTypeMusic, profile.ActivityType)
		assert.ElementsMatch(t, genres, profile.Genres)
		assert.ElementsMatch(t, instruments, profile.Instruments)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile not found", func(t *testing.T) {
		profileID := 999
		description := "Updated music profile"
		genres := []string{"rock"}
		instruments := []string{"guitar"}

		// Expect query to check if profile exists returning no rows
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Wrong activity type", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated music profile"
		genres := []string{"rock"}
		instruments := []string{"guitar"}

		// Expect query to check if profile exists with wrong activity type
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeImprov)) // Wrong type

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidActivityType, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Empty instruments list", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated music profile"
		genres := []string{"rock"}
		var instruments []string // Empty list

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeMusic))

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrEmptyInstruments, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid music genre", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated music profile"
		genres := []string{"rock", "invalid_genre"}
		instruments := []string{"guitar"}

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeMusic))

		// Expect query to check first genre (valid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check second genre (invalid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidMusicGenre, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Invalid instrument", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Updated music profile"
		genres := []string{"rock"}
		instruments := []string{"guitar", "invalid_instrument"}

		// Expect query to check if profile exists
		mock.ExpectQuery("SELECT user_id, activity_type").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"user_id", "activity_type"}).
				AddRow(userID, ActivityTypeMusic))

		// Expect queries to check if genres exist
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(genres[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check first instrument (valid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(instruments[0]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		// Expect query to check second instrument (invalid)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(instruments[1]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		// Call the method being tested
		profile, err := service.UpdateMusicProfile(profileID, description, genres, instruments)

		// Check results
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidInstrument, err)
		assert.Nil(t, profile)

		// Verify that all expected queries were executed
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
