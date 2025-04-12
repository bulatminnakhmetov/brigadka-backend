package auth

import (
	"database/sql"
	"errors"
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

// GetUserByEmail получает пользователя по email
func (r *PostgresUserRepository) GetUserByEmail(email string) (*User, error) {
	query := `
        SELECT user_id, email, password_hash, full_name, gender, age, city_id 
        FROM users 
        WHERE email = $1
    `

	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.UserID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Gender,
		&user.Age,
		&user.CityID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

// CreateUser создает нового пользователя в базе данных
func (r *PostgresUserRepository) CreateUser(user *User) error {
	query := `
        INSERT INTO users (email, password_hash, full_name, gender, age, city_id)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING user_id
    `

	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Gender,
		user.Age,
		user.CityID,
	).Scan(&user.UserID)

	if err != nil {
		return err
	}

	return nil
}
