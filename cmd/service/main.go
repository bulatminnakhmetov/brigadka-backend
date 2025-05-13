package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/database"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/media"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/messaging"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/handler/profile"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/logging"
	mediarepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/media"
	messagingrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/messaging"
	profilerepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/profile"
	userrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/user"

	authservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/auth"
	mediaservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/media"
	messagingservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/messaging"
	profileservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/profile"

	mediastorage "github.com/bulatminnakhmetov/brigadka-backend/internal/storage/media"

	pushhandler "github.com/bulatminnakhmetov/brigadka-backend/internal/handler/push"
	pushrepo "github.com/bulatminnakhmetov/brigadka-backend/internal/repository/push"
	pushservice "github.com/bulatminnakhmetov/brigadka-backend/internal/service/push"

	firebase "firebase.google.com/go/v4"
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
		Host:     getEnv("DB_HOST", nil),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", nil),
		Password: getEnv("DB_PASSWORD", nil),
		DBName:   getEnv("DB_NAME", nil),
		SSLMode:  getEnv("DB_SSL_MODE", ptr("disable")),
	}

	jwtSecret := getEnv("JWT_SECRET", nil)
	serverPort := getEnv("SERVER_PORT", ptr("8080"))
	appVersion := getEnv("APP_VERSION", ptr("dev"))

	// Подключение к базе данных
	db, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Инициализация S3-совместимого хранилища для Backblaze B2
	s3Storage, err := mediastorage.NewS3StorageProvider(
		getEnv("B2_ACCESS_KEY_ID", nil),
		getEnv("B2_SECRET_ACCESS_KEY", nil),
		getEnv("B2_ENDPOINT", nil), // Выберите нужный регион
		getEnv("B2_BUCKET_NAME", nil),
		getEnv("CLOUDFLARE_CDN_DOMAIN", nil),
		"media", // Путь для загрузки в бакете
		getEnv("B2_PUBLIC_ENDPOINT", ptr("")),
	)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	mediaRepo := mediarepo.NewRepository(db)

	// Инициализация сервиса медиа
	mediaService := mediaservice.NewMediaService(mediaRepo, s3Storage)

	// Инициализация репозитория и хендлера авторизации
	userRepo := userrepo.NewPostgresUserRepository(db)
	authService := authservice.NewAuthService(userRepo, jwtSecret)
	authHandler := auth.NewAuthHandler(authService)

	// Инициализация сервиса и хендлера профилей
	profileRepo := profilerepo.NewPostgresRepository(db)
	profileService := profileservice.NewProfileService(profileRepo, mediaRepo)
	profileHandler := profile.NewProfileHandler(profileService)

	// Инициализация хендлера медиа
	mediaHandler := media.NewMediaHandler(mediaService)

	// Load APNS private key
	apnsPrivateKey := []byte{}
	apnsPrivateKeySource := getEnv("APNS_PRIVATE_KEY", ptr(""))
	if apnsPrivateKeySource != "" {
		var err error
		apnsPrivateKey, err = LoadAPNSPrivateKey(apnsPrivateKeySource)
		if err != nil {
			log.Printf("Warning: Failed to load APNS private key: %v", err)
		}
	}

	pushRepo := pushrepo.NewPostgresRepository(db)
	pushConfig := pushservice.Config{
		APNSKeyID:  getEnv("APNS_KEY_ID", ptr("")),
		APNSTeamID: getEnv("APNS_TEAM_ID", ptr("")),
		// In a real implementation, load private key from file or environment
		APNSPrivateKey:  apnsPrivateKey,
		APNSBundleID:    getEnv("APNS_BUNDLE_ID", ptr("")),
		APNSDevelopment: getEnv("APP_ENV", ptr("development")) != "production",
	}

	// Initialize Firebase app

	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("error initializing app: %v", err)
	}

	// Get Messaging client
	firebaseClient, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error getting Messaging client: %v", err)
	}

	pushService := pushservice.NewPushService(pushRepo, pushConfig, firebaseClient)
	pushHandler := pushhandler.NewHandler(pushService)

	// Инициализация сервиса и хендлера сообщений
	messagingRepo := messagingrepo.NewRepository(db)
	messagingService := messagingservice.NewService(messagingRepo, profileRepo)
	messagingHandler := messaging.NewHandler(messagingService, profileService, pushService)

	// Создание роутера
	r := chi.NewRouter()

	// Базовые middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(logging.ErrorLogger)

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
			"environment": getEnv("APP_ENV", ptr("development")),
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
		r.Post("/refresh", authHandler.RefreshToken)
	})

	// Защищенные маршруты (требуют аутентификации)
	r.Group(func(r chi.Router) {
		r.Use(authHandler.AuthMiddleware)

		r.Route("/api", func(r chi.Router) {
			r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
				userID := r.Context().Value("user_id").(int)
				email := r.Context().Value("email").(string)
				w.Write([]byte(fmt.Sprintf("Protected resource. User ID: %d, Email: %s", userID, email)))
			})

			// Маршруты для работы с профилями (требуют аутентификации)
			r.Route("/profiles", func(r chi.Router) {

				r.Post("/", profileHandler.CreateProfile)
				r.Get("/{userID}", profileHandler.GetProfile)
				r.Patch("/{userID}", profileHandler.UpdateProfile)

				// Регистрация обработчиков для справочников
				r.Route("/catalog", func(r chi.Router) {
					r.Get("/improv-styles", profileHandler.GetImprovStyles)
					r.Get("/improv-goals", profileHandler.GetImprovGoals)
					r.Get("/genders", profileHandler.GetGenders)
					r.Get("/cities", profileHandler.GetCities)
				})

				r.Post("/search", profileHandler.SearchProfiles)
			})

			// Маршруты для работы с медиа (требуют аутентификации)
			r.Route("/media", func(r chi.Router) {
				r.Post("/", mediaHandler.UploadMedia)
			})

			// Маршруты для работы с сообщениями (требуют аутентификации)
			r.Post("/chats", messagingHandler.CreateChat)
			r.Get("/chats", messagingHandler.GetUserChats)
			r.Post("/chats/direct", messagingHandler.GetOrCreateDirectChat)
			r.Get("/chats/{chatID}", messagingHandler.GetChat)
			r.Get("/chats/{chatID}/messages", messagingHandler.GetChatMessages)
			r.Post("/chats/{chatID}/messages", messagingHandler.SendMessage)
			r.Post("/chats/{chatID}/participants", messagingHandler.AddParticipant)
			r.Delete("/chats/{chatID}/participants/{userID}", messagingHandler.RemoveParticipant)
			r.Post("/messages/{messageID}/reactions", messagingHandler.AddReaction)
			r.Delete("/messages/{messageID}/reactions/{reactionCode}", messagingHandler.RemoveReaction)
			r.HandleFunc("/ws/chat", messagingHandler.HandleWebSocket)

			r.Post("/push/register", pushHandler.RegisterToken)
			r.Delete("/push/unregister", pushHandler.UnregisterToken)
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
func getEnv(key string, fallback *string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	if fallback == nil {
		panic(fmt.Sprintf("Environment variable %s is not set and no fallback provided", key))
	}
	return *fallback
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

// LoadAPNSPrivateKey loads an APNS private key from a file path or from base64-encoded environment variable
func LoadAPNSPrivateKey(source string) ([]byte, error) {
	// Check if the source is a file path
	if strings.HasPrefix(source, "file://") {
		filePath := strings.TrimPrefix(source, "file://")
		return os.ReadFile(filePath)
	}

	// Check if the source is a base64-encoded string
	if strings.HasPrefix(source, "base64://") {
		encodedKey := strings.TrimPrefix(source, "base64://")
		return base64.StdEncoding.DecodeString(encodedKey)
	}

	// If the source is empty, return empty key
	if source == "" {
		return []byte{}, nil
	}

	// Otherwise assume it's the actual key content
	return []byte(source), nil
}

func ptr[T any](val T) *T {
	return &val
}
