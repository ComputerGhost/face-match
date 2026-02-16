#!/usr/bin/env bash
set -e

echo "Building ingest..."
go build -o bin/ingest ./cmd/ingest

echo "Building server..."
go build -o bin/server ./cmd/server
