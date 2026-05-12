package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (*model.Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}
