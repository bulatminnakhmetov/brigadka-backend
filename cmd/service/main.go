package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/bulatminnakhmetov/brigadka-backend/internal/auth"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/database"
	"github.com/bulatminnakhmetov/brigadka-backend/internal/profile"
)

func main() {
	// Загрузка конфигурации из переменных окружения
	dbConfig := &database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "brigadka"),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}

	jwtSecret := getEnv("JWT_SECRET", "your-secret-key-replace-in-production")
	serverPort := getEnv("SERVER_PORT", "8080")

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

	// Создание роутера
	r := chi.NewRouter()

	// Базовые middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS middleware
	// r.Use(cors.Handler(cors.Options{
	// 	AllowedOrigins:   []string{"*"}, // В продакшене лучше ограничить конкретными доменами
	// 	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
	// 	ExposedHeaders:   []string{"Link"},
	// 	AllowCredentials: true,
	// 	MaxAge:           300, // Maximum value not readily apparent
	// }))

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
			// Здесь можно добавить другие методы для работы с профилями
			// r.Get("/{id}", profileHandler.GetProfile)
			// r.Put("/{id}", profileHandler.UpdateProfile)
			// r.Delete("/{id}", profileHandler.DeleteProfile)
		})

		// Здесь можно добавить другие защищенные маршруты
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
