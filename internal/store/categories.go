package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Category struct {
	ID          int64
	DisplayName string
}

type CategoryStore struct {
	pool *pgxpool.Pool
}

func NewCategoryStore(pool *pgxpool.Pool) *CategoryStore {
	return &CategoryStore{pool: pool}
}

func (store *CategoryStore) FetchId(ctx context.Context, category string) (int64, error) {
	var ID int64
	row := store.pool.QueryRow(ctx, "SELECT id FROM categories WHERE display_name = $1", category)
	err := row.Scan(&ID)
	if err != nil {
		return 0, err
	}
	return ID, nil
}

func (store *CategoryStore) List(ctx context.Context) ([]Category, error) {
	rows, err := store.pool.Query(ctx, `SELECT id, display_name FROM categories ORDER BY display_name ASC`)
	if err != nil {
		return nil, fmt.Errorf("store: categories list: %w", err)
	}
	defer rows.Close()

	out := make([]Category, 0, 16)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.DisplayName); err != nil {
			return nil, fmt.Errorf("store: categories scan: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: categories rows: %w", err)
	}
	return out, nil
}
