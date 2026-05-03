package service

import (
	"context"

	"github.com/dimaglobin/order-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id int64) (*model.Order, error)
	UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error)
	ListByUserID(ctx context.Context, userID int64) ([]*model.Order, error)
}
