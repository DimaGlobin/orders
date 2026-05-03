package transport_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
	"github.com/dimaglobin/order-service/internal/transport"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeService struct {
	createFn     func(ctx context.Context, order *model.Order) error
	getFn        func(ctx context.Context, id int64) (*model.Order, error)
	cancelFn     func(ctx context.Context, id int64) (*model.Order, error)
	listByUserFn func(ctx context.Context, userID int64) ([]*model.Order, error)
}

func (f *fakeService) CreateOrder(ctx context.Context, o *model.Order) error {
	return f.createFn(ctx, o)
}
func (f *fakeService) GetOrder(ctx context.Context, id int64) (*model.Order, error) {
	return f.getFn(ctx, id)
}
func (f *fakeService) CancelOrder(ctx context.Context, id int64) (*model.Order, error) {
	return f.cancelFn(ctx, id)
}
func (f *fakeService) ListByUser(ctx context.Context, userID int64) ([]*model.Order, error) {
	return f.listByUserFn(ctx, userID)
}

func newRequest(method, target, body string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestHandler_Create(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		svcErr     error
		wantStatus int
	}{
		{
			name:       "201 happy path",
			body:       `{"user_id":1,"items":[{"product_id":10,"quantity":2,"price":500}]}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "400 invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: empty items",
			body:       `{"user_id":1,"items":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: zero user_id",
			body:       `{"user_id":0,"items":[{"product_id":1,"quantity":1,"price":1}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: zero price",
			body:       `{"user_id":1,"items":[{"product_id":1,"quantity":1,"price":0}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "500 service error",
			body:       `{"user_id":1,"items":[{"product_id":10,"quantity":2,"price":500}]}`,
			svcErr:     errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				createFn: func(_ context.Context, _ *model.Order) error { return tt.svcErr },
			}
			h := transport.NewHandler(svc, discardLogger())

			w := httptest.NewRecorder()
			h.Create(w, newRequest(http.MethodPost, "/orders", tt.body))

			if w.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandler_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		svcErr     error
		wantStatus int
	}{
		{"200 happy", "42", nil, http.StatusOK},
		{"400 invalid id", "abc", nil, http.StatusBadRequest},
		{"400 zero id", "0", nil, http.StatusBadRequest},
		{"404 not found", "42", apperrors.ErrNotFound, http.StatusNotFound},
		{"500 internal", "42", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				getFn: func(_ context.Context, id int64) (*model.Order, error) {
					if tt.svcErr != nil {
						return nil, tt.svcErr
					}
					return &model.Order{ID: id, UserID: 1, Status: model.StatusNew}, nil
				},
			}
			h := transport.NewHandler(svc, discardLogger())

			req := newRequest(http.MethodGet, "/orders/"+tt.id, "")
			req.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()
			h.GetByID(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

func TestHandler_GetByID_DecodesResponseBody(t *testing.T) {
	svc := &fakeService{
		getFn: func(_ context.Context, id int64) (*model.Order, error) {
			return &model.Order{ID: id, UserID: 7, Status: model.StatusNew}, nil
		},
	}
	h := transport.NewHandler(svc, discardLogger())

	req := newRequest(http.MethodGet, "/orders/42", "")
	req.SetPathValue("id", "42")
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	var resp transport.OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != 42 || resp.UserID != 7 || resp.Status != "new" {
		t.Errorf("response: %+v", resp)
	}
}

func TestHandler_CancelOrder(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
	}{
		{"200 happy", nil, http.StatusOK},
		{"404 not found", apperrors.ErrNotFound, http.StatusNotFound},
		{"409 conflict", fmt.Errorf("wrap: %w", apperrors.ErrConflict), http.StatusConflict},
		{"500 unknown", errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				cancelFn: func(_ context.Context, id int64) (*model.Order, error) {
					if tt.svcErr != nil {
						return nil, tt.svcErr
					}
					return &model.Order{ID: id, Status: model.StatusCancelled}, nil
				},
			}
			h := transport.NewHandler(svc, discardLogger())

			req := newRequest(http.MethodPost, "/orders/1/cancel", "")
			req.SetPathValue("id", "1")
			w := httptest.NewRecorder()
			h.CancelOrder(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandler_ListByUser(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantLen    int
	}{
		{"200 with results", "user_id=5", http.StatusOK, 2},
		{"200 empty list", "user_id=99", http.StatusOK, 0},
		{"400 missing user_id", "", http.StatusBadRequest, 0},
		{"400 invalid user_id", "user_id=abc", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				listByUserFn: func(_ context.Context, userID int64) ([]*model.Order, error) {
					if userID == 5 {
						return []*model.Order{{ID: 1}, {ID: 2}}, nil
					}
					return []*model.Order{}, nil
				},
			}
			h := transport.NewHandler(svc, discardLogger())

			req := newRequest(http.MethodGet, "/orders?"+tt.query, "")
			w := httptest.NewRecorder()
			h.ListByUser(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status: want %d, got %d", tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK {
				var resp []transport.OrderResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(resp) != tt.wantLen {
					t.Errorf("len: want %d, got %d", tt.wantLen, len(resp))
				}
			}
		})
	}
}
