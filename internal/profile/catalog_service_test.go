package profile

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetActivityTypes(t *testing.T) {
	// Создаем mock для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		// Настраиваем mock для запроса
		rows := sqlmock.NewRows([]string{"activity_type", "description"}).
			AddRow("improv", "Комедийная импровизация").
			AddRow("music", "Музыкальное исполнение")

		mock.ExpectQuery("SELECT activity_type, description FROM activity_type_catalog").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetActivityTypes("ru")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 2)
		assert.Equal(t, "improv", catalog[0].Code)
		assert.Equal(t, "Комедийная импровизация", catalog[0].Description)
		assert.Equal(t, "music", catalog[1].Code)
		assert.Equal(t, "Музыкальное исполнение", catalog[1].Description)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Empty result", func(t *testing.T) {
		// Настраиваем mock для запроса с пустым результатом
		rows := sqlmock.NewRows([]string{"activity_type", "description"})

		mock.ExpectQuery("SELECT activity_type, description FROM activity_type_catalog").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetActivityTypes("ru")

		// Проверяем результаты - должен быть пустой массив без ошибки
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 0)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Database error", func(t *testing.T) {
		// Настраиваем mock для возврата ошибки
		mock.ExpectQuery("SELECT activity_type, description FROM activity_type_catalog").
			WillReturnError(sql.ErrConnDone)

		// Вызываем тестируемый метод
		catalog, err := service.GetActivityTypes("ru")

		// Проверяем результаты - должна быть ошибка и nil каталог
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.Nil(t, catalog)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetImprovStyles(t *testing.T) {
	// Создаем mock для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case - Russian language", func(t *testing.T) {
		// Настраиваем mock для запроса на русском языке
		rows := sqlmock.NewRows([]string{"style_code", "label", "description"}).
			AddRow("Short Form", "Короткая форма", "Короткие игры и зарисовки").
			AddRow("Long Form", "Длинная форма", "Продолжительные импровизации")

		mock.ExpectQuery("SELECT isc.style_code, ist.label, ist.description").
			WithArgs("ru").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetImprovStyles("ru")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 2)
		assert.Equal(t, "Short Form", catalog[0].Code)
		assert.Equal(t, "Короткая форма", catalog[0].Label)
		assert.Equal(t, "Короткие игры и зарисовки", catalog[0].Description)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Success case - English language", func(t *testing.T) {
		// Настраиваем mock для запроса на английском языке
		rows := sqlmock.NewRows([]string{"style_code", "label", "description"}).
			AddRow("Short Form", "Short Form", "Fast-paced, game-based improv").
			AddRow("Long Form", "Long Form", "Extended scenes and narratives")

		mock.ExpectQuery("SELECT isc.style_code, ist.label, ist.description").
			WithArgs("en").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetImprovStyles("en")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 2)
		assert.Equal(t, "Short Form", catalog[0].Code)
		assert.Equal(t, "Short Form", catalog[0].Label)
		assert.Equal(t, "Fast-paced, game-based improv", catalog[0].Description)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Database error", func(t *testing.T) {
		// Настраиваем mock для возврата ошибки
		mock.ExpectQuery("SELECT isc.style_code, ist.label, ist.description").
			WithArgs("ru").
			WillReturnError(sql.ErrConnDone)

		// Вызываем тестируемый метод
		catalog, err := service.GetImprovStyles("ru")

		// Проверяем результаты - должна быть ошибка и nil каталог
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)
		assert.Nil(t, catalog)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetImprovGoals(t *testing.T) {
	// Создаем mock для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		// Настраиваем mock для запроса
		rows := sqlmock.NewRows([]string{"goal_code", "label", "description"}).
			AddRow("Hobby", "Хобби", "Занятие импровом для удовольствия").
			AddRow("Career", "Карьера", "Импровизация как профессиональный путь").
			AddRow("Experiment", "Эксперимент", "Изучение импрова ради нового опыта")

		mock.ExpectQuery("SELECT igc.goal_code, igt.label, igt.description").
			WithArgs("ru").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetImprovGoals("ru")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 3)
		assert.Equal(t, "Hobby", catalog[0].Code)
		assert.Equal(t, "Хобби", catalog[0].Label)
		assert.Equal(t, "Career", catalog[1].Code)
		assert.Equal(t, "Карьера", catalog[1].Label)
		assert.Equal(t, "Experiment", catalog[2].Code)
		assert.Equal(t, "Эксперимент", catalog[2].Label)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Scan error", func(t *testing.T) {
		// Настраиваем mock с некорректными данными (ошибка сканирования)
		rows := sqlmock.NewRows([]string{"goal_code", "label"}). // Отсутствует столбец description
										AddRow("Hobby", "Хобби")

		mock.ExpectQuery("SELECT igc.goal_code, igt.label, igt.description").
			WithArgs("ru").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetImprovGoals("ru")

		// Проверяем результаты - должна быть ошибка и nil каталог
		assert.Error(t, err)
		assert.Nil(t, catalog)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetMusicGenres(t *testing.T) {
	// Создаем mock для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		// Настраиваем mock для запроса
		rows := sqlmock.NewRows([]string{"genre_code", "label"}).
			AddRow("rock", "Рок").
			AddRow("jazz", "Джаз").
			AddRow("classical", "Классика").
			AddRow("pop", "Поп-музыка").
			AddRow("electronic", "Электронная музыка")

		mock.ExpectQuery("SELECT mgc.genre_code, mgt.label").
			WithArgs("ru").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetMusicGenres("ru")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 5)
		assert.Equal(t, "rock", catalog[0].Code)
		assert.Equal(t, "Рок", catalog[0].Label)
		assert.Equal(t, "jazz", catalog[1].Code)
		assert.Equal(t, "Джаз", catalog[1].Label)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

func TestGetMusicInstruments(t *testing.T) {
	// Создаем mock для базы данных
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Создаем экземпляр сервиса с mock БД
	service := NewProfileService(db)

	t.Run("Success case", func(t *testing.T) {
		// Настраиваем mock для запроса
		rows := sqlmock.NewRows([]string{"instrument_code", "label"}).
			AddRow("acoustic_guitar", "Акустическая гитара").
			AddRow("electric_guitar", "Электрогитара").
			AddRow("bass_guitar", "Бас-гитара").
			AddRow("piano", "Фортепиано").
			AddRow("drums", "Ударные").
			AddRow("violin", "Скрипка").
			AddRow("voice", "Вокал")

		mock.ExpectQuery("SELECT mic.instrument_code, mit.label").
			WithArgs("ru").
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		catalog, err := service.GetMusicInstruments("ru")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 7)
		assert.Equal(t, "acoustic_guitar", catalog[0].Code)
		assert.Equal(t, "Акустическая гитара", catalog[0].Label)
		assert.Equal(t, "voice", catalog[6].Code)
		assert.Equal(t, "Вокал", catalog[6].Label)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("Empty result with default language", func(t *testing.T) {
		// Если язык не указан, должен использоваться язык по умолчанию (ru)
		// Настраиваем mock для запроса с пустой строкой языка
		rows := sqlmock.NewRows([]string{"instrument_code", "label"}).
			AddRow("acoustic_guitar", "Акустическая гитара")

		mock.ExpectQuery("SELECT mic.instrument_code, mit.label").
			WithArgs("ru"). // Должен быть заменен на ru в сервисе
			WillReturnRows(rows)

		// Вызываем тестируемый метод без указания языка
		catalog, err := service.GetMusicInstruments("")

		// Проверяем результаты
		assert.NoError(t, err)
		assert.NotNil(t, catalog)
		assert.Len(t, catalog, 1)

		// Проверяем, что все ожидаемые запросы были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
