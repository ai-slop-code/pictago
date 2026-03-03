package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DAO interface {
	GetUserById(ctx context.Context, id uuid.UUID) (*Model, error)
	GetUserByEmail(ctx context.Context, email string) (*Model, error)
	SaveUser(ctx context.Context, user *Model) error
	DeleteUserById(ctx context.Context, id string) error
}

type dao struct {
	db *sqlx.DB
}

func (d *dao) GetUserById(ctx context.Context, id uuid.UUID) (*Model, error) {
	var user Model
	query := `SELECT * FROM users WHERE id=$1`
	err := d.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *dao) GetUserByEmail(ctx context.Context, email string) (*Model, error) {
	var user Model
	query := `SELECT * FROM users WHERE email=$1`
	err := d.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (d *dao) SaveUser(ctx context.Context, user *Model) error {
	query := `
INSERT INTO users (id, username, email, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE SET
    username = EXCLUDED.username,
    email = EXCLUDED.email,
    created_at = EXCLUDED.created_at,
    permissions = EXCLUDED.permissions
`
	_, err := d.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Username,
		user.Email,
		user.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}
	return nil
}

func (d *dao) DeleteUserById(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id=$1`
	_, err := d.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

var _ DAO = &dao{}

func NewDAO(db *sqlx.DB) DAO {
	return &dao{db: db}
}
