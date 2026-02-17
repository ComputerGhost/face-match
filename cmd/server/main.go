package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/face-match/internal/app"
	"github.com/face-match/internal/service"
	"github.com/face-match/internal/store"
)

type Server struct {
	config *app.Config
}

func main() {
	config := &app.Config{
		AIEndpoint:  os.Getenv("AI_ENDPOINT"),
		DatabaseUrl: os.Getenv("DATABASE_URL"),
		DataRoot:    os.Getenv("DATA_ROOT"),
		WebEndpoint: os.Getenv("WEB_ENDPOINT"),
	}
	config.ThumbsPath = filepath.Join(config.DataRoot, "/images/thumbs")

	srv := &Server{
		config: config,
	}

	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("web/static")))
	mux.HandleFunc("/api/categories", srv.handleCategories)
	mux.HandleFunc("/api/search", srv.handleSearch)
	mux.Handle("/thumbs/", http.StripPrefix("/thumbs/", http.FileServer(http.Dir("data/images/thumbs"))))

	server := &http.Server{
		Addr:         config.WebEndpoint,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Listening at %s", config.WebEndpoint)
	log.Fatal(server.ListenAndServe())
}

func (srv *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	pool, err := store.Open(r.Context(), srv.config.DatabaseUrl)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	categoryStore := store.NewCategoryStore(pool)

	categories, err := categoryStore.List(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (srv *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	// Load form data

	categories := r.Form["categories[]"]
	categoryIDs := make([]int64, 0, len(categories))
	for _, category := range categories {
		id, err := strconv.ParseInt(category, 10, 64)
		if err == nil {
			categoryIDs = append(categoryIDs, id)
		}
	}

	file, _, err := r.FormFile("image")
	if err != nil || file == nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
	}
	defer func() { _ = file.Close() }()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
	}

	// Perform the search:

	pool, err := store.Open(r.Context(), srv.config.DatabaseUrl)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	searchService := service.NewSearchService(srv.config, pool)

	results, err := searchService.Search(r.Context(), categoryIDs, buf.Bytes())
	if err != nil {
		log.Printf("search service error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Return the result:
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
