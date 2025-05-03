package user

import (
	"database/sql"
	"errors"
)

type User struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

var (
	ErrUserNotFound = errors.New("user not found")
)

type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository создает новый экземпляр репозитория пользователей
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// BeginTx starts a new transaction
func (r *PostgresUserRepository) BeginTx() (*sql.Tx, error) {
	return r.db.Begin()
}

// GetUserByEmail получает пользователя по email
func (r *PostgresUserRepository) GetUserByEmail(email string) (*User, error) {
	query := `
        SELECT id, email, password_hash
        FROM users 
        WHERE email = $1
    `

	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// CreateUser создает нового пользователя в базе данных
func (r *PostgresUserRepository) CreateUser(user *User) error {
	query := `
        INSERT INTO users (email, password_hash)
        VALUES ($1, $2)
        RETURNING id
    `

	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
	).Scan(&user.ID)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID получает пользователя по ID
func (r *PostgresUserRepository) GetUserByID(id int) (*User, error) {
	query := `
        SELECT id, email, password_hash
        FROM users 
        WHERE id = $1
    `

	var user User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
