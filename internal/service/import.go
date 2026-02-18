package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/face-match/internal/ai"
	"github.com/face-match/internal/app"
	"github.com/face-match/internal/hash"
	"github.com/face-match/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ImportService struct {
	config        *app.Config
	categoryStore *store.CategoryStore
	imageStore    *store.ImageStore
	personStore   *store.PersonStore
}

func NewImportService(config *app.Config, pool *pgxpool.Pool) *ImportService {
	return &ImportService{
		config:        config,
		categoryStore: store.NewCategoryStore(pool),
		imageStore:    store.NewImageStore(pool),
		personStore:   store.NewPersonStore(pool),
	}
}

func (service *ImportService) Import(ctx context.Context, category string) error {
	categoryId, err := service.categoryStore.FetchId(ctx, category)
	if err != nil {
		return fmt.Errorf("service: fetch category id: %w", err)
	}

	files, err := fetchInputFiles(service.config)
	if err != nil {
		return fmt.Errorf("service: fetch files: %w", err)
	}
	log.Printf("Importing %d file(s) from %s into category id %d",
		len(files), service.config.InputPath, categoryId)

	for _, f := range files {
		if err := processFile(ctx, service, categoryId, f); err != nil {
			log.Printf("Error processing file %s: %v", f, err)
		}
	}
	return nil
}

func fetchInputFiles(config *app.Config) ([]string, error) {
	files, err := os.ReadDir(config.InputPath)
	if err != nil {
		return nil, err
	}

	var imageFiles []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(f.Name()))
		switch extension {
		case ".jpg", ".jpeg", ".png", ".webp":
			imageFiles = append(imageFiles, f.Name())
		default:
		}
	}

	// Sorting makes repeat runs predictable
	slices.Sort(imageFiles)

	return imageFiles, nil
}

func processFile(ctx context.Context, service *ImportService, categoryId int64, filename string) error {
	name, tag, err := parseInboxFilename(filename)
	if err != nil {
		return fmt.Errorf("service: parse inbox filename: %w", err)
	}

	imageBytes, err := os.ReadFile(filepath.Join(service.config.InputPath, filename))
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	embedding, err := ai.FetchEmbedding(ctx, service.config.AIEndpoint, imageBytes)
	if err != nil {
		return fmt.Errorf("fetch embedding: %w", err)
	}

	// Save person to database:

	person := store.Person{
		CategoryId:        categoryId,
		DisplayName:       name,
		DisambiguationTag: tag,
		IsHidden:          false,
	}
	personID, err := service.personStore.Upsert(ctx, &person)
	if err != nil {
		return fmt.Errorf("upsert person: %w", err)
	}

	// Save image to database:

	imageHash, err := hash.DHash64(imageBytes)
	if err != nil {
		return fmt.Errorf("hash image: %w", err)
	}
	exists, err := service.imageStore.VerifyNoHash(ctx, imageHash)
	if err != nil {
		return fmt.Errorf("fetch id by hash: %w", err)
	}
	if exists {
		return fmt.Errorf("image already processed")
	}

	image := store.Image{
		CategoryID: categoryId,
		PersonID:   personID,
		ImageHash:  imageHash,
		Embedding:  embedding,
	}
	imageID, err := service.imageStore.Insert(ctx, &image)
	if err != nil {
		return fmt.Errorf("insert image: %w", err)
	}

	// Move/create files:
	if err := os.Rename(filepath.Join(service.config.InputPath, filename), filepath.Join(service.config.FinishedPath, filename)); err != nil {
		return fmt.Errorf("move to ok: %w", err)
	}

	log.Printf("OK: %s => person_id=%d image_id=%d", filename, personID, imageID)
	return nil
}

// Expected formats:
// - "Bob Marley.jpg" -> name = "Bob Marley", tag = ""
// - "Park Jeonghwa [exid].jpg" -> name="Park Jeonghwa", tag="exid"
func parseInboxFilename(filename string) (string, string, error) {
	// Trim extension
	extension := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, extension)

	// Trim copy count
	extension = filepath.Ext(base)
	if _, err := strconv.ParseFloat(extension, 64); err == nil {
		base = strings.TrimSuffix(base, extension)
	}

	var tag string = ""
	var name string = ""

	// Look for disambiguation tag
	tagStart := strings.LastIndex(base, "[")
	tagEnd := strings.LastIndex(base, "]")
	if tagStart != -1 && tagEnd == len(base)-1 && tagStart < tagEnd {
		tag = strings.TrimSpace(base[tagStart+1 : tagEnd])
		name = strings.TrimSpace(base[:tagStart])
	} else {
		name = strings.TrimSpace(base)
	}

	if name == "" {
		return "", "", fmt.Errorf("invalid filename (empty name): %s", filename)
	}
	return name, tag, nil
}
