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

	// Returned from reading but not used in writing
	DisplayName       string
	DisambiguationTag string
	CosineDistance    float32
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

func (store *ImageStore) Search(ctx context.Context, categoryIDs []int64, embedding []float32) ([]Image, error) {
	if len(categoryIDs) == 0 {
		return nil, nil
	}

	query := `
		SELECT i.id, i.category_id, i.person_id, p.display_name, p.disambiguation_tag,
			   i.embedding <=> $2 AS cosine_distance
		FROM images i
		JOIN people p ON p.id = i.person_id
		WHERE i.category_id = ANY($1)
		ORDER BY cosine_distance
		LIMIT 10`
	vec := pgvector.NewVector(embedding)
	rows, err := store.pool.Query(ctx, query, categoryIDs, vec)
	if err != nil {
		return nil, fmt.Errorf("search: images select: %s", err)
	}
	defer rows.Close()

	out := make([]Image, 0, 10)
	for rows.Next() {
		var image Image
		if err := rows.Scan(&image.ID, &image.CategoryID, &image.PersonID, &image.DisplayName, &image.DisambiguationTag, &image.CosineDistance); err != nil {
			return nil, fmt.Errorf("search: images scan: %s", err)
		}
		out = append(out, image)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search: images rows: %s", err)
	}
	return out, nil
}
