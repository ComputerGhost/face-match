package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Person struct {
	ID                int64
	CategoryId        int64
	Category          string // Used for select, not insert
	DisplayName       string
	DisambiguationTag string
	IsHidden          bool
}

type PersonStore struct {
	pool *pgxpool.Pool
}

func NewPersonStore(pool *pgxpool.Pool) *PersonStore {
	return &PersonStore{pool: pool}
}

func (store *PersonStore) Purge(ctx context.Context, personID int64) error {
	if _, err := store.pool.Exec(ctx, `DELETE FROM images WHERE person_id = $1`, personID); err != nil {
		return fmt.Errorf("store: person purge: %w", err)
	}
	if _, err := store.pool.Exec(ctx, `DELETE FROM people WHERE id = $1`, personID); err != nil {
		return fmt.Errorf("store: person purge: %w", err)
	}
	return nil
}

func (store *PersonStore) Upsert(ctx context.Context, person *Person) (int64, error) {
	var id int64
	err := store.pool.QueryRow(ctx, `
		INSERT INTO people (category_id, display_name, disambiguation_tag, is_hidden)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (category_id, display_name, disambiguation_tag)
		DO UPDATE SET display_name = excluded.display_name
		RETURNING id
	`, person.CategoryId, person.DisplayName, person.DisambiguationTag, person.IsHidden).Scan(&id)
	return id, err
}

func (store *PersonStore) Search(ctx context.Context, query string) ([]Person, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT p.id, c.display_name category, p.display_name, p.disambiguation_tag, p.is_hidden
		FROM people p
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE p.display_name LIKE '%' || $1 || '%'
		LIMIT 10
	`, query)
	if err != nil {
		return nil, fmt.Errorf("store: person search: %w", err)
	}
	defer rows.Close()

	out := make([]Person, 0, 16)
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Category, &p.DisplayName, &p.DisambiguationTag, &p.IsHidden); err != nil {
			return nil, fmt.Errorf("store: person scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: person rows: %w", err)
	}
	return out, nil
}

func (store *PersonStore) SetHidden(cxt context.Context, id int64, hide bool) error {
	_, err := store.pool.Exec(cxt, `
		UPDATE people
		SET is_hidden = $1
		WHERE id = $2
	`, hide, id)
	if err != nil {
		return fmt.Errorf("store: person hide: %w", err)
	}
	return nil
}
