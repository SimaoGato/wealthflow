# WealthFlow

A forward-looking personal finance engine built on double-entry accounting principles. WealthFlow bridges the gap between financial planning and daily execution by generating actionable "Money Moves" checklists and providing precise time-to-purchase projections for wishlist items.

## Introduction

WealthFlow implements a **dimensional double-entry ledger** that tracks money across two layers:

- **Physical Layer**: Real-world accounts (checking accounts, savings, investment accounts)
- **Virtual Layer**: Logical budget allocations within physical accounts (e.g., "Free Cash", "Fixed Costs", "Wants Vault")

This dual-layer architecture enables forward-looking financial planning while maintaining strict accounting integrity. Every transaction is validated to ensure debits equal credits within each layer, preventing data corruption and ensuring accurate balance calculations.

## Key Technical Features

### Multi-Layer Accounting

- **Physical Buckets**: Real accounts (e.g., "CGD Checking", "XTB")
- **Virtual Buckets**: Budget subdivisions within physical accounts (e.g., "Free Cash", "Fixed Costs")
- **External Buckets**: Income sources and expense categories (e.g., "Employer", "Groceries")
- **Equity Buckets**: Investment accounts with separate book value and market value tracking

### Real-Time Balance Integrity

PostgreSQL triggers automatically update `buckets.current_balance` on every transaction entry insert, ensuring balances remain consistent without application-level synchronization.

### gRPC-First API Design

All API contracts are defined in Protobuf (`.proto` files) and generated for both Go (server) and Dart (client). This schema-first approach ensures type safety and eliminates API drift between frontend and backend.

### Precision Math

All currency calculations use `shopspring/decimal` to avoid floating-point errors. Amounts are stored as absolute values with a `type` column (`DEBIT`/`CREDIT`) to determine direction.

## System Architecture

WealthFlow follows **Clean Architecture** principles with clear separation of concerns:

```
backend/
├── cmd/server/          # Application entry point
├── internal/
│   ├── domain/          # Pure business entities (no external dependencies)
│   ├── usecase/         # Business logic orchestration
│   └── adapter/
│       ├── grpc/        # gRPC server implementation
│       └── repository/  # PostgreSQL data access (raw SQL)
└── tests/integration/   # End-to-end test suite
```

[ARCHITECTURE_DIAGRAM]

### Domain Layer

Pure Go structs representing core entities: `Bucket`, `Transaction`, `SplitRule`, `Task`. No database or external dependencies.

### Use Case Layer

Business logic services:
- `InflowService`: Handles income recording and split rule execution
- `ExpenseService`: Creates double-layer expense entries
- `InvestmentService`: Manages market value tracking and P/L calculations
- `DashboardService`: Aggregates net worth and liquidity metrics

### Adapter Layer

- **gRPC Adapter**: Maps Protobuf requests to use case services, handles authentication
- **Repository Adapter**: Raw SQL queries using `pgx/v5`, parameterized to prevent SQL injection

### Database Layer

PostgreSQL with:
- **Transactions**: Double-entry headers and entries with layer separation
- **Triggers**: Automatic balance updates on `transaction_entries` inserts
- **Migrations**: Managed via `golang-migrate`

## Getting Started

### Prerequisites

- **Docker & Docker Compose**: For local database and services
- **Go 1.23+**: Backend development
- **Flutter**: Frontend development (latest stable)
- **Protoc**: Protocol Buffer compiler
  ```bash
  # macOS
  brew install protobuf
  
  # Linux
  apt-get install protobuf-compiler
  ```

### Installation

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd wealthflow
   ```

2. **Install Protobuf plugins**:
   ```bash
   make install-deps
   ```

3. **Generate code from Protobuf definitions**:
   ```bash
   make gen
   ```

4. **Start services**:
   ```bash
   make docker-up
   ```

5. **Run database migrations**:
   ```bash
   make migrate-up
   ```

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make install-deps` | Install Protobuf plugins for Go and Dart |
| `make gen` | Generate Go and Dart code from `.proto` files |
| `make clean` | Remove generated code files |
| `make docker-up` | Start Docker Compose services (Postgres + Backend) |
| `make docker-down` | Stop and remove Docker Compose services |
| `make migrate-up` | Run database migrations |
| `make test-integration` | Run end-to-end integration tests |

## API Contract

### gRPC Service

The API is defined in `proto/wealthflow/v1/service.proto`. Key RPCs:

- `RecordInflow`: Log income and trigger split rule engine
- `LogExpense`: Create double-layer expense entries
- `UpdateInvestment`: Update market value for equity buckets
- `ListBuckets`: Query buckets with optional type filter
- `ListTransactions`: Paginated transaction history
- `GetNetWorth`: Calculate total net worth (liquidity + equity)

### Authentication

All state-changing RPCs require authentication via gRPC metadata:

```go
// Example: Set authorization header
metadata.AppendToOutgoingContext(ctx, "authorization", "dev-token")
```

**Development**: Default token is `dev-token` (configurable via `API_TOKEN` environment variable).

**Production**: Replace with JWT or OAuth2 tokens in the interceptor.

### Code Generation

After modifying `.proto` files, regenerate client/server code:

```bash
make gen
```

This generates:
- Go code: `backend/internal/adapter/grpc/wealthflow/v1/`
- Dart code: `frontend/lib/generated/wealthflow/v1/`

## Development

### Running the Backend

The backend server starts automatically with `docker-compose up`. It listens on port `8080` for gRPC connections.

To run locally (without Docker):

```bash
cd backend
go run cmd/server/main.go
```

### Integration Tests

Integration tests verify the full flow from API to database:

```bash
make test-integration
```

Tests require:
- Docker Compose services running (`make docker-up`)
- Database migrations applied (`make migrate-up`)

The test suite (`backend/tests/integration/e2e_test.go`) covers:
- Inflow recording with split rules
- Expense logging with double-layer entries
- Balance integrity verification
- Net worth calculations

### Unit Tests

Run unit tests for business logic:

```bash
cd backend
go test ./...
```

### CI/CD Pipeline

GitHub Actions workflows (`.github/workflows/`) run on every push and pull request:

- **Go Tests**: Unit and integration test execution
- **Linting**: `golangci-lint` code quality checks
- **Flutter Tests**: Frontend widget and unit tests

All checks must pass before merging to `main`.

## Project Structure

```
wealthflow/
├── backend/              # Go gRPC server
│   ├── cmd/server/      # Entry point
│   ├── internal/        # Application code
│   ├── db/migrations/   # SQL migrations
│   └── tests/           # Test suites
├── frontend/            # Flutter application
│   ├── lib/            # Dart source code
│   └── lib/generated/  # Generated gRPC client code
├── proto/              # Protobuf definitions
│   └── wealthflow/v1/
├── docker-compose.yml  # Local development services
└── Makefile           # Common development tasks
```
