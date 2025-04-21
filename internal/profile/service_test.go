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
		styles := []string{"Short Form", "Harold"}
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
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "created_at"}).
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
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "created_at"}).
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

func TestGetProfile(t *testing.T) {
	// Создаем mock базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Get improv profile", func(t *testing.T) {
		profileID := 1
		userID := 1
		description := "Test Description"
		activityType := ActivityTypeImprov
		now := time.Now()
		goal := "Hobby"
		styles := []string{"Short Form", "Harold"}
		lookingForTeam := true

		// Ожидаем запрос на получение базового профиля
		mock.ExpectQuery("SELECT profile_id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "user_id", "description", "activity_type", "created_at"}).
				AddRow(profileID, userID, description, activityType, now))

		// Expect query to get improv goal and looking_for_team flag
		mock.ExpectQuery("SELECT goal, looking_for_team").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"goal", "looking_for_team"}).AddRow(goal, lookingForTeam))

		// Ожидаем запрос на получение стилей импровизации
		mock.ExpectQuery("SELECT style").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"style"}).
				AddRow(styles[0]).
				AddRow(styles[1]))

		// Вызываем тестируемый метод
		profileResp, err := service.GetProfile(profileID)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, profileResp)
		assert.NotNil(t, profileResp.ImprovProfile)
		assert.Nil(t, profileResp.MusicProfile)

		// Check improv profile details
		assert.Equal(t, profileID, profileResp.ImprovProfile.ProfileID)
		assert.Equal(t, userID, profileResp.ImprovProfile.UserID)
		assert.Equal(t, description, profileResp.ImprovProfile.Description)
		assert.Equal(t, activityType, profileResp.ImprovProfile.ActivityType)
		assert.Equal(t, goal, profileResp.ImprovProfile.Goal)
		assert.ElementsMatch(t, styles, profileResp.ImprovProfile.Styles)
		assert.Equal(t, lookingForTeam, profileResp.ImprovProfile.LookingForTeam)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Get music profile", func(t *testing.T) {
		profileID := 2
		userID := 2
		description := "Music Profile"
		activityType := ActivityTypeMusic
		now := time.Now()
		genres := []string{"rock", "jazz"}
		instruments := []string{"guitar", "piano"}

		// Ожидаем запрос на получение базового профиля
		mock.ExpectQuery("SELECT profile_id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"profile_id", "user_id", "description", "activity_type", "created_at"}).
				AddRow(profileID, userID, description, activityType, now))

		// Ожидаем запрос на получение жанров музыки
		mock.ExpectQuery("SELECT genre_code").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"genre_code"}).
				AddRow(genres[0]).
				AddRow(genres[1]))

		// Ожидаем запрос на получение инструментов
		mock.ExpectQuery("SELECT instrument_code").
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"instrument_code"}).
				AddRow(instruments[0]).
				AddRow(instruments[1]))

		// Вызываем тестируемый метод
		profileResp, err := service.GetProfile(profileID)

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, profileResp)
		assert.NotNil(t, profileResp.MusicProfile)
		assert.Nil(t, profileResp.ImprovProfile)

		// Check music profile details
		assert.Equal(t, profileID, profileResp.MusicProfile.ProfileID)
		assert.Equal(t, userID, profileResp.MusicProfile.UserID)
		assert.Equal(t, description, profileResp.MusicProfile.Description)
		assert.Equal(t, activityType, profileResp.MusicProfile.ActivityType)
		assert.ElementsMatch(t, genres, profileResp.MusicProfile.Genres)
		assert.ElementsMatch(t, instruments, profileResp.MusicProfile.Instruments)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Profile not found", func(t *testing.T) {
		profileID := 999

		// Ожидаем запрос на получение базового профиля с ошибкой
		mock.ExpectQuery("SELECT profile_id, user_id, description, activity_type, created_at").
			WithArgs(profileID).
			WillReturnError(sql.ErrNoRows)

		// Вызываем тестируемый метод
		profileResp, err := service.GetProfile(profileID)

		// Проверяем результаты
		assert.Error(t, err)
		assert.Equal(t, ErrProfileNotFound, err)
		assert.Nil(t, profileResp)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
