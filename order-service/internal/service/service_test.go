package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
	"github.com/dimaglobin/order-service/internal/service"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type fakeRepo struct {
	createFn       func(ctx context.Context, order *model.Order) error
	getByIDFn      func(ctx context.Context, id int64) (*model.Order, error)
	updateStatusFn func(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error)
	listByUserIDFn func(ctx context.Context, userID int64) ([]*model.Order, error)
}

func (f *fakeRepo) Create(ctx context.Context, order *model.Order) error {
	return f.createFn(ctx, order)
}
func (f *fakeRepo) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	return f.getByIDFn(ctx, id)
}
func (f *fakeRepo) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
	return f.updateStatusFn(ctx, id, status)
}
func (f *fakeRepo) ListByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	return f.listByUserIDFn(ctx, userID)
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

	order := &model.Order{UserID: 7}
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

	err := svc.CreateOrder(context.Background(), &model.Order{UserID: 1})
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCalled := false
			repo := &fakeRepo{
				getByIDFn: func(_ context.Context, id int64) (*model.Order, error) {
					if tt.getErr != nil {
						return nil, tt.getErr
					}
					return &model.Order{ID: id, Status: tt.current}, nil
				},
				updateStatusFn: func(_ context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
					updateCalled = true
					return &model.Order{ID: id, Status: status}, nil
				},
			}
			svc := service.NewService(repo, discardLogger())

			_, err := svc.CancelOrder(context.Background(), 1)

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
	want := &model.Order{ID: 42, UserID: 1, Status: model.StatusNew}
	repo := &fakeRepo{
		getByIDFn: func(_ context.Context, id int64) (*model.Order, error) {
			if id != 42 {
				t.Errorf("expected id 42, got %d", id)
			}
			return want, nil
		},
	}
	svc := service.NewService(repo, discardLogger())

	got, err := svc.GetOrder(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("expected same pointer back from repo")
	}
}

func TestService_ListByUser_DelegatesToRepo(t *testing.T) {
	want := []*model.Order{{ID: 1}, {ID: 2}}
	repo := &fakeRepo{
		listByUserIDFn: func(_ context.Context, userID int64) ([]*model.Order, error) {
			if userID != 5 {
				t.Errorf("expected userID 5, got %d", userID)
			}
			return want, nil
		},
	}
	svc := service.NewService(repo, discardLogger())

	got, err := svc.ListByUser(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 orders, got %d", len(got))
	}
}
