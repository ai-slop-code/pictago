package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ai-slop-code/pictago/internal/log"
	"github.com/google/uuid"
)

type Service interface {
	GetUserByID(ctx context.Context, userID string) (*Model, error)
	CreateUser(ctx context.Context, user Model) (*Model, error)
}

type service struct {
	dao    DAO
	logger log.Logger
}

func (s *service) GetUserByID(ctx context.Context, userID string) (*Model, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Failed to parse user id", "error", err)
		return nil, err
	}
	return s.dao.GetUserById(ctx, id)
}

func (s *service) CreateUser(ctx context.Context, user Model) (*Model, error) {
	curUsr, err := s.dao.GetUserByEmail(ctx, user.Email)
	if errors.Is(err, sql.ErrNoRows) && curUsr != nil {
		return nil, errors.New("user with this email already exists")
	}
	user.CreatedAt = time.Now()
	user.ID = uuid.New()
	if err := s.dao.SaveUser(ctx, &user); err != nil {
		s.logger.Error("Failed to save user", "error", err)
		return nil, err
	}
	s.logger.Info("Created user", "user", user)
	return &user, nil
}

var _ Service = &service{}

func NewService(dao DAO) Service {
	return &service{
		dao:    dao,
		logger: log.NewDefaultLogger(),
	}
}
