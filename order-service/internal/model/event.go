package model

import "time"

type OrderCreated struct {
	OrderID   int64     `json:"order_id"`
	UserID    int64     `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
