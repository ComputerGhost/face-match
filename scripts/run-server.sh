#!/usr/bin/env bash
set -e

set -a
source .env
set +a

# Ensure AI sidecar is running
if ! curl -sf http://localhost:8081/healthz > /dev/null; then
  echo "AI sidecar is not running on :8081"
  echo "Run: scripts/run-ai-sidecar.sh"
  exit 1
fi

./bin/server "$@"
