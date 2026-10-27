# Employee API: Gin + PostgreSQL

Employee CRUD with PostgreSQL persistence, constructor-based dependency injection, a repository contract, Swagger UI, and a per-IP rate limiter. This is a separate database iteration of the employee API; the running application has no list-storage fallback.

## Requirements

- Go 1.25 or newer. Use your existing Go installation (`go version`).
- PostgreSQL 17+ locally or Docker Desktop with its Linux engine running.

## Quick start (PowerShell)

From the repository root:

```powershell
cd Z:\Study\employee-api-db-gin
Copy-Item .env.example .env
# Edit .env and replace the sample POSTGRES_PASSWORD before starting Compose.
docker compose up -d --wait

# Set this to match the password/port in .env. URL-encode special characters.
$env:DATABASE_URL = 'postgres://employee_app:employee_dev_password@127.0.0.1:5433/employee_api?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/api
```

The sample password is for local development only; if you change it in .env, update DATABASE_URL accordingly. Compose reads .env automatically; Go does not. Set DATABASE_URL in your shell or GoLand environment. If you already have PostgreSQL, create a dedicated database and role, set DATABASE_URL, and skip Compose. `sslmode=disable` is for the local database; configure verified TLS for remote deployments.

- [Swagger UI](http://127.0.0.1:8080/swagger/index.html)
- [Swagger JSON](http://127.0.0.1:8080/swagger/doc.json)
- [Liveness](http://127.0.0.1:8080/healthz)

HTTP defaults to port 8080. Compose exposes PostgreSQL only on localhost port 5433 to avoid a common local 5432 conflict. Data lives in a named Docker volume and survives API restarts and ordinary `docker compose down`. Changing POSTGRES_PASSWORD after the volume is initialized does not change the existing database password.

## CRUD example

```powershell
$base = 'http://127.0.0.1:8080/api/v1/employees'
$body = @{ name = 'Anand'; email = 'anand@example.com'; department = 'Engineering' } | ConvertTo-Json
$created = Invoke-RestMethod -Method Post -Uri $base -ContentType 'application/json' -Body $body
$id = $created.data.id
Invoke-RestMethod -Uri $base
Invoke-RestMethod -Uri "$base/$id"
$update = @{ name = 'Anand T'; email = 'anand@example.com'; department = 'Platform' } | ConvertTo-Json
Invoke-RestMethod -Method Put -Uri "$base/$id" -ContentType 'application/json' -Body $update
Invoke-RestMethod -Method Delete -Uri "$base/$id"
```

| Method | Route | Success |
|---|---|---|
| POST | /api/v1/employees | 201 with Location |
| GET | /api/v1/employees | 200, paginated list (default ID ascending) |
| GET | /api/v1/employees/:id | 200 |
| PUT | /api/v1/employees/:id | 200, full replacement |
| DELETE | /api/v1/employees/:id | 204, no body |

Success envelope: `{"data": ...}`; errors: `{"error":{"code":"not_found","message":"employee not found"}}`. Empty lists are `[]`. EmployeeRequest is the create/update payload; EmployeeResponse is the public response. Name and department are trimmed and required (1–100 bytes). Email is normalized and validated (up to 254 bytes); the database atomically enforces case-insensitive uniqueness. PostgreSQL generates IDs; sequence gaps are normal after failed inserts.

Invalid IDs/input/JSON return 400, missing employees 404, duplicate emails 409, oversized bodies 413, and wrong media types 415. Unexpected database errors are logged while clients receive generic 500 responses. Request bodies are limited to 1 MiB; unknown fields, JSON null, and extra JSON values are rejected.

## Structure and dependency injection

```text
cmd/api/                          API startup, dependency wiring, shutdown
cmd/migrate/                      Versioned migration command
internal/config/                  Environment configuration and validation
internal/database/                PostgreSQL pool initialization and connection check
internal/employee/                Domain model, request, service, validation, repository contract
internal/repository/              PostgreSQL adapter and repository tests
internal/httpapi/                 Routes, handlers, DTOs, shared HTTP helpers, rate limiter, Swagger
internal/apidocs/                 Generated Swagger files
internal/testutil/                Test doubles used only by automated tests
migrations/                       Embedded versioned up/down SQL
.run/                             Shared GoLand run/debug configuration
docs/architecture.md              Design notes and operational boundaries
compose.yaml                      Local PostgreSQL service and persistent volume
.env.example                      Non-secret local configuration template
```

Request flow: HTTP handler -> employee service -> employee.Repository -> EmployeeRepository -> PostgreSQL. The service knows no SQL or Gin. main injects a GORM-backed repository. GORM CRUD operations, bounded pool size, request contexts, and query timeouts live in the storage adapter. Tests remain beside the code in *_test.go files. List-based test doubles are isolated in testutil and never wired into the API.

## Configuration

| Variable | Default | Purpose |
|---|---|---|
| DATABASE_URL | required | PostgreSQL connection URL; never commit real credentials |
| HTTP_ADDR | 127.0.0.1:8080 | API listen address |
| DB_MAX_CONNECTIONS | 10 | Pool limit, 1–100 per API process |
| DB_CONNECT_TIMEOUT | 5s | Startup connection timeout |
| DB_QUERY_TIMEOUT | 3s | Per-repository-operation timeout |
| GIN_MODE | debug | Set release for deployment |
| POSTGRES_PASSWORD | required by Compose | Local container initialization password |
| POSTGRES_PORT | 5433 | Local Compose PostgreSQL port |

The API checks database connectivity and the employee schema before starting. Apply migrations explicitly, before starting an instance; the API does not perform DDL on startup. Ctrl+C gracefully drains HTTP requests before closing the pool.

## Migrations

```powershell
go run ./cmd/migrate up
go run ./cmd/migrate version
```

Migrations use golang-migrate, an embedded SQL source, a schema_migrations version table, and a database lock. Re-running up is safe when no changes remain. Add numbered pairs such as 000002_description.up.sql and .down.sql; never edit a migration already applied to a shared database. A failed migration marks the version dirty: inspect and repair the database before reconciling its version through your migration tooling. Down SQL is provided for reviewed rollback procedures; the wrapper deliberately exposes only up/version. Use a migration role with DDL privileges in production and a separate least-privilege runtime role. The Compose role is a local development convenience.

## Rate limiting and Swagger

GET /api/v1/employees allows 10 requests per client IP per rolling 5-second window. Excess calls receive 429 and Retry-After (seconds, rounded up). Other endpoints are unaffected. Quotas are process-local; multi-replica deployments need a shared limiter. Proxy headers are not trusted by default.

Swagger UI is available at /swagger/index.html and serves assets locally. It uses Swagger 2.0 and the current server origin. After editing annotations or API schemas:

```powershell
go generate ./internal/httpapi
```

Commit generated internal/apidocs files with annotation changes. The generator is pinned to Swag v1.16.6.

## Verification

```powershell
go fmt ./...
go test ./...
go vet ./...
go build ./cmd/api
go build ./cmd/migrate

# Real PostgreSQL tests: set a dedicated test database URL, never a production database.
$env:TEST_DATABASE_URL = $env:DATABASE_URL
go test -tags=integration ./internal/repository -v
```

Ordinary tests require no database and cover HTTP behavior, validation, Swagger, rate limiting, configuration, and database error mapping. Integration tests create a uniquely named schema in the selected database and remove that schema afterward. They apply migrations twice, exercise real SQL-backed CRUD, atomic uniqueness under concurrent writes, context cancellation, and visibility across connection pools. The test role needs CREATE SCHEMA privileges. The integration command fails when TEST_DATABASE_URL is missing rather than silently skipping coverage. Optional `go test -race ./...` requires a supported C compiler on Windows.

## GoLand

Open this repository root. Under Settings > Go > GOROOT select your existing system installation (for example C:\Program Files\Go). Select the shared **Employee API** configuration, add DATABASE_URL through Run > Edit Configurations > Environment, and Run or Debug. Keep credentials local; do not save them into the shared XML. Run migrations before debugging.

## Files to commit

Commit cmd/, internal/, migrations/, docs/, .run/, compose.yaml, .env.example, .gitignore, go.mod, go.sum, and README.md. Keep .idea/, .env and other secret environment files, binaries, and coverage output out of Git.

The list endpoint returns bounded pages; updates use last-write-wins behavior. Authentication/authorization, TLS/trusted proxies, metrics, backups, and shared rate limiting require deployment-specific decisions before public production use.

## GORM persistence

The application uses gorm.io/gorm and gorm.io/driver/postgres. database.Open returns *gorm.DB; its underlying database/sql pool provides bounded connection reuse. Repository constructors accept *gorm.DB. All employee CRUD uses GORM methods rather than direct pgx queries. PostgreSQL remains Docker-hosted through compose.yaml.

GORM does not replace the versioned SQL migration workflow. AutoMigrate is not called; continue using go run ./cmd/migrate up. The migration command uses the GORM connection's underlying SQL pool with golang-migrate. The PostgreSQL GORM dialect uses pgx internally, so pgx remains an indirect dependency even though application code does not import it. Switching database engines still requires changing the dialect, migration driver, and SQL migrations.

## Employee pagination and sorting

Use the collection route, with optional query parameters:

```text
GET /api/v1/employees?page=2&page_size=10&sort_by=name&sort_order=asc
```

| Parameter | Default | Allowed values |
|---|---|---|
| page | 1 | Integer, 1–1000000 |
| page_size | 20 | Integer, 1–100 |
| sort_by | id | id, name, email, department |
| sort_order | asc | asc, desc |

Requests without query parameters now return the first 20 employees rather than the entire collection. The data array remains in place, with an additional pagination object:

```json
{
  "data": [],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_items": 0,
    "total_pages": 0
  }
}
```

Invalid, empty, or repeated supported query parameters return 400. Sort fields and directions are validated; user strings are never used as raw ORDER BY fragments. Text sorting follows the database collation. Equal values are ordered by ID ascending as a deterministic tiebreaker. Pages beyond the last return an empty data array and the actual totals. The existing 10-requests-per-5-seconds rate limit covers every query variation on this route.

EmployeeListRequest and EmployeePage define the service/repository contract. The GORM repository executes COUNT plus a LIMIT/OFFSET query; it does not load the full collection to page in application memory. Totals and page data are separate reads, so concurrent writes can change their relationship. Offset pagination can also shift between requests during concurrent writes; deep pages may be expensive. Consider cursor pagination for large, frequently changing collections. No schema migration is needed for this API change.
