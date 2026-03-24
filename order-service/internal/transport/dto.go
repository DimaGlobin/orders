package transport

import "time"

type CreateOrderRequest struct {
	UserID int64              `json:"user_id"`
	Items  []CreateItemRequest `json:"items"`
}

type CreateItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	Price     int64 `json:"price"`
}

type OrderResponse struct {
	ID        int64          `json:"id"`
	UserID    int64          `json:"user_id"`
	Status    string         `json:"status"`
	Items     []ItemResponse `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
}

type ItemResponse struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
	Price     int64 `json:"price"`
}
