package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/face-match/internal/ai"
	"github.com/face-match/internal/app"
	"github.com/face-match/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchService struct {
	config        *app.Config
	categoryStore *store.CategoryStore
	imageStore    *store.ImageStore
	personStore   *store.PersonStore
}

type SearchResult struct {
	ID                int64
	CategoryID        int64
	PersonID          int64
	DisplayName       string
	DisambiguationTag string
	SimilarityScore   float32
}

func NewSearchService(config *app.Config, pool *pgxpool.Pool) *SearchService {
	return &SearchService{
		config:        config,
		categoryStore: store.NewCategoryStore(pool),
		imageStore:    store.NewImageStore(pool),
		personStore:   store.NewPersonStore(pool),
	}
}

func (s *SearchService) Search(ctx context.Context, categoryIDs []int64, imageBytes []byte) ([]SearchResult, error) {
	embedding, err := ai.FetchEmbedding(ctx, s.config.AIEndpoint, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("fetch embedding: %s", err)
	}

	images, err := s.imageStore.Search(ctx, categoryIDs, embedding)
	if err != nil {
		return nil, fmt.Errorf("search: %s", err)
	}

	if images == nil || len(images) == 0 {
		return []SearchResult{}, nil
	}

	bestByPerson := make(map[int64]SearchResult, len(images))
	for _, img := range images {
		score := float32(1.0) - img.CosineDistance
		if score < 0 {
			score = 0
		}

		r := SearchResult{
			ID:                img.ID,
			CategoryID:        img.CategoryID,
			PersonID:          img.PersonID,
			DisplayName:       img.DisplayName,
			DisambiguationTag: img.DisambiguationTag,
			SimilarityScore:   score,
		}

		prev, ok := bestByPerson[img.PersonID]
		if !ok || r.SimilarityScore > prev.SimilarityScore {
			bestByPerson[img.PersonID] = r
		}
	}

	out := make([]SearchResult, 0, len(bestByPerson))
	for _, r := range bestByPerson {
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].SimilarityScore == out[j].SimilarityScore {
			return out[i].PersonID < out[j].PersonID
		}
		return out[i].SimilarityScore > out[j].SimilarityScore
	})

	return out, nil
}
