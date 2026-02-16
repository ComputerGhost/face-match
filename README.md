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

See the "scripts" folder. Run these from the project root.

Run the AI sidecar first, because that's needed by the other programs. Run the ingest to populate the data. Finally run the server to play around with the AI.
