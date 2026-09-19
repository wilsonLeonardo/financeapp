BACKEND  := backend
FRONTEND := frontend
BINARY   := bin/server

.DEFAULT_GOAL := help

## help: list the available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

# Backend
## build: compile the API into backend/bin/server
build:
	@cd $(BACKEND) && go build -o $(BINARY) ./cmd/server

## run: run the API (needs postgres and redis up)
run:
	@cd $(BACKEND) && go run ./cmd/server

## test: run the Go test suite
test:
	@cd $(BACKEND) && go test ./...

## test-race: run the Go test suite with the race detector
test-race:
	@cd $(BACKEND) && go test -race ./...

## cover: write and open a coverage report
cover:
	@cd $(BACKEND) && go test -coverprofile=coverage.out ./... \
		&& go tool cover -func=coverage.out | tail -1 \
		&& go tool cover -html=coverage.out -o coverage.html
	@echo "report: $(BACKEND)/coverage.html"

## generate: regenerate the mocks in internal/testutils/mocks
generate:
	@cd $(BACKEND) && go generate ./...

## docs: regenerate the OpenAPI spec from the handler annotations
docs:
	@cd $(BACKEND) && go tool swag init -g cmd/server/main.go -o docs \
		--parseDependency --parseInternal --quiet
	@echo "spec written to $(BACKEND)/docs; UI at http://localhost:8080/swagger/index.html"

## fmt: format the Go sources
fmt:
	@cd $(BACKEND) && gofmt -w .

## vet: run go vet
vet:
	@cd $(BACKEND) && go vet ./...

## tidy: prune and sync go.mod and go.sum
tidy:
	@cd $(BACKEND) && go mod tidy

# Frontend
## web-install: install the frontend dependencies
web-install:
	@cd $(FRONTEND) && npm ci

## web-dev: start the Vite dev server
web-dev:
	@cd $(FRONTEND) && npm run dev

## web-test: run the frontend test suite
web-test:
	@cd $(FRONTEND) && npm test

## web-build: build the production bundle
web-build:
	@cd $(FRONTEND) && npm run build

# Stack
## up: start the whole stack with docker compose
up:
	@docker compose up --build -d

## down: stop the stack
down:
	@docker compose down

## logs: follow the stack logs
logs:
	@docker compose logs -f

## db: open a psql shell on the running database
db:
	@docker compose exec postgres psql -U $${POSTGRES_USER:-financeapp} -d $${POSTGRES_DB:-financeapp}

# Everything
## check: what CI runs — format check, vet and both test suites
check: vet test web-test
	@cd $(BACKEND) && test -z "$$(gofmt -l .)" || (echo "these files need gofmt:" && gofmt -l . && exit 1)
	@echo "all checks passed"

## clean: remove build and coverage artefacts
clean:
	@rm -rf $(BACKEND)/bin $(BACKEND)/coverage.out $(BACKEND)/coverage.html $(FRONTEND)/dist

.PHONY: help build run test test-race cover generate docs fmt vet tidy \
        web-install web-dev web-test web-build up down logs db check clean
