package model

import "time"

// OrderEvent mirrors the event published by order-service to the "orders" topic.
// Keep this struct in sync with order-service/internal/model/event.go.
//
// IDs are UUID strings; we keep them as strings here because the notifier only
// uses them for logging and email rendering — no validation needed.
//
// Schema version: 1
type OrderEvent struct {
	EventID    string           `json:"event_id"`
	Type       string           `json:"type"`
	Version    int              `json:"version"`
	OrderID    string           `json:"order_id"`
	UserID     string           `json:"user_id"`
	Status     string           `json:"status"`
	Items      []OrderEventItem `json:"items"`
	OccurredAt time.Time        `json:"occurred_at"`
}

type OrderEventItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Price     int64  `json:"price"` // cents
}

const (
	EventOrderCreated   = "order.created"
	EventOrderCancelled = "order.cancelled"
)
