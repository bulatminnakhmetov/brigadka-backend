package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// CatalogIntegrationTestSuite определяет набор интеграционных тестов для справочников
type CatalogIntegrationTestSuite struct {
	suite.Suite
	appUrl string
	token  string
}

// SetupSuite подготавливает общее окружение перед запуском всех тестов
func (s *CatalogIntegrationTestSuite) SetupSuite() {
	s.appUrl = os.Getenv("TEST_APP_URL")
	if s.appUrl == "" {
		s.appUrl = "http://localhost:8080" // Значение по умолчанию для локального тестирования
	}

	// Создаем пользователя и получаем токен для всех тестов
	profileSuite := ProfileIntegrationTestSuite{appUrl: s.appUrl}
	_, token, err := profileSuite.createTestUser()
	if err != nil {
		s.T().Fatalf("Failed to create test user: %v", err)
	}
	s.token = token
}

// TestCatalogEndpoints тестирует все эндпоинты каталогов
func (s *CatalogIntegrationTestSuite) TestCatalogEndpoints() {
	t := s.T()

	// Тестовые URL и ожидаемый минимальный размер ответа
	testCases := []struct {
		url           string
		minItemsCount int
		validCodes    []string // Примеры кодов, которые должны присутствовать
	}{
		{
			url:           "/api/profiles/catalog/activity-types",
			minItemsCount: 2,
			validCodes:    []string{"improv", "music"},
		},
		{
			url:           "/api/profiles/catalog/improv-styles",
			minItemsCount: 2,
			validCodes:    []string{"Short Form", "Long Form"},
		},
		{
			url:           "/api/profiles/catalog/improv-goals",
			minItemsCount: 3,
			validCodes:    []string{"Hobby", "Career", "Experiment"},
		},
		{
			url:           "/api/profiles/catalog/music-genres",
			minItemsCount: 4,
			validCodes:    []string{"rock", "jazz", "classical", "pop"},
		},
		{
			url:           "/api/profiles/catalog/music-instruments",
			minItemsCount: 5,
			validCodes:    []string{"acoustic_guitar", "piano", "drums", "voice"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.url, func(t *testing.T) {
			// Проверяем доступность каталога на русском языке (по умолчанию)
			s.checkCatalogLanguage(t, tc.url, "ru", tc.minItemsCount, tc.validCodes)

			// Проверяем доступность каталога на английском языке
			s.checkCatalogLanguage(t, tc.url, "en", tc.minItemsCount, tc.validCodes)
		})
	}
}

// TestActivityTypesCatalog подробно тестирует каталог типов активности
func (s *CatalogIntegrationTestSuite) TestActivityTypesCatalog() {
	t := s.T()
	url := "/api/profiles/catalog/activity-types"

	// Получаем каталог
	var catalog profile.ActivityTypeCatalog
	s.getCatalog(t, url, "ru", &catalog)

	// Проверяем наличие конкретных типов активности
	var foundImprov, foundMusic bool
	for _, item := range catalog {
		switch item.Code {
		case "improv":
			foundImprov = true
			assert.Contains(t, strings.ToLower(item.Description), "импровизация", "Improv description should contain 'импровизация' in Russian")
		case "music":
			foundMusic = true
			assert.Contains(t, strings.ToLower(item.Description), "музыка", "Music description should contain 'музыка' in Russian")
		}
	}

	assert.True(t, foundImprov, "Activity type 'improv' should be present")
	assert.True(t, foundMusic, "Activity type 'music' should be present")
}

// TestImprovStylesCatalog подробно тестирует каталог стилей импровизации
func (s *CatalogIntegrationTestSuite) TestImprovStylesCatalog() {
	t := s.T()
	url := "/api/profiles/catalog/improv-styles"

	// Получаем каталог на русском
	var catalogRu profile.ImprovStyleCatalog
	s.getCatalog(t, url, "ru", &catalogRu)

	// Проверяем наличие и перевод стилей
	var foundShortForm, foundLongForm bool
	for _, item := range catalogRu {
		switch item.Code {
		case "Short Form":
			foundShortForm = true
			assert.Contains(t, item.Label, "форма", "Short Form should have proper Russian translation")
		case "Long Form":
			foundLongForm = true
			assert.Contains(t, item.Label, "форма", "Long Form should have proper Russian translation")
		}
	}

	assert.True(t, foundShortForm, "Style 'Short Form' should be present")
	assert.True(t, foundLongForm, "Style 'Long Form' should be present")

	// Получаем каталог на английском
	var catalogEn profile.ImprovStyleCatalog
	s.getCatalog(t, url, "en", &catalogEn)

	// Проверяем соответствие переводов
	for _, itemRu := range catalogRu {
		for _, itemEn := range catalogEn {
			if itemRu.Code == itemEn.Code {
				assert.NotEqual(t, itemRu.Label, itemEn.Label,
					"Russian and English labels should be different for code: %s", itemRu.Code)
			}
		}
	}
}

// TestImprovGoalsCatalog подробно тестирует каталог целей импровизации
func (s *CatalogIntegrationTestSuite) TestImprovGoalsCatalog() {
	t := s.T()
	url := "/api/profiles/catalog/improv-goals"

	// Получаем каталог
	var catalog profile.ImprovGoalCatalog
	s.getCatalog(t, url, "ru", &catalog)

	// Проверяем наличие конкретных целей
	expectedGoals := []string{"Hobby", "Career", "Experiment"}
	for _, goal := range expectedGoals {
		found := false
		for _, item := range catalog {
			if item.Code == goal {
				found = true
				assert.NotEmpty(t, item.Label, "Goal %s should have a non-empty label", goal)
				assert.NotEmpty(t, item.Description, "Goal %s should have a non-empty description", goal)
				break
			}
		}
		assert.True(t, found, "Goal %s should be present in catalog", goal)
	}

	// Проверяем различие переводов
	var catalogEn profile.ImprovGoalCatalog
	s.getCatalog(t, url, "en", &catalogEn)

	for _, itemRu := range catalog {
		for _, itemEn := range catalogEn {
			if itemRu.Code == itemEn.Code {
				assert.NotEqual(t, itemRu.Label, itemEn.Label,
					"Russian and English labels should be different for goal: %s", itemRu.Code)
				assert.NotEqual(t, itemRu.Description, itemEn.Description,
					"Russian and English descriptions should be different for goal: %s", itemRu.Code)
			}
		}
	}
}

// TestMusicGenresCatalog подробно тестирует каталог музыкальных жанров
func (s *CatalogIntegrationTestSuite) TestMusicGenresCatalog() {
	t := s.T()
	url := "/api/profiles/catalog/music-genres"

	// Получаем каталог
	var catalog profile.MusicGenreCatalog
	s.getCatalog(t, url, "ru", &catalog)

	// Проверяем наличие конкретных жанров
	expectedGenres := []string{"rock", "jazz", "classical", "pop", "electronic"}
	for _, genre := range expectedGenres {
		found := false
		for _, item := range catalog {
			if item.Code == genre {
				found = true
				assert.NotEmpty(t, item.Label, "Genre %s should have a non-empty label", genre)
				break
			}
		}
		assert.True(t, found, "Genre %s should be present in catalog", genre)
	}
}

// TestMusicInstrumentsCatalog подробно тестирует каталог музыкальных инструментов
func (s *CatalogIntegrationTestSuite) TestMusicInstrumentsCatalog() {
	t := s.T()
	url := "/api/profiles/catalog/music-instruments"

	// Получаем каталог
	var catalog profile.MusicInstrumentCatalog
	s.getCatalog(t, url, "ru", &catalog)

	// Проверяем наличие популярных инструментов
	expectedInstruments := []string{
		"acoustic_guitar", "electric_guitar", "bass_guitar",
		"piano", "synthesizer", "drums", "violin", "voice",
	}

	for _, instrument := range expectedInstruments {
		found := false
		for _, item := range catalog {
			if item.Code == instrument {
				found = true
				assert.NotEmpty(t, item.Label, "Instrument %s should have a non-empty label", instrument)
				break
			}
		}
		assert.True(t, found, "Instrument %s should be present in catalog", instrument)
	}

	// Проверяем размер каталога
	assert.GreaterOrEqual(t, len(catalog), 10, "Instrument catalog should have at least 10 items")
}

// TestCatalogErrors тестирует ошибки при запросе каталогов
func (s *CatalogIntegrationTestSuite) TestCatalogErrors() {
	t := s.T()

	// Тестовые случаи для ошибок
	testCases := []struct {
		name           string
		url            string
		method         string
		authHeader     bool
		expectedStatus int
	}{
		{
			name:           "Unauthorized request",
			url:            "/api/profiles/catalog/activity-types",
			method:         "GET",
			authHeader:     false,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Method not allowed",
			url:            "/api/profiles/catalog/activity-types",
			method:         "POST",
			authHeader:     true,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Non-existent catalog",
			url:            "/api/profiles/catalog/non-existent",
			method:         "GET",
			authHeader:     true,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, s.appUrl+tc.url, nil)
			if tc.authHeader {
				req.Header.Set("Authorization", "Bearer "+s.token)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode,
				"Expected status %d for %s request to %s",
				tc.expectedStatus, tc.method, tc.url)
		})
	}
}

// Вспомогательные методы

// checkCatalogLanguage проверяет доступность каталога на указанном языке
func (s *CatalogIntegrationTestSuite) checkCatalogLanguage(t *testing.T, url, lang string, minItemsCount int, validCodes []string) {
	fullUrl := fmt.Sprintf("%s%s?lang=%s", s.appUrl, url, lang)
	req, _ := http.NewRequest("GET", fullUrl, nil)
	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Request to %s should not fail", fullUrl)
	defer resp.Body.Close()

	// Проверяем статус ответа
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Request to %s should return 200 OK", fullUrl)

	// Читаем и проверяем тело ответа
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Should be able to read response body")

	// Декодируем JSON
	var items []profile.TranslatedItem
	err = json.Unmarshal(body, &items)
	assert.NoError(t, err, "Response should be valid JSON")

	// Проверяем количество элементов
	assert.GreaterOrEqual(t, len(items), minItemsCount,
		"Catalog should have at least %d items", minItemsCount)

	// Проверяем наличие ожидаемых кодов
	foundCodes := make(map[string]bool)
	for _, code := range validCodes {
		foundCodes[code] = false
	}

	for _, item := range items {
		if _, exists := foundCodes[item.Code]; exists {
			foundCodes[item.Code] = true
			assert.NotEmpty(t, item.Label, "Item with code %s should have non-empty label", item.Code)
		}
	}

	// Проверяем, что все ожидаемые коды найдены
	for code, found := range foundCodes {
		assert.True(t, found, "Catalog should contain item with code %s", code)
	}
}

// getCatalog получает каталог с указанного URL
func (s *CatalogIntegrationTestSuite) getCatalog(t *testing.T, url, lang string, target interface{}) {
	fullUrl := fmt.Sprintf("%s%s?lang=%s", s.appUrl, url, lang)
	req, _ := http.NewRequest("GET", fullUrl, nil)
	req.Header.Set("Authorization", "Bearer "+s.token)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err, "Request to %s should not fail", fullUrl)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Request to %s should return 200 OK", fullUrl)

	err = json.NewDecoder(resp.Body).Decode(target)
	assert.NoError(t, err, "Should be able to decode response body to target type")
}

// TestCatalogIntegration запускает набор интеграционных тестов для каталогов
func TestCatalogIntegration(t *testing.T) {
	// Пропускаем тесты, если задана переменная окружения SKIP_INTEGRATION_TESTS
	if os.Getenv("SKIP_INTEGRATION_TESTS") != "" {
		t.Skip("Skipping integration tests")
	}

	suite.Run(t, new(CatalogIntegrationTestSuite))
}
