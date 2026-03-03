package user

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(db, "postgres")
	return sqlxDB, mock, func() { db.Close() }
}

func TestDao_GetUserById(t *testing.T) {
	t.Run("should get user by id", func(t *testing.T) {
		sqlxDB, mock, teardown := setupMockDB(t)
		defer teardown()
		tDao := NewDAO(sqlxDB)

		user := &Model{
			ID:        uuid.New(),
			Username:  "carol",
			Email:     "carol@example.com",
			CreatedAt: time.Now(),
		}

		rows := sqlmock.NewRows([]string{"id", "username", "email", "created_at"}).
			AddRow(user.ID.String(), user.Username, user.Email, user.CreatedAt)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users WHERE id=$1`)).
			WithArgs(user.ID.String()).
			WillReturnRows(rows)

		usr, err := tDao.GetUserById(t.Context(), user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user, usr)
	})

	t.Run("should return error if user does not exist", func(t *testing.T) {
		sqlxDB, mock, teardown := setupMockDB(t)
		defer teardown()
		tDao := NewDAO(sqlxDB)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol@example.com",
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users WHERE id=$1`)).
			WithArgs(user.ID.String()).
			WillReturnError(errors.New("user not found"))

		usr, err := tDao.GetUserById(t.Context(), user.ID)

		assert.Nil(t, usr)
		assert.Error(t, err)
	})
}

func TestDao_GetUserByEmail(t *testing.T) {
	t.Run("should get user by email", func(t *testing.T) {
		sqlxDB, mock, teardown := setupMockDB(t)
		defer teardown()
		tDao := NewDAO(sqlxDB)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol@example.com",
		}
		rows := sqlmock.NewRows([]string{"id", "username", "email", "created_at"}).
			AddRow(user.ID.String(), user.Username, user.Email, user.CreatedAt)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users WHERE email=$1`)).
			WithArgs(user.Email).
			WillReturnRows(rows)

		usr, err := tDao.GetUserByEmail(t.Context(), user.Email)
		assert.NoError(t, err)
		assert.Equal(t, user, usr)
	})

	t.Run("should return error if user does not exist", func(t *testing.T) {
		sqlxDB, mock, teardown := setupMockDB(t)
		defer teardown()
		tDao := NewDAO(sqlxDB)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol@example.com",
		}

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM users WHERE email=$1`)).
			WithArgs(user.ID.String()).
			WillReturnError(errors.New("user not found"))

		usr, err := tDao.GetUserByEmail(t.Context(), user.ID.String())

		assert.Nil(t, usr)
		assert.Error(t, err)
	})
}

func TestDao_SaveUser(t *testing.T) {
	t.Run("should save user", func(t *testing.T) {
		db, mock, cleanup := setupMockDB(t)
		defer cleanup()
		tDao := NewDAO(db)

		user := &Model{
			ID:        uuid.New(),
			Username:  "carol",
			Email:     "carol@example.com",
			CreatedAt: time.Now(),
		}

		mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO users (id, username, email, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    created_at = EXCLUDED.created_at,
    permissions = EXCLUDED.permissions
`)).
			WithArgs(user.ID, user.Username, user.Email, user.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := tDao.SaveUser(t.Context(), user)
		assert.NoError(t, err)
	})

	t.Run("should not fail if user already exists", func(t *testing.T) {
		db, mock, cleanup := setupMockDB(t)
		defer cleanup()
		tDao := NewDAO(db)
		user := &Model{
			ID:        uuid.New(),
			Username:  "carol",
			Email:     "john.doe@gmail.com",
			CreatedAt: time.Now(),
		}
		mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO users (id, username, email, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    created_at = EXCLUDED.created_at,
    permissions = EXCLUDED.permissions
`)).
			WithArgs(user.ID, user.Username, user.Email, user.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := tDao.SaveUser(t.Context(), user)
		assert.NoError(t, err)
	})

	t.Run("should return error if cannot save user", func(t *testing.T) {
		db, mock, cleanup := setupMockDB(t)
		defer cleanup()
		tDao := NewDAO(db)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol.test@gmail.com",
		}
		mock.ExpectExec(regexp.QuoteMeta(`
INSERT INTO users (id, username, email, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    created_at = EXCLUDED.created_at,
    permissions = EXCLUDED.permissions
`)).
			WithArgs(user.ID, user.Username, user.Email, user.CreatedAt).
			WillReturnError(sql.ErrNoRows)
		err := tDao.SaveUser(t.Context(), user)
		assert.Error(t, err)
	})
}

func TestDao_DeleteUserById(t *testing.T) {
	t.Run("should delete user", func(t *testing.T) {
		db, mock, cleanup := setupMockDB(t)
		defer cleanup()
		tDao := NewDAO(db)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol.test@gmail.com",
		}
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id=$1`)).
			WithArgs(user.ID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		err := tDao.DeleteUserById(t.Context(), user.ID.String())
		assert.NoError(t, err)
	})

	t.Run("should return error if user does not exist", func(t *testing.T) {
		db, mock, cleanup := setupMockDB(t)
		defer cleanup()
		tDao := NewDAO(db)
		user := &Model{
			ID:       uuid.New(),
			Username: "carol",
			Email:    "carol.test@gmail.com",
		}

		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM users WHERE id=$1`)).
			WithArgs(user.ID.String()).
			WillReturnError(errors.New("user not found"))
		err := tDao.DeleteUserById(t.Context(), user.ID.String())
		assert.Error(t, err)
	})
}
