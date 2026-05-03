package transport_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/transport"
)

func TestCreateOrderRequest_Validate(t *testing.T) {
	validItem := transport.CreateItemRequest{ProductID: 10, Quantity: 1, Price: 100}

	tests := []struct {
		name      string
		req       transport.CreateOrderRequest
		wantOK    bool
		wantField string // substring expected in error
	}{
		{
			name:   "valid",
			req:    transport.CreateOrderRequest{UserID: 1, Items: []transport.CreateItemRequest{validItem}},
			wantOK: true,
		},
		{
			name:      "user_id zero",
			req:       transport.CreateOrderRequest{UserID: 0, Items: []transport.CreateItemRequest{validItem}},
			wantField: "user_id",
		},
		{
			name:      "user_id negative",
			req:       transport.CreateOrderRequest{UserID: -1, Items: []transport.CreateItemRequest{validItem}},
			wantField: "user_id",
		},
		{
			name:      "items empty",
			req:       transport.CreateOrderRequest{UserID: 1, Items: nil},
			wantField: "items",
		},
		{
			name: "product_id zero",
			req: transport.CreateOrderRequest{UserID: 1, Items: []transport.CreateItemRequest{
				{ProductID: 0, Quantity: 1, Price: 1},
			}},
			wantField: "items[0].product_id",
		},
		{
			name: "quantity zero",
			req: transport.CreateOrderRequest{UserID: 1, Items: []transport.CreateItemRequest{
				{ProductID: 1, Quantity: 0, Price: 1},
			}},
			wantField: "items[0].quantity",
		},
		{
			name: "price zero",
			req: transport.CreateOrderRequest{UserID: 1, Items: []transport.CreateItemRequest{
				{ProductID: 1, Quantity: 1, Price: 0},
			}},
			wantField: "items[0].price",
		},
		{
			name: "second item invalid",
			req: transport.CreateOrderRequest{UserID: 1, Items: []transport.CreateItemRequest{
				validItem,
				{ProductID: 1, Quantity: 1, Price: -5},
			}},
			wantField: "items[1].price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantOK {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !errors.Is(err, apperrors.ErrValidation) {
				t.Errorf("expected errors.Is(err, ErrValidation), got %v", err)
			}
			if !strings.Contains(err.Error(), tt.wantField) {
				t.Errorf("expected error to mention %q, got %q", tt.wantField, err.Error())
			}
		})
	}
}
