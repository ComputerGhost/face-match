#!/usr/bin/env bash
set -e

# Load env vars
set -a
source .env
set +a

cd ai-sidecar

if [ ! -d ".venv" ]; then
  echo "Virtualenv not found. Run setup first."
  exit 1
fi

source .venv/bin/activate

uvicorn app:APP --host 0.0.0.0 --port 8081 --reload
