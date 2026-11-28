# Jobs

crashr: job search with a CV tailored to each job

## Quick Start

```bash
# Apply database migrations
go run ./cmd/migrate

# Start the development server
go run ./cmd/server
```

The server starts on [http://localhost:5020](http://localhost:5020).

## API Endpoints

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/api/v1` | Welcome message |
| GET | `/api/v1/info` | Service info |
| GET | `/api/v1/health/live` | Liveness probe |
| GET | `/api/v1/health/ready` | Readiness probe |
| GET | `/api/v1/jobs` | The catalog plus the caller's own jobs, newest first; filters `q`, `city`, `country`, `include_remote`, `area`, `level` |
| GET | `/api/v1/jobs/{id}` | One visible job |
| POST | `/api/v1/jobs/read` | Read a link or pasted text into a draft (saves nothing) |
| POST | `/api/v1/jobs` | Add a job |
| DELETE | `/api/v1/jobs/{id}` | Remove one of the caller's own jobs |

The jobs API contract (shapes, rules, reading a link or text, the catalog)
is [`docs/api/jobs.md`](../../docs/api/jobs.md). Ads are read by rules
in `internal/extract` (skills dictionary in `internal/extract/skills.json`);
links are fetched by `internal/fetch` (Greenhouse, Lever, Ashby,
SmartRecruiters, BambooHR and Workable links through their public APIs, other links from
the page), which refuses non-public addresses on every connection. Schema: `migrations/00100_jobs.sql` (the catalog starts empty).

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
