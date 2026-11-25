# Applications API

Service: `services/applications` (Go). The browser calls `/api/applications/v1/...`; Traefik and the Vite proxy
rewrite it to `/api/v1/...` on the service. JSON is `snake_case`. Errors use the service's existing error envelope,
with `context.field` on validation errors. No login yet: every request runs as the dev identity, which owns its rows.

The service tracks what a person does with each job: saved, CV ready, applied, and the jobs they skipped. It stores
only the job's id; the job itself lives in the jobs service (`docs/api/jobs.md`), and the web app joins the two. This
contract is the one the web app already uses against its mock, plus the timestamps.

## Endpoints

| Method | Path (service side) | Body | Response |
| --- | --- | --- | --- |
| GET | `/api/v1/applications` | | 200 `{ items: [Application] }`, most recently changed first |
| PUT | `/api/v1/applications/{job_id}` | `{ status }` | 200 `Application`: creates or updates |
| DELETE | `/api/v1/applications/{job_id}` | | 204; 404 when there is none |
| GET | `/api/v1/skips` | | 200 `{ job_ids: [string] }`, most recently skipped first |
| PUT | `/api/v1/skips/{job_id}` | | 204; skipping twice is fine |
| DELETE | `/api/v1/skips/{job_id}` | | 204, also when it was not skipped |
| DELETE | `/api/v1/skips` | | 204: brings back every skipped job |

`{job_id}` must be a UUID (422 with `context.field: "job_id"` otherwise). The service does not check that the job
exists. Every row belongs to the caller; another person's rows are never visible or changed.

## Shape

```text
Application { job_id, status, saved_at, cv_ready_at, applied_at, created_at, updated_at }
```

| Field | Rule |
| --- | --- |
| `status` | one of `saved`, `cv_ready`, `applied` (422 otherwise) |
| `saved_at`, `cv_ready_at`, `applied_at` | when the application last entered that status; `null` if it never did |

## Status rules

- A new application takes the status sent, and that status's timestamp is now.
- Setting `cv_ready` on an `applied` application keeps it `applied` (a new CV for a job you already applied to) and
  sets `cv_ready_at`.
- Any other change sets the status and its timestamp; earlier timestamps are kept.
- `updated_at` moves on every PUT, so the list order follows the latest change.
- DELETE removes the application and its timestamps; saving the job again starts over.
