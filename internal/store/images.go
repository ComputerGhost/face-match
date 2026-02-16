package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Image struct {
	ID         int64
	CategoryID int64
	PersonID   int64
	ImageHash  uint64
	Embedding  []float32
}

type ImageStore struct {
	pool *pgxpool.Pool
}

func NewImageStore(pool *pgxpool.Pool) *ImageStore {
	return &ImageStore{pool: pool}
}

func (store *ImageStore) VerifyNoHash(ctx context.Context, hash uint64) (bool, error) {
	var ID uint64
	row := store.pool.QueryRow(ctx, `SELECT * FROM images WHERE hash = $1`, hash)
	err := row.Scan(&ID)
	if err != nil {
		return false, fmt.Errorf("store: fetch image id by hash: %w", err)
	}
	if ID != 0 {
		return false, fmt.Errorf("store: hash collision with image id %d", ID)
	}
	return true, nil
}

func (store *ImageStore) Insert(ctx context.Context, image *Image) (int64, error) {
	var id int64
	err := store.pool.QueryRow(ctx, `
		INSERT INTO images (category_id, person_id, image_hash, embedding)
		VALUES ($1, $2, $3, $5)
		RETURNING id
	`, image.CategoryID, image.PersonID, image.ImageHash, image.Embedding).Scan(&id)
	return id, err
}
