.PHONY: help up down logs restart shell test lint fmt build generate mocks test-short

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

up:
	@cd ../docker && docker compose up --build

down:
	@cd ../docker && docker compose down

logs:
	@cd ../docker && docker compose logs -f bank-api

restart:
	@cd ../docker && docker compose restart bank-api

shell:
	@cd ../docker && docker compose exec bank-api sh

test:
	@cd ../docker && docker compose exec bank-api go test -v ./...

test-cover:
	@cd ../docker && docker compose exec bank-api go test -v -cover ./...

lint:
	@cd ../docker && docker compose exec bank-api golangci-lint run

fmt:
	@cd ../docker && docker compose exec bank-api gofmt -w .

build:
	@cd ../docker && docker compose exec bank-api go build -o bin/bank ./cmd/bank

generate: ## Generate API code from OpenAPI spec
	@cd api/cfg && go tool oapi-codegen -config dtos.yaml ../openapi.yaml
	@cd api/cfg && go tool oapi-codegen -config server.yaml ../openapi.yaml
	@cd api/cfg && go tool oapi-codegen -config spec.yaml ../openapi.yaml

mocks: ## Generate mocks for testing
	@cd ../docker && docker compose exec -e MOCKERY_VERSION= bank-api mockery

test-short: ## Run unit tests only (no database)
	@cd ../docker && docker compose exec bank-api go test -v -short ./...

.DEFAULT_GOAL := help
