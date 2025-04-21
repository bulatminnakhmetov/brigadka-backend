package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/bulatminnakhmetov/brigadka-backend/docs" // Импорт сгенерированной документации
	"github.com/bulatminnakhmetov/brigadka-backend/internal/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/database"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/media" // Новый импорт
	"github.com/bulatminnakhmetov/brigadka-backend/internal/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/search" // Добавляем импорт пакета search
)

// @title           Brigadka API
// @version         1.0
// @description     API для сервиса Brigadka
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@brigadka.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// HealthResponse представляет ответ от health endpoint
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// Объявление startTime в глобальной области видимости
var startTime time.Time

// Инициализация времени запуска при загрузке пакета
func init() {
	startTime = time.Now()
}

// @Summary      Проверка здоровья сервиса
// @Description  Возвращает статус сервиса
// @Tags         health
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Failure      503  {object}  HealthResponse
// @Router       /health [get]
func healthHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, appVersion string) {
	// Проверка соединения с базой данных
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		response := HealthResponse{
			Status:    "error",
			Version:   appVersion,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Если соединение с БД в порядке, возвращаем статус OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := HealthResponse{
		Status:    "healthy",
		Version:   appVersion,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

func main() {
	_ = godotenv.Load()
	// Загрузка конфигурации из переменных окружения
	dbConfig := &database.Config{
		Host:     getEnv("DB_HOST", ""),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", ""),
		Password: getEnv("DB_PASSWORD", ""),
		DBName:   getEnv("DB_NAME", ""),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	jwtSecret := getEnv("JWT_SECRET", "")
	serverPort := getEnv("SERVER_PORT", "8080")
	appVersion := getEnv("APP_VERSION", "dev")

	// Подключение к базе данных
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Инициализация репозитория и хендлера авторизации
	userRepo := auth.NewPostgresUserRepository(db)
	authHandler := auth.NewAuthHandler(userRepo, jwtSecret)

	// Инициализация сервиса и хендлера профилей
	profileService := profile.NewProfileService(db)
	profileHandler := profile.NewProfileHandler(profileService)

	// Инициализация S3-совместимого хранилища для Backblaze B2
	s3Storage, err := media.NewS3StorageProvider(
		getEnv("B2_ACCESS_KEY_ID", ""),
		getEnv("B2_SECRET_ACCESS_KEY", ""),
		getEnv("B2_ENDPOINT", ""), // Выберите нужный регион
		getEnv("B2_BUCKET_NAME", ""),
		getEnv("CLOUDFLARE_CDN_DOMAIN", ""),
		"media", // Путь для загрузки в бакете
	)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Инициализация сервиса медиа
	mediaService := media.NewMediaService(db, s3Storage)

	// Инициализация хендлера медиа
	mediaHandler := media.NewMediaHandler(mediaService)

	// Инициализация сервиса и хендлера поиска
	searchService := search.NewSearchService(db)
	searchHandler := search.NewSearchHandler(searchService)

	// Создание роутера
	r := chi.NewRouter()

	// Базовые middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Подключение Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // URL для доступа к API документации
	))

	// Health endpoint для проверки работоспособности сервиса
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		healthHandler(w, r, db, appVersion)
	})

	// Расширенный health check с дополнительной информацией
	r.Get("/health/details", func(w http.ResponseWriter, r *http.Request) {
		details := map[string]interface{}{
			"status":      "healthy",
			"version":     appVersion,
			"timestamp":   time.Now().Format(time.RFC3339),
			"environment": getEnv("APP_ENV", "development"),
			"services": map[string]interface{}{
				"database": map[string]interface{}{
					"status": "connected",
					"host":   dbConfig.Host,
					"name":   dbConfig.DBName,
				},
			},
			"uptime": time.Since(startTime).String(),
		}

		// Проверка соединения с базой данных
		if err := db.Ping(); err != nil {
			details["status"] = "error"
			details["services"].(map[string]interface{})["database"].(map[string]interface{})["status"] = "error"
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
	})

	// Публичные маршруты аутентификации
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)
		r.Get("/verify", authHandler.Verify)
	})

	// Защищенные маршруты (требуют аутентификации)
	r.Group(func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware)

		r.Get("/api/protected", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value("user_id").(int)
			email := r.Context().Value("email").(string)
			w.Write([]byte(fmt.Sprintf("Protected resource. User ID: %d, Email: %s", userID, email)))
		})

		// Маршруты для работы с профилями (требуют аутентификации)
		r.Route("/api/profiles", func(r chi.Router) {
			r.Post("/", profileHandler.CreateProfile)
			r.Get("/{id}", profileHandler.GetProfile)

			// Регистрация обработчиков для справочников
			r.Route("/catalog", func(r chi.Router) {
				r.Get("/activity-types", profileHandler.GetActivityTypes)
				r.Get("/improv-styles", profileHandler.GetImprovStyles)
				r.Get("/improv-goals", profileHandler.GetImprovGoals)
				r.Get("/music-genres", profileHandler.GetMusicGenres)
				r.Get("/music-instruments", profileHandler.GetMusicInstruments)
			})

			// Новый маршрут для получения медиа профиля
			r.Get("/{id}/media", mediaHandler.GetMediaByProfile)
		})

		// Маршруты для работы с поиском (требуют аутентификации)
		r.Route("/api/search", func(r chi.Router) {
			r.Get("/profiles", searchHandler.SearchProfilesGet)
			r.Post("/profiles", searchHandler.SearchProfiles)
		})

		// Маршруты для работы с медиа (требуют аутентификации)
		r.Route("/api/media", func(r chi.Router) {
			r.Post("/upload", mediaHandler.UploadMedia)
			r.Get("/{id}", mediaHandler.GetMedia)
			r.Delete("/{id}", mediaHandler.DeleteMedia)
		})
	})

	// Запуск сервера с корректной обработкой graceful shutdown
	server := &http.Server{
		Addr:    ":" + serverPort,
		Handler: r,
	}

	// Запуск сервера в горутине
	go func() {
		log.Printf("Server is starting on port %s", serverPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on port %s: %v\n", serverPort, err)
		}
	}()

	// Канал для обработки сигналов завершения
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Ожидание сигнала
	<-stop

	// Корректное завершение работы сервера
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server gracefully stopped")
}

// Вспомогательные функции для работы с переменными окружения
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback == "" {
		panic(fmt.Sprintf("Environment variable %s is not set and no fallback provided", key))
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return fallback
}
