package model

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusNew       OrderStatus = "new"
	StatusConfirmed OrderStatus = "confirmed"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Status    OrderStatus
	Items     []OrderItem
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	Price     int64 // cents
	CreatedAt time.Time
}
