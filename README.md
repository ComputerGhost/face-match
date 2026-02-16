# face-match
Simple face search website and tools

I am learning the Go language, and this is my "Hello, world!"

## Setup

 1. Create a new PostgreSQL dataabse
 2. Set the environment variables

### Environment variables

Used by ingest, server, and similar Go tools
 * AI_ENDPOINT
 * DATABASE_URL
 * DATA_ROOT

Used by the Python AI
 * MODEL_DIR - default: ./models
 * ORT_PROVIDERS  - default: CPUExecutionProvider
 * GPU_ID         - default: -1

## Usage

 1. Run the ai-sidecar and note the URL it's running at.
