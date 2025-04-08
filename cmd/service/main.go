package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/Minnakhmetov/brigadka-backend/internal/auth"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	database, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	auth := &auth.AuthController{
		DB:     database,
		JWTKey: []byte("secret_key"),
	}

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	r.POST("/register", auth.Register)
	r.POST("/login", auth.Login)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // значение по умолчанию
	}

	r.Run("0.0.0.0:" + port)
}
