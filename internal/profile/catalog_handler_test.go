package profile

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тесты для всех хендлеров справочников
func TestGetActivityTypesHandler(t *testing.T) {
	// Основной успешный тест
	t.Run("Success - default language", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := ActivityTypeCatalog{
			{Code: "improv", Label: "improv", Description: "Комедийная импровизация"},
			{Code: "music", Label: "music", Description: "Музыкальное исполнение"},
		}

		// Настраиваем поведение мока - без указания языка должен использоваться ru
		mockService.On("GetActivityTypes", "").Return(testCatalog, nil)

		// Создаем тестовый запрос без параметра lang
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/activity-types", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetActivityTypes(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response ActivityTypeCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})

	// Тест с указанием языка
	t.Run("Success - specific language", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные на английском
		testCatalog := ActivityTypeCatalog{
			{Code: "improv", Label: "improv", Description: "Comedy improvisation"},
			{Code: "music", Label: "music", Description: "Musical performance"},
		}

		// Настраиваем поведение мока для английского языка
		mockService.On("GetActivityTypes", "en").Return(testCatalog, nil)

		// Создаем тестовый запрос с параметром lang=en
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/activity-types?lang=en", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetActivityTypes(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response ActivityTypeCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})

	// Тест с ошибкой от сервиса
	t.Run("Service error", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Настраиваем поведение мока для возврата ошибки
		expectedError := errors.New("database error")
		mockService.On("GetActivityTypes", "ru").Return(nil, expectedError)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/activity-types?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetActivityTypes(rr, req)

		// Проверяем статус ответа - должна быть ошибка сервера
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		// Проверяем сообщение об ошибке
		assert.Contains(t, rr.Body.String(), "Internal server error")

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})

	// Тест с неправильным методом запроса
	t.Run("Method not allowed", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовый POST запрос вместо GET
		req, _ := http.NewRequest("POST", "/api/profiles/catalog/activity-types", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetActivityTypes(rr, req)

		// Проверяем статус ответа - должен быть Method Not Allowed
		assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)

		// Проверяем сообщение об ошибке
		assert.Contains(t, rr.Body.String(), "Method not allowed")

		// Проверяем, что мок НЕ был вызван
		mockService.AssertNotCalled(t, "GetActivityTypes")
	})

	// Тест с пустым результатом от сервиса
	t.Run("Empty catalog", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Настраиваем поведение мока для возврата пустого каталога
		emptyCatalog := ActivityTypeCatalog{}
		mockService.On("GetActivityTypes", "ru").Return(emptyCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/activity-types?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetActivityTypes(rr, req)

		// Проверяем статус ответа - должен быть успешным
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа - должен быть пустой массив
		assert.Equal(t, "[]\n", rr.Body.String())

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})
}

func TestGetImprovStylesHandler(t *testing.T) {
	// Основной успешный тест
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := ImprovStyleCatalog{
			{Code: "Short Form", Label: "Короткая форма", Description: "Короткие игры и зарисовки"},
			{Code: "Long Form", Label: "Длинная форма", Description: "Продолжительные импровизации"},
		}

		// Настраиваем поведение мока
		mockService.On("GetImprovStyles", "ru").Return(testCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/improv-styles?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetImprovStyles(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response ImprovStyleCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)
		assert.Len(t, response, 2)
		assert.Equal(t, "Short Form", response[0].Code)
		assert.Equal(t, "Короткая форма", response[0].Label)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})

	// Тест с ошибкой
	t.Run("Service error", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Настраиваем поведение мока для возврата ошибки
		expectedError := errors.New("database error")
		mockService.On("GetImprovStyles", "ru").Return(nil, expectedError)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/improv-styles?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetImprovStyles(rr, req)

		// Проверяем статус ответа - должна быть ошибка сервера
		assert.Equal(t, http.StatusInternalServerError, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})
}

func TestGetImprovGoalsHandler(t *testing.T) {
	// Основной успешный тест
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := ImprovGoalCatalog{
			{Code: "Hobby", Label: "Хобби", Description: "Занятие импровом для удовольствия"},
			{Code: "Career", Label: "Карьера", Description: "Импровизация как профессиональный путь"},
		}

		// Настраиваем поведение мока
		mockService.On("GetImprovGoals", "ru").Return(testCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/improv-goals?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetImprovGoals(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response ImprovGoalCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})
}

func TestGetMusicGenresHandler(t *testing.T) {
	// Основной успешный тест
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := MusicGenreCatalog{
			{Code: "rock", Label: "Рок"},
			{Code: "jazz", Label: "Джаз"},
			{Code: "classical", Label: "Классика"},
		}

		// Настраиваем поведение мока
		mockService.On("GetMusicGenres", "ru").Return(testCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/music-genres?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetMusicGenres(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response MusicGenreCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)
		assert.Len(t, response, 3)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})
}

func TestGetMusicInstrumentsHandler(t *testing.T) {
	// Основной успешный тест
	t.Run("Success", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := MusicInstrumentCatalog{
			{Code: "acoustic_guitar", Label: "Акустическая гитара"},
			{Code: "piano", Label: "Фортепиано"},
			{Code: "drums", Label: "Ударные"},
		}

		// Настраиваем поведение мока
		mockService.On("GetMusicInstruments", "ru").Return(testCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/music-instruments?lang=ru", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetMusicInstruments(rr, req)

		// Проверяем статус ответа
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем содержимое ответа
		var response MusicInstrumentCatalog
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, testCatalog, response)
		assert.Len(t, response, 3)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})

	// Тест для неподдерживаемого языка (должен вернуть данные на языке по умолчанию)
	t.Run("Unsupported language", func(t *testing.T) {
		mockService := new(MockProfileService)
		handler := NewProfileHandler(mockService)

		// Создаем тестовые данные
		testCatalog := MusicInstrumentCatalog{
			{Code: "acoustic_guitar", Label: "Акустическая гитара"},
		}

		// Ожидаем, что unsupported_lang будет заменен на дефолтный язык в сервисе
		mockService.On("GetMusicInstruments", "unsupported_lang").Return(testCatalog, nil)

		// Создаем тестовый запрос
		req, _ := http.NewRequest("GET", "/api/profiles/catalog/music-instruments?lang=unsupported_lang", nil)
		rr := httptest.NewRecorder()

		// Вызываем обработчик
		handler.GetMusicInstruments(rr, req)

		// Проверяем статус ответа - должен быть успешный, даже для неподдерживаемого языка
		assert.Equal(t, http.StatusOK, rr.Code)

		// Проверяем, что мок был вызван с ожидаемыми аргументами
		mockService.AssertExpectations(t)
	})
}
