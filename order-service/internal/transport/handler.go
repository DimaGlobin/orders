package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dimaglobin/order-service/internal/model"
)

type Handler struct {
	orders OrderService
	log    *slog.Logger
}

func NewHandler(orders OrderService, log *slog.Logger) *Handler {
	return &Handler{
		orders: orders,
		log:    log,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("failed to decode request", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	items := make([]model.OrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = model.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	order := &model.Order{
		UserID: req.UserID,
		Items:  items,
	}

	if err := h.orders.CreateOrder(r.Context(), order); err != nil {
		h.log.Error("failed to create order", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := toOrderResponse(order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid order id", http.StatusBadRequest)
		return
	}

	order, err := h.orders.GetOrder(r.Context(), id)
	if err != nil {
		h.log.Error("failed to get order", "error", err, "order_id", id)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := toOrderResponse(order)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func toOrderResponse(order *model.Order) OrderResponse {
	items := make([]ItemResponse, len(order.Items))
	for i, item := range order.Items {
		items[i] = ItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	return OrderResponse{
		ID:        order.ID,
		UserID:    order.UserID,
		Status:    string(order.Status),
		Items:     items,
		CreatedAt: order.CreatedAt,
	}
}
