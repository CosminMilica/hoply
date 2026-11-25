# Profile

Hoply: job search with a CV tailored to each job

## Quick Start

```bash
# Apply database migrations
go run ./cmd/migrate

# Start the development server
go run ./cmd/server
```

The server starts on [http://localhost:5010](http://localhost:5010).

## API Endpoints

| Method | Path | Description |
| ------ | ---- | ----------- |
| GET | `/api/v1` | Welcome message |
| GET | `/api/v1/info` | Service info |
| GET | `/api/v1/health/live` | Liveness probe |
| GET | `/api/v1/health/ready` | Readiness probe |
| GET | `/api/v1/profiles/me` | The caller's profile, created empty on first call |
| PATCH | `/api/v1/profiles/me` | Change the sent profile fields (`null` clears `notice_period_weeks`, `salary_min`) |
| PUT | `/api/v1/profiles/me/skills` | Replace the skills |
| POST | `/api/v1/profiles/me/experience` | Add an experience entry |
| PUT | `/api/v1/profiles/me/experience/{id}` | Replace an experience entry |
| DELETE | `/api/v1/profiles/me/experience/{id}` | Remove an experience entry |
| POST | `/api/v1/profiles/me/projects` | Add a project |
| PUT | `/api/v1/profiles/me/projects/{id}` | Replace a project |
| DELETE | `/api/v1/profiles/me/projects/{id}` | Remove a project |
| POST | `/api/v1/profiles/me/education` | Add an education entry |
| PUT | `/api/v1/profiles/me/education/{id}` | Replace an education entry |
| DELETE | `/api/v1/profiles/me/education/{id}` | Remove an education entry |
| POST | `/api/v1/profiles/me/certifications` | Add a certification |
| PUT | `/api/v1/profiles/me/certifications/{id}` | Replace a certification |
| DELETE | `/api/v1/profiles/me/certifications/{id}` | Remove a certification |
| POST | `/api/v1/profiles/me/languages` | Add a language |
| DELETE | `/api/v1/profiles/me/languages/{id}` | Remove a language |

The profile API contract (shapes, validation rules, ordering) is
[`docs/api/profile.md`](../../docs/api/profile.md). Every request runs as the
identity bound by the auth (or, with `auth.mode: none`, the dev) middleware;
the profile is keyed by its tenant and subject. Schema: `migrations/00100_profiles.sql` and `00101_profile_v2.sql` (which converts
existing rows to the v2 shape).

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
