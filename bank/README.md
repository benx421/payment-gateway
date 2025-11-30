# Mock Bank Service

This is a simple generic banking API that processes card transactions using a double-entry accounting model with accounts and transaction ledger. It supports authorization holds, captures, voids, and refunds with idempotency keys.

## Quick Start

```bash
make up
```

This starts:

- PostgreSQL 16 with persistent data
- Bank API with **hot reload** (auto-restarts on code changes)
- Debug logging enabled
- API available at <http://localhost:8787>

## Prerequisites

- Docker & Docker Compose

Everything else runs inside containers.

## Database Migrations

Migrations run automatically when you `make up`. The database schema includes:

- `accounts`: Customer accounts with card details
- `transactions`: Transaction ledger (auth holds, captures, voids, refunds)
- `idempotency_keys`: Request deduplication

## Available Make Commands

```bash
make help         # Show all available commands

# Development
make up           # Start application
make down         # Stop application
make logs         # View logs (streaming)
make restart      # Restart API service
make shell        # Open shell in container

# Testing & Quality (runs inside container)
make test         # Run all tests (unit + integration)
make test-short   # Run tests (faster, no race detector)
make lint         # Run golangci-lint
make fmt          # Format code with gofmt
make build        # Build binary
make mocks        # Regenerate mocks with mockery
```

## Database Configuration

The application uses environment variables for database configuration:

```bash
DB_HOST=localhost       # Database host (default: localhost)
DB_PORT=5432           # Database port (default: 5432)
DB_USER=postgres       # Database user (default: postgres)
DB_PASSWORD=postgres   # Database password (default: postgres)
DB_NAME=mockbank      # Database name (default: mockbank)
DB_SSLMODE=disable    # SSL mode (default: disable)
```

## Test Accounts

The migrations seed the following test accounts:

| Card Number      | CVV | Expiry  | Balance  | Notes   |
|------------------|-----|---------|----------|---------|
| 4532015112830366 | 123 | 12/2025 | $10,000  | Valid   |
| 4556737586899855 | 456 | 06/2026 | $500     | Valid   |
| 5425233430109903 | 321 | 09/2025 | $50      | Valid   |
| 4024007198964305 | 789 | 03/2024 | $5,000   | Expired |

## API Documentation

Swagger UI available at: <http://localhost:8787/docs>

## Chaos Engineering

The API includes configurable failure injection for testing client resilience:

```bash
FAILURE_RATE=0.05    # 5% of requests return 500
MIN_LATENCY_MS=100   # Minimum added latency
MAX_LATENCY_MS=2000  # Maximum added latency
```
