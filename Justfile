set dotenv-load

default: test

alias tbe := test-be
alias tfe := test-fe

# Run all tests and checks (backend + frontend)
test args='':
    cd backend && go vet ./... && gotestsum ./... -- {{ args }}
    cd frontend && npm run check && npm test

# Run backend tests and checks
test-be args='':
    cd backend && go vet ./... && gotestsum ./... -- {{ args }}

# Run frontend tests and checks (npm run check syncs svelte-kit first)
test-fe:
    cd frontend && npm run check && npm test

# Lint backend (golangci-lint) and frontend (eslint + prettier --check)
lint:
    cd backend && golangci-lint run
    cd frontend && npm run lint

# Auto-fix lint findings and formatting
lint-fix:
    cd backend && golangci-lint run --fix
    cd frontend && npm run lint:fix && npm run format

# Run regular tests with certain args that gotestsum seems to ignore
gotest args='':
    cd backend && go test ./... {{ args }}

# Run tests with coverage report
cover:
    cd backend && gotestsum -- -coverprofile=coverage.out ./... && go tool cover -func=coverage.out

# Run frontend tests with coverage report
cover-fe:
    cd frontend && npm run test:coverage

# Build the backend binary
build:
    cd backend && go build -o bin/flockdeck ./cmd/server

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
    migrate create -ext sql -dir backend/migrations -seq -digits 3 {{ name }}

# Run the ingestion script (usage: just ingest US-OR)
ingest *args:
    cd backend && go run ./cmd/ingest {{ args }}

# Seed a local user with simulated study history (usage: just seed you@example.com --days 30)
seed email *args:
    cd backend && go run ./cmd/seed {{ args }} {{ email }}

# Start the backend server (hot-reload via air)
be:
    cd backend && air

# Start the frontend dev server
fe:
    cd frontend && npm run dev -- --host

# Start backend + frontend together (Ctrl+C stops both)
run:
    #!/usr/bin/env bash
    just be &
    trap "kill %1" EXIT
    just fe

# Install all the required tools
install-tools:
    brew install sqlc golangci-lint
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    go install github.com/air-verse/air@latest
    go install gotest.tools/gotestsum@latest
