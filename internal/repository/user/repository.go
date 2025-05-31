package user

import (
	"database/sql"
	"errors"
)

type User struct {
	ID            int    `json:"id"`
	Email         string `json:"email"`
	PasswordHash  string `json:"-"`
	EmailVerified bool   `json:"email_verified"`
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
        SELECT id, email, password_hash, email_verified
        FROM users 
        WHERE email = $1
    `

	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
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
        INSERT INTO users (email, password_hash, email_verified)
        VALUES ($1, $2, $3)
        RETURNING id
    `

	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.EmailVerified,
	).Scan(&user.ID)

	if err != nil {
		return err
	}

	return nil
}

// UpdateUser updates an existing user's information
func (r *PostgresUserRepository) UpdateUser(user *User) error {
	query := `
        UPDATE users
        SET email = $2, password_hash = $3, email_verified = $4
        WHERE id = $1
    `

	_, err := r.db.Exec(
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.EmailVerified,
	)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID получает пользователя по ID
func (r *PostgresUserRepository) GetUserByID(id int) (*User, error) {
	query := `
        SELECT id, email, password_hash, email_verified
        FROM users 
        WHERE id = $1
    `

	var user User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// UpdateEmailVerificationStatus updates the email verification status for a user
func (r *PostgresUserRepository) UpdateEmailVerificationStatus(userID int, verified bool) error {
	query := `
        UPDATE users
        SET email_verified = $2
        WHERE id = $1
    `

	_, err := r.db.Exec(query, userID, verified)
	return err
}
