package profile

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *PostgresRepository) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	repo := NewPostgresRepository(db)
	return db, mock, repo
}

func TestCheckUserExists(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)")).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.CheckUserExists(1)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCheckProfileExists(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM profiles WHERE user_id = $1)")).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.CheckProfileExists(2)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGetProfile_NotFound(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT user_id, full_name, birthday, gender, city_id, 
               bio, goal, looking_for_team, created_at 
        FROM profiles WHERE user_id = $1
    `)).
		WithArgs(3).
		WillReturnError(sql.ErrNoRows)

	profile, err := repo.GetProfile(3)
	assert.Nil(t, profile)
	assert.Equal(t, ErrProfileNotExists, err)
}

func TestGetProfileAvatar_NoAvatar(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT media_id FROM profile_media 
        WHERE user_id = $1 AND role = 'avatar'
        LIMIT 1
    `)).
		WithArgs(4).
		WillReturnError(sql.ErrNoRows)

	avatar, err := repo.GetProfileAvatar(4)
	assert.NoError(t, err)
	assert.Nil(t, avatar)
}

func TestAddImprovStyles(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta(`
            INSERT INTO improv_profile_styles (user_id, style)
            VALUES ($1, $2)
        `)).
		WithArgs(5, "style1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
            INSERT INTO improv_profile_styles (user_id, style)
            VALUES ($1, $2)
        `)).
		WithArgs(5, "style2").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddImprovStyles(tx, 5, []string{"style1", "style2"})
	assert.NoError(t, err)
	tx.Rollback()
}

func TestUpdateProfile_NoFields(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	update := &UpdateProfileModel{UserID: 1}
	err = repo.UpdateProfile(tx, update)
	assert.NoError(t, err)
	tx.Rollback()
}

func TestUpdateProfile_WithFields(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectBegin()
	tx, err := db.Begin()
	assert.NoError(t, err)

	fullName := "Test User"
	update := &UpdateProfileModel{UserID: 1, FullName: &fullName}

	mock.ExpectExec(regexp.QuoteMeta("UPDATE profiles SET full_name = $1 WHERE user_id = $2")).
		WithArgs(fullName, 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.UpdateProfile(tx, update)
	assert.NoError(t, err)
	tx.Rollback()
}

func TestValidateImprovGoal(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS(SELECT 1 FROM improv_goals_catalog WHERE goal_id = $1)")).
		WithArgs("goal1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	valid, err := repo.ValidateImprovGoal("goal1")
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestGetImprovStylesCatalog(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT isc.style_code, ist.label
        FROM improv_style_catalog isc
        LEFT JOIN improv_style_translation ist ON isc.style_code = ist.style_code AND ist.lang = $1
    `)).
		WithArgs("ru").
		WillReturnRows(sqlmock.NewRows([]string{"style_code", "label"}).
			AddRow("style1", "Style 1").
			AddRow("style2", "Style 2"))

	items, err := repo.GetImprovStylesCatalog("ru")
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "style1", items[0].Code)
	assert.Equal(t, "Style 1", items[0].Label)
}
