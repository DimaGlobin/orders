package model

import "time"

type EventType string

const (
	EventOrderCreated   EventType = "order.created"
	EventOrderCancelled EventType = "order.cancelled"
)

// OrderEvent is the Kafka message published to the "orders" topic.
// Both EventOrderCreated and EventOrderCancelled use this struct — consumers
// check the Type field to decide how to handle the message.
//
// Schema version: 1
// Topic:          orders
// Key:            strconv.FormatInt(OrderID, 10)  — guarantees per-order ordering
type OrderEvent struct {
	Type       EventType        `json:"type"`
	Version    int              `json:"version"` // bump on breaking schema changes
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
