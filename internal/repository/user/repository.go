package user

import (
	"database/sql"
	"errors"
)

type UserInfo struct {
	FullName string `json:"full_name"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty"`
	CityID   int    `json:"city_id,omitempty"`
}

type User struct {
	UserInfo
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
        SELECT id, email, password_hash, full_name, gender, age, city_id 
        FROM users 
        WHERE email = $1
    `

	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Gender,
		&user.Age,
		&user.CityID,
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
        INSERT INTO users (email, password_hash, full_name, gender, age, city_id)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `

	err := r.db.QueryRow(
		query,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.Gender,
		user.Age,
		user.CityID,
	).Scan(&user.ID)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID получает пользователя по ID
func (r *PostgresUserRepository) GetUserByID(id int) (*User, error) {
	query := `
        SELECT id, email, password_hash, full_name, gender, age, city_id 
        FROM users 
        WHERE id = $1
    `

	var user User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.Gender,
		&user.Age,
		&user.CityID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// GetUserByID получает пользователя по ID
func (r *PostgresUserRepository) GetUserInfoByID(id int) (*UserInfo, error) {
	user, err := r.GetUserByID(id)
	if err != nil {
		return nil, err
	}

	return &user.UserInfo, nil
}

// UpdateUserInfo updates the user information for a given user ID
func (r *PostgresUserRepository) UpdateUserInfo(tx *sql.Tx, userID int, info *UserInfo) error {
	query := `
        UPDATE users 
        SET full_name = $1, gender = $2, age = $3, city_id = $4
        WHERE id = $5
    `

	result, err := tx.Exec(
		query,
		info.FullName,
		info.Gender,
		info.Age,
		info.CityID,
		userID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
