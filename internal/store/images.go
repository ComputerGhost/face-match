package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type Image struct {
	ID         int64
	CategoryID int64
	PersonID   int64
	ImageHash  int64
	Embedding  []float32
}

type ImageStore struct {
	pool *pgxpool.Pool
}

func NewImageStore(pool *pgxpool.Pool) *ImageStore {
	return &ImageStore{pool: pool}
}

func (store *ImageStore) VerifyNoHash(ctx context.Context, hash int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM images WHERE image_hash = $1)`
	err := store.pool.QueryRow(ctx, query, hash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check image existence: %w", err)
	}
	return exists, nil
}

func (store *ImageStore) Insert(ctx context.Context, image *Image) (int64, error) {
	vec := pgvector.NewVector(image.Embedding)
	var id int64
	err := store.pool.QueryRow(ctx, `
		INSERT INTO images (category_id, person_id, image_hash, embedding)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, image.CategoryID, image.PersonID, image.ImageHash, vec).Scan(&id)
	return id, err
}
