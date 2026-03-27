.PHONY: proto build-client build-server test lint clean docker-up docker-down

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags "-X main.buildVersion=${VERSION} -X main.buildDate=${DATE}"

PROTO_DIR = internal/pkg/proto
proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
    	--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
       $(PROTO_DIR)/gophkeeper.proto

build-client: proto ## Build client binary
	go build ${LDFLAGS} -o bin/gophkeeper cmd/client/main.go

build-server: proto ## Build server binary
	go build ${LDFLAGS} -o bin/server cmd/server/main.go

docker-up: ## Start development database
	docker-compose up -d gophkeeper-postgres
	@echo "Waiting for gophkeeper-postgres to be ready..."
	@sleep 3

docker-down: ## Stop development database
	docker-compose down

test: proto ## Run tests with coverage
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linter
	golangci-lint run ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

deps: ## Tidy dependencies
	go mod tidy
	go mod verify
