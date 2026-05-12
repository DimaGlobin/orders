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

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
	"github.com/dimaglobin/order-service/internal/transport"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeService struct {
	createFn func(ctx context.Context, order *model.Order) error
	getFn    func(ctx context.Context, id uuid.UUID) (*model.Order, error)
	cancelFn func(ctx context.Context, id uuid.UUID) (*model.Order, error)
	listFn   func(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}

func (f *fakeService) CreateOrder(ctx context.Context, o *model.Order) error {
	return f.createFn(ctx, o)
}
func (f *fakeService) GetOrder(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	return f.getFn(ctx, id)
}
func (f *fakeService) CancelOrder(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	return f.cancelFn(ctx, id)
}
func (f *fakeService) ListOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	return f.listFn(ctx, userID)
}

func newRequest(method, target, body string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func TestHandler_Create(t *testing.T) {
	uid := uuid.New().String()
	pid := uuid.New().String()
	validBody := fmt.Sprintf(`{"user_id":%q,"items":[{"product_id":%q,"quantity":2,"price":500}]}`, uid, pid)

	tests := []struct {
		name       string
		body       string
		svcErr     error
		wantStatus int
	}{
		{
			name:       "201 happy path",
			body:       validBody,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "400 invalid JSON",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: empty items",
			body:       fmt.Sprintf(`{"user_id":%q,"items":[]}`, uid),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: nil user_id",
			body:       fmt.Sprintf(`{"user_id":%q,"items":[{"product_id":%q,"quantity":1,"price":1}]}`, uuid.Nil, pid),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "400 validation: zero price",
			body:       fmt.Sprintf(`{"user_id":%q,"items":[{"product_id":%q,"quantity":1,"price":0}]}`, uid, pid),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "500 service error",
			body:       validBody,
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
	validID := uuid.New().String()

	tests := []struct {
		name       string
		id         string
		svcErr     error
		wantStatus int
	}{
		{"200 happy", validID, nil, http.StatusOK},
		{"400 invalid id", "abc", nil, http.StatusBadRequest},
		{"400 nil id", uuid.Nil.String(), nil, http.StatusBadRequest},
		{"404 not found", validID, apperrors.ErrNotFound, http.StatusNotFound},
		{"500 internal", validID, errors.New("boom"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				getFn: func(_ context.Context, id uuid.UUID) (*model.Order, error) {
					if tt.svcErr != nil {
						return nil, tt.svcErr
					}
					return &model.Order{ID: id, UserID: uuid.New(), Status: model.StatusNew}, nil
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
	orderID := uuid.New()
	userID := uuid.New()
	svc := &fakeService{
		getFn: func(_ context.Context, id uuid.UUID) (*model.Order, error) {
			return &model.Order{ID: id, UserID: userID, Status: model.StatusNew}, nil
		},
	}
	h := transport.NewHandler(svc, discardLogger())

	req := newRequest(http.MethodGet, "/orders/"+orderID.String(), "")
	req.SetPathValue("id", orderID.String())
	w := httptest.NewRecorder()
	h.GetByID(w, req)

	var resp transport.OrderResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ID != orderID || resp.UserID != userID || resp.Status != "new" {
		t.Errorf("response: %+v", resp)
	}
}

func TestHandler_CancelOrder(t *testing.T) {
	orderID := uuid.New().String()

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
				cancelFn: func(_ context.Context, id uuid.UUID) (*model.Order, error) {
					if tt.svcErr != nil {
						return nil, tt.svcErr
					}
					return &model.Order{ID: id, Status: model.StatusCancelled}, nil
				},
			}
			h := transport.NewHandler(svc, discardLogger())

			req := newRequest(http.MethodPost, "/orders/"+orderID+"/cancel", "")
			req.SetPathValue("id", orderID)
			w := httptest.NewRecorder()
			h.CancelOrder(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: want %d, got %d, body=%s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestHandler_ListOrders(t *testing.T) {
	withResults := uuid.New()

	tests := []struct {
		name       string
		query      string
		wantStatus int
		wantLen    int
	}{
		{"200 with results", "user_id=" + withResults.String(), http.StatusOK, 2},
		{"200 empty list", "user_id=" + uuid.New().String(), http.StatusOK, 0},
		{"400 missing user_id", "", http.StatusBadRequest, 0},
		{"400 invalid user_id", "user_id=abc", http.StatusBadRequest, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeService{
				listFn: func(_ context.Context, userID uuid.UUID) ([]*model.Order, error) {
					if userID == withResults {
						return []*model.Order{{ID: uuid.New()}, {ID: uuid.New()}}, nil
					}
					return []*model.Order{}, nil
				},
			}
			h := transport.NewHandler(svc, discardLogger())

			req := newRequest(http.MethodGet, "/orders?"+tt.query, "")
			w := httptest.NewRecorder()
			h.ListOrders(w, req)

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
