set dotenv-load

default: test

# Run all backend tests
test args='':
    cd backend && go test {{ args }} ./...

# Start the backend server
run:
    cd backend && go run ./cmd/server

# Build the backend binary
build:
    cd backend && go build -o bin/lifer ./cmd/server

# Run migrations up
migrate-up:
    migrate -path backend/migrations -database "$DATABASE_URL" up

# Roll back one migration
migrate-down:
    migrate -path backend/migrations -database "$DATABASE_URL" down 1

# Regenerate sqlc types after schema changes
generate:
    cd backend && sqlc generate

# Create a new migration (usage: just migration name=add_something)
migration name:
    migrate create -ext sql -dir backend/migrations -seq {{ name }}

# Start the frontend dev server
frontend:
    cd frontend && npm run dev

# Run the ingestion script (usage: just ingest US-OR)
ingest *args:
    cd backend && go run ./cmd/ingest {{ args }}

# Install all the required tools
install-tools:
    brew install sqlc
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
