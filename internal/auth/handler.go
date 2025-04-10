package auth

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type RegisterCredentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	CityID   int    `json:"city_id" binding:"required"`
	Gender   string `json:"gender" binding:"required"`
	Age      int    `json:"age" binding:"required"`
}

type LoginCredentials struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthController struct {
	DB     *sql.DB
	JWTKey []byte
}

func (ac *AuthController) Register(c *gin.Context) {
	var creds RegisterCredentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input. All fields are required"})
		return
	}

	// Validate age
	if creds.Age < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid age value"})
		return
	}

	// Validate city exists
	var cityExists bool
	err := ac.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM cities WHERE city_id = $1)", creds.CityID).Scan(&cityExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !cityExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid city ID"})
		return
	}

	// Check if gender exists in gender_catalog
	var genderExists bool
	err = ac.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM gender_catalog WHERE gender_code = $1)", creds.Gender).Scan(&genderExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !genderExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gender code"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hashing failed"})
		return
	}

	_, err = ac.DB.Exec(
		`INSERT INTO users (email, password_hash, full_name, city_id, gender, age) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		creds.Email,
		string(hash),
		creds.FullName,
		creds.CityID,
		creds.Gender,
		creds.Age,
	)
	if err != nil {
		log.Printf("Database error in Register: %v", err)
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already registered"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered"})
}

func (ac *AuthController) Login(c *gin.Context) {
	var creds LoginCredentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	var userID int
	var storedHash string
	err := ac.DB.QueryRow("SELECT user_id, password_hash FROM users WHERE email = $1", creds.Email).Scan(&userID, &storedHash)
	if err != nil {
		log.Printf("Database error in Login: %v", err)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(creds.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(ac.JWTKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}
