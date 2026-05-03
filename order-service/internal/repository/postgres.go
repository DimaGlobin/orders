package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dimaglobin/order-service/internal/apperrors"
	"github.com/dimaglobin/order-service/internal/model"
)

const outboxTopic = "orders"

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (r *Postgres) Create(ctx context.Context, order *model.Order) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`INSERT INTO orders (user_id, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			order.UserID, order.Status, order.CreatedAt, order.UpdatedAt,
		).Scan(&order.ID)
		if err != nil {
			return fmt.Errorf("insert order: %w", err)
		}

		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			err := tx.QueryRow(ctx,
				`INSERT INTO order_items (order_id, product_id, quantity, price)
				 VALUES ($1, $2, $3, $4)
				 RETURNING id, created_at`,
				order.Items[i].OrderID,
				order.Items[i].ProductID,
				order.Items[i].Quantity,
				order.Items[i].Price,
			).Scan(&order.Items[i].ID, &order.Items[i].CreatedAt)
			if err != nil {
				return fmt.Errorf("insert item: %w", err)
			}
		}

		return writeOutbox(ctx, tx, order, model.EventOrderCreated)
	})
}

func (r *Postgres) GetByID(ctx context.Context, id int64) (*model.Order, error) {
	order, err := selectOrder(ctx, r.pool, id)
	if err != nil {
		return nil, err
	}
	items, err := selectItems(ctx, r.pool, id)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return order, nil
}

func (r *Postgres) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (*model.Order, error) {
	var result *model.Order
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		var o model.Order
		err := tx.QueryRow(ctx,
			`UPDATE orders SET status = $1, updated_at = NOW()
			 WHERE id = $2
			 RETURNING id, user_id, status, created_at, updated_at`,
			status, id,
		).Scan(&o.ID, &o.UserID, &o.Status, &o.CreatedAt, &o.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update order: %w", err)
		}

		items, err := selectItemsTx(ctx, tx, id)
		if err != nil {
			return err
		}
		o.Items = items

		if status == model.StatusCancelled {
			if err := writeOutbox(ctx, tx, &o, model.EventOrderCancelled); err != nil {
				return err
			}
		}

		result = &o
		return nil
	})
	return result, err
}

func (r *Postgres) ListByUserID(ctx context.Context, userID int64) ([]*model.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, status, created_at, updated_at
		 FROM orders WHERE user_id = $1 ORDER BY id`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("select orders: %w", err)
	}
	defer rows.Close()

	orders := make([]*model.Order, 0)
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, &o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, o := range orders {
		items, err := selectItems(ctx, r.pool, o.ID)
		if err != nil {
			return nil, err
		}
		o.Items = items
	}
	return orders, nil
}

func (r *Postgres) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func selectOrder(ctx context.Context, pool *pgxpool.Pool, id int64) (*model.Order, error) {
	var o model.Order
	err := pool.QueryRow(ctx,
		`SELECT id, user_id, status, created_at, updated_at FROM orders WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.UserID, &o.Status, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select order: %w", err)
	}
	return &o, nil
}

func selectItems(ctx context.Context, pool *pgxpool.Pool, orderID int64) ([]model.OrderItem, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, order_id, product_id, quantity, price, created_at
		 FROM order_items WHERE order_id = $1 ORDER BY id`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("select items: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

func selectItemsTx(ctx context.Context, tx pgx.Tx, orderID int64) ([]model.OrderItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id, order_id, product_id, quantity, price, created_at
		 FROM order_items WHERE order_id = $1 ORDER BY id`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("select items: %w", err)
	}
	defer rows.Close()
	return scanItems(rows)
}

func scanItems(rows pgx.Rows) ([]model.OrderItem, error) {
	items := make([]model.OrderItem, 0)
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.Price, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func writeOutbox(ctx context.Context, tx pgx.Tx, order *model.Order, eventType model.EventType) error {
	items := make([]model.OrderEventItem, len(order.Items))
	for i, item := range order.Items {
		items[i] = model.OrderEventItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}
	evt := model.OrderEvent{
		Type:       eventType,
		Version:    1,
		OrderID:    order.ID,
		UserID:     order.UserID,
		Status:     string(order.Status),
		Items:      items,
		OccurredAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO outbox (topic, key, payload) VALUES ($1, $2, $3)`,
		outboxTopic, strconv.FormatInt(order.ID, 10), payload,
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}
	return nil
}
