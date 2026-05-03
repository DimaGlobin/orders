package transport

import (
	"context"

	"github.com/dimaglobin/order-service/internal/model"
)

type OrderService interface {
	CreateOrder(ctx context.Context, order *model.Order) error
	GetOrder(ctx context.Context, id int64) (*model.Order, error)
	CancelOrder(ctx context.Context, id int64) (*model.Order, error)
	ListByUser(ctx context.Context, userID int64) ([]*model.Order, error)
}
