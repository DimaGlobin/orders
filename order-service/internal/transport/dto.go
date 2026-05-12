package transport

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
)

// ── Requests ─────────────────────────────────────────────────────────────────

type CreateOrderRequest struct {
	UserID uuid.UUID           `json:"user_id"`
	Items  []CreateItemRequest `json:"items"`
}

func (r CreateOrderRequest) Validate() error {
	if r.UserID == uuid.Nil {
		return apperrors.NewValidationError("user_id", "must be a valid uuid")
	}
	if len(r.Items) == 0 {
		return apperrors.NewValidationError("items", "must contain at least one item")
	}
	for i, item := range r.Items {
		if item.ProductID == uuid.Nil {
			return apperrors.NewValidationError(
				fmt.Sprintf("items[%d].product_id", i), "must be a valid uuid")
		}
		if item.Quantity <= 0 {
			return apperrors.NewValidationError(
				fmt.Sprintf("items[%d].quantity", i), "must be a positive integer")
		}
		if item.Price <= 0 {
			return apperrors.NewValidationError(
				fmt.Sprintf("items[%d].price", i), "must be a positive integer")
		}
	}
	return nil
}

type CreateItemRequest struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     int64     `json:"price"` // cents
}

// ── Responses ─────────────────────────────────────────────────────────────────

type OrderResponse struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id"`
	Status    string         `json:"status"`
	Items     []ItemResponse `json:"items"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type ItemResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     int64     `json:"price"` // cents
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func toOrdersResponse(orders []*model.Order) []OrderResponse {
	result := make([]OrderResponse, len(orders))
	for i, o := range orders {
		result[i] = toOrderResponse(o)
	}
	return result
}

func toOrderResponse(o *model.Order) OrderResponse {
	items := make([]ItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = ItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return OrderResponse{
		ID:        o.ID,
		UserID:    o.UserID,
		Status:    string(o.Status),
		Items:     items,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
