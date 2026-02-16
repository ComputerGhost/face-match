package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, databaseUrl string) (*pgxpool.Pool, error) {
	if databaseUrl == "" {
		return nil, errors.New("store: DatabaseUrl is required")
	}

	poolConfig, err := pgxpool.ParseConfig(databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("store: parse db url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("store: create pool: %w", err)
	}

	healthContext, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(healthContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}

	return pool, nil
}

func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	transaction, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if err := fn(transaction); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit transaction: %w", err)
	}
	return nil
}
