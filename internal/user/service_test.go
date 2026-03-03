package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockUserDao struct {
	usr Model
	err error
}

func (m mockUserDao) GetUserById(context.Context, uuid.UUID) (*Model, error) {
	return &m.usr, m.err
}

func (m mockUserDao) GetUserByEmail(context.Context, string) (*Model, error) {
	return &m.usr, m.err
}

func (m mockUserDao) SaveUser(context.Context, *Model) error {
	return m.err
}

func (m mockUserDao) DeleteUserById(context.Context, string) error {
	return m.err
}

var _ DAO = mockUserDao{}

func TestService_GetUserByID(t *testing.T) {
	t.Run("should be able to get user by id", func(t *testing.T) {
		tUsr := Model{}
		mockDao := mockUserDao{
			usr: tUsr,
		}
		svc := NewService(mockDao)
		usr, err := svc.GetUserByID(t.Context(), uuid.New().String())
		require.NoError(t, err)
		require.Equal(t, &tUsr, usr)
	})

	t.Run("should return error if database throws error", func(t *testing.T) {
		mockDao := mockUserDao{
			err: errors.New("database error"),
		}
		svc := NewService(mockDao)
		_, err := svc.GetUserByID(t.Context(), uuid.New().String())
		require.Error(t, err)
	})

	t.Run("should throw error on invalid UUID format", func(t *testing.T) {
		mockDao := mockUserDao{}
		svc := NewService(mockDao)
		_, err := svc.GetUserByID(t.Context(), "invalid-uuid")
		require.Error(t, err)
	})
}

func TestService_CreateUser(t *testing.T) {
	t.Run("should be able to create user", func(t *testing.T) {
		tUsr := Model{}
		mockDao := mockUserDao{}
		svc := NewService(mockDao)
		_, err := svc.CreateUser(t.Context(), tUsr)
		require.NoError(t, err)
	})

	t.Run("should throw error if database throws error", func(t *testing.T) {
		mockDao := mockUserDao{err: errors.New("database error")}
		svc := NewService(mockDao)
		_, err := svc.CreateUser(t.Context(), Model{})
		require.Error(t, err)
	})

	t.Run("should return error if user already exists", func(t *testing.T) {
		mockDao := mockUserDao{
			err: sql.ErrNoRows,
			usr: Model{},
		}
		svc := NewService(mockDao)
		_, err := svc.CreateUser(t.Context(), Model{})
		require.Error(t, err)
	})
}
