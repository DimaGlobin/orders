package model

import "time"

// OrderEvent mirrors the event published by order-service to the "orders" topic.
// Keep this struct in sync with order-service/internal/model/event.go.
//
// Schema version: 1
type OrderEvent struct {
	EventID    string           `json:"event_id"`
	Type       string           `json:"type"`
	Version    int              `json:"version"`
	OrderID    int64            `json:"order_id"`
	UserID     int64            `json:"user_id"`
	Status     string           `json:"status"`
	Items      []OrderEventItem `json:"items"`
	OccurredAt time.Time        `json:"occurred_at"`
}

type OrderEventItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	Price     int64 `json:"price"` // cents
}

const (
	EventOrderCreated   = "order.created"
	EventOrderCancelled = "order.cancelled"
)
