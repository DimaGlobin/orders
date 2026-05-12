package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
	"github.com/dimaglobin/order-service/internal/service"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeRepo struct {
	createFn       func(ctx context.Context, order *model.Order) error
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*model.Order, error)
	updateStatusFn func(ctx context.Context, id uuid.UUID, status model.OrderStatus) (*model.Order, error)
	listOrdersFn   func(ctx context.Context, userID uuid.UUID) ([]*model.Order, error)
}

func (f *fakeRepo) Create(ctx context.Context, order *model.Order) error {
	return f.createFn(ctx, order)
}
func (f *fakeRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) (*model.Order, error) {
	return f.updateStatusFn(ctx, id, status)
}
func (f *fakeRepo) ListOrders(ctx context.Context, userID uuid.UUID) ([]*model.Order, error) {
	return f.listOrdersFn(ctx, userID)
}

func TestService_CreateOrder_SetsStatusAndTimestamps(t *testing.T) {
	var captured *model.Order
	repo := &fakeRepo{
		createFn: func(_ context.Context, order *model.Order) error {
			captured = order
			return nil
		},
	}
	svc := service.NewService(repo, discardLogger())

	order := &model.Order{UserID: uuid.New()}
	if err := svc.CreateOrder(context.Background(), order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if captured.Status != model.StatusNew {
		t.Errorf("status: want %q, got %q", model.StatusNew, captured.Status)
	}
	if captured.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if !captured.UpdatedAt.Equal(captured.CreatedAt) {
		t.Error("UpdatedAt should equal CreatedAt on create")
	}
}

func TestService_CreateOrder_PropagatesRepoError(t *testing.T) {
	wantErr := errors.New("db down")
	repo := &fakeRepo{
		createFn: func(_ context.Context, _ *model.Order) error { return wantErr },
	}
	svc := service.NewService(repo, discardLogger())

	err := svc.CreateOrder(context.Background(), &model.Order{UserID: uuid.New()})
	if !errors.Is(err, wantErr) {
		t.Errorf("want error %v, got %v", wantErr, err)
	}
}

func TestService_CancelOrder(t *testing.T) {
	tests := []struct {
		name        string
		current     model.OrderStatus
		getErr      error
		wantErrIs   error
		wantUpdated bool
	}{
		{"new → cancelled", model.StatusNew, nil, nil, true},
		{"already cancelled", model.StatusCancelled, nil, apperrors.ErrConflict, false},
		{"already confirmed", model.StatusConfirmed, nil, apperrors.ErrConflict, false},
		{"not found", "", apperrors.ErrNotFound, apperrors.ErrNotFound, false},
	}

	orderID := uuid.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false
			repo := &fakeRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Order, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return &model.Order{ID: id, Status: tt.current}, nil
				},
				updateStatusFn: func(_ context.Context, id uuid.UUID, status model.OrderStatus) (*model.Order, error) {
					updateCalled = true
					return &model.Order{ID: id, Status: status}, nil
				},
			}
			svc := service.NewService(repo, discardLogger())

			_, err := svc.CancelOrder(context.Background(), orderID)

			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("want errors.Is %v, got %v", tt.wantErrIs, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if updateCalled != tt.wantUpdated {
				t.Errorf("updateStatus called: want %v, got %v", tt.wantUpdated, updateCalled)
			}
		})
	}
}

func TestService_GetOrder_DelegatesToRepo(t *testing.T) {
	wantID := uuid.New()
	want := &model.Order{ID: wantID, UserID: uuid.New(), Status: model.StatusNew}
	repo := &fakeRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*model.Order, error) {
			if id != wantID {
				t.Errorf("expected id %s, got %s", wantID, id)
			}
			return want, nil
		},
	}
	svc := service.NewService(repo, discardLogger())

	got, err := svc.GetOrder(context.Background(), wantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected same pointer back from repo")
	}
}

func TestService_ListOrders_DelegatesToRepo(t *testing.T) {
	userID := uuid.New()
	want := []*model.Order{{ID: uuid.New()}, {ID: uuid.New()}}
	repo := &fakeRepo{
		listOrdersFn: func(_ context.Context, gotUser uuid.UUID) ([]*model.Order, error) {
			if gotUser != userID {
				t.Errorf("expected userID %s, got %s", userID, gotUser)
			}
			return want, nil
		},
	}
	svc := service.NewService(repo, discardLogger())

	got, err := svc.ListOrders(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 orders, got %d", len(got))
	}
}
