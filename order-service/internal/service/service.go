package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/dimaglobin/order-service/internal/model"
)

type Service struct {
	repo Repository
	log  *slog.Logger
}

func NewService(repo Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

func (s *Service) CreateOrder(ctx context.Context, order *model.Order) error {
	order.Status = model.StatusNew
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now
	return s.repo.Create(ctx, order)
}

func (s *Service) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}
