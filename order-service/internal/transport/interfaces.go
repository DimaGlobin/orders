package transport

import (
	"context"

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/model"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrder(ctx context.Context, id uuid.UUID) (*model.Order, error)
	CancelOrder(ctx context.Context, id uuid.UUID) (*model.Order, error)
	ListOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}
