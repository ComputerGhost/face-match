#!/usr/bin/env bash
set -e

echo "Building ingest..."
go build -o bin/ingest ./cmd/ingest

echo "Building server..."
go build -o bin/server ./cmd/server

echo "Building scrapers..."

# Find all scraper main.go files
find cmd/scrape -name main.go | while read -r main; do
  dir="$(dirname "$main")"
  name="$(basename "$dir")"

  echo "  - $name"
  go build -o "bin/$name" "./$dir"
done

echo "Build complete."