# Applications

Hoply: job search with a CV tailored to each job

## Quick Start

```bash
# Apply database migrations
go run ./cmd/migrate

# Start the development server
go run ./cmd/server
```

The server starts on [http://localhost:5030](http://localhost:5030).

## API Endpoints

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/api/v1` | Welcome message |
| GET | `/api/v1/info` | Service info |
| GET | `/api/v1/health/live` | Liveness probe |
| GET | `/api/v1/health/ready` | Readiness probe |
| GET | `/api/v1/applications` | The caller's applications, most recently changed first |
| PUT | `/api/v1/applications/{job_id}` | Create or move an application (`saved`, `cv_ready`, `applied`) |
| DELETE | `/api/v1/applications/{job_id}` | Remove an application |
| GET | `/api/v1/skips` | The caller's skipped job ids, most recently skipped first |
| PUT | `/api/v1/skips/{job_id}` | Skip a job |
| DELETE | `/api/v1/skips/{job_id}` | Bring one skipped job back |
| DELETE | `/api/v1/skips` | Bring every skipped job back |

The applications API contract (shapes, status rules, ownership) is
[`docs/api/applications.md`](../../docs/api/applications.md). Rows belong to
the identity's tenant (a person), kept apart by row-level security. Schema: `migrations/00100_applications.sql`,
`00101_tenant_rows.sql`.

## Layout

| Path | Contents |
| ---- | -------- |
| `cmd/server`, `cmd/migrate` | Entry points |
| `internal/app` | Composition root: routes, services and middleware |
| `internal/routes`, `internal/services`, `internal/data/repositories` | HTTP, business rules, persistence |
| `internal/middleware`, `internal/ports` | Generated runtime and port contracts |
| `migrations` | Embedded goose SQL migrations: `00001`–`00099` belong to forge, project migrations start at `00100`; never delete an applied one (see `OPERATIONS.md`) |
| `tests/unit`, `tests/integration`, `tests/e2e` | Native suites |

## Commands

| Command | Description |
| ------- | ----------- |
| `go run ./cmd/server` | Start server |
| `go test ./tests/unit/...` | Unit tests |
| `DATABASE_URL=... go test ./tests/integration/... ./tests/e2e/...` | Integration and E2E tests |
| `go vet ./...` | Static checks |
| `go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...` | Lint |
| `go run ./cmd/migrate` | Apply pending migrations (same as `up`) |
| `go run ./cmd/migrate status` | List migrations and their state (read-only) |
| `go run ./cmd/migrate version` | Print the highest applied version (read-only) |
| `go run ./cmd/migrate down` | Roll back the most recently applied migration |

## Docker

```bash
docker compose up -d
```

Starts the app, its migrations, PostgreSQL, and optionally pgAdmin (`--profile tools`).
