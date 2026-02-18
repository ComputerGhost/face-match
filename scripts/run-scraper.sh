#!/usr/bin/env bash
set -e

if [ $# -lt 1 ]; then
  echo "Usage: $0 <scraper-name> [args...]"
  exit 1
fi

SCRAPER="$1"
shift

# Load environment variables
set -a
source .env
set +a

BIN="./bin/$SCRAPER"

if [ ! -x "$BIN" ]; then
  echo "Scraper not found or not executable: $BIN"
  echo "Did you run scripts/build.sh?"
  exit 1
fi

exec "$BIN" "$@"
