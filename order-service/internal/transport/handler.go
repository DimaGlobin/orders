package transport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
)

type Handler struct {
	orders OrderService
	log    *slog.Logger
}

func NewHandler(orders OrderService, log *slog.Logger) *Handler {
	return &Handler{orders: orders, log: log}
}

// RegisterRoutes wires all order endpoints into mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.Create)
	mux.HandleFunc("GET /orders/{id}", h.GetByID)
	mux.HandleFunc("GET /orders", h.ListByUser)
	mux.HandleFunc("POST /orders/{id}/cancel", h.CancelOrder)
}

// POST /orders
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		writeErrorFrom(w, err)
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
	order := &model.Order{UserID: req.UserID, Items: items}

	if err := h.orders.CreateOrder(r.Context(), order); err != nil {
		h.log.Error("create order", "error", err)
		writeErrorFrom(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toOrderResponse(order))
}

// GET /orders/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}

	order, err := h.orders.GetOrder(r.Context(), id)
	if err != nil {
		h.log.Error("get order", "error", err, "order_id", id)
		writeErrorFrom(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// GET /orders?user_id={id}
func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := parseID(w, r.URL.Query().Get("user_id"))
	if !ok {
		return
	}

	orders, err := h.orders.ListByUser(r.Context(), userID)
	if err != nil {
		h.log.Error("list orders", "error", err, "user_id", userID)
		writeErrorFrom(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrdersResponse(orders))
}

// POST /orders/{id}/cancel
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r.PathValue("id"))
	if !ok {
		return
	}

	order, err := h.orders.CancelOrder(r.Context(), id)
	if err != nil {
		h.log.Error("cancel order", "error", err, "order_id", id)
		writeErrorFrom(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toOrderResponse(order))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeErrorFrom(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperrors.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, apperrors.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, apperrors.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
