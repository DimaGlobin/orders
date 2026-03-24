package service

import (
	"context"

	"github.com/dimaglobin/order-service/internal/model"
)

type Repository interface {
	Create(ctx context.Context, order *model.Order) error
	GetByID(ctx context.Context, id int64) (*model.Order, error)
}
