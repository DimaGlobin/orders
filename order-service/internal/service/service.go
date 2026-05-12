package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/apperrors"
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

func (s *Service) GetOrder(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) CancelOrder(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	order, err := s.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.Status != model.StatusNew {
		return nil, fmt.Errorf("cannot cancel order with status %q: %w", order.Status, apperrors.ErrConflict)
	}
	return s.repo.UpdateStatus(ctx, id, model.StatusCancelled)
}

func (s *Service) ListOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	return s.repo.ListOrders(ctx, userID)
}
