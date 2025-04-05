package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func main() {
	r := gin.Default()

	r.POST("/echo", func(c *gin.Context) {
		var body map[string]interface{}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		c.JSON(http.StatusOK, body)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // значение по умолчанию
	}

	r.Run(":" + port)
}
