# Mocked services

Two services the web app calls through typed clients don't exist yet: applications and cv. In the browser, MSW
answers them from in-memory stores, which reset on every page load. MSW runs on the Vite dev server only
(`VITE_MSW_ENABLED=true` in `apps/frontend/.env.development`); a production build has neither the handlers nor
the worker script. The profile and jobs services are real (`docs/api/profile.md`, `docs/api/jobs.md`): they have no
handler, and their requests pass through to the network.

JSON is `snake_case`. Errors use the platform envelope `{ message, type, detail }`.

```text
apps/frontend/src/
  custom/mocks/handlers.ts       the list the browser worker serves; delete a service's line when it ships
  features/applications/mocks/   handlers.ts (store inside)
  features/cv/mocks/             handlers.ts, engine/ (the tailoring engine, plain TypeScript)
```

## applications (`/api/applications/v1`)

| Method | Path | Body | Response |
| --- | --- | --- | --- |
| GET | `/applications` | | 200 `{ items: [Application] }`, most recently changed first |
| PUT | `/applications/{job_id}` | `{ status }` | 200 `Application` (creates or updates), 422 for an unknown status |
| DELETE | `/applications/{job_id}` | | 204, 404 |
| GET | `/skips` | | 200 `{ job_ids: [string] }` |
| PUT | `/skips/{job_id}` | | 204 |
| DELETE | `/skips/{job_id}` | | 204 |
| DELETE | `/skips` | | 204 (brings back every skipped job) |

```text
Application { job_id, status: saved | cv_ready | applied, updated_at }
```

The UI sets `cv_ready` after it generates a CV. A job that is already `applied` stays `applied`.

## cv (`/api/cv/v1`)

| Method | Path | Query / body | Response |
| --- | --- | --- | --- |
| GET | `/jobs/{job_id}/questions` | | 200 `PrepQuestions`, 404 unknown job, 502 profile or job unreadable |
| POST | `/cvs` | `CvRequest` | 201 `Cv`, 404 unknown job, 422 invalid body, 502 profile or job unreadable |
| GET | `/cvs` | `job_id` (optional) | 200 `{ items: [Cv] }`, newest first |
| GET | `/cvs/{id}` | | 200 `Cv`, 404 |

```text
PrepQuestions { job_id, base_score, covered: [string], gaps: [{ skill, ask, example }],
                responsibilities: [{ index, text, question, prefill }] }
CvRequest     { job_id, skip_roles: [experience id], focus: [skill] (max 3, empty = top 3),
                confirmed: [int] }   the key responsibilities the person confirmed for this job
Cv            { id, job_id, created_at, job_title, company, file_name, highlights: [keyword],
                profile: CvDocument, tailored: CvDocument,
                ats: { passed, total, checks: [{ ok, label, hint }] }, preview: { from, to } | null,
                snapshot: Profile }
CvDocument    { name, headline, contact: [string], links: [string], summary, skills: [string],
                experience: [{ role, meta, bullets: [string] }], projects: [{ name, meta, bullets: [string] }],
                education: [{ degree, meta }], certifications: [string], languages, additional: [string] }
```

`file_name` is `<Name>_CV_<Company>.pdf`, with every run of characters other than letters and digits replaced by
`_`; without a name it is `CV_<Company>.pdf`.

The service reads the profile itself. The mock does this with MSW `bypass()` against
`/api/profile/v1/profiles/me`, so the real profile service answers. The whole profile is kept in the `Cv` as
`snapshot`, and both versions are built from it, so later profile edits don't change a CV already made. It reads
the job the same way, from `/api/jobs/v1/jobs/{id}` with `bypass()`, and maps it for the engine: the requirements
to write around are `must_have` then `nice_to_have`, their wording is "Experience with X." / "X is a plus.", and the
key responsibilities it asks about are the job's first four `responsibilities`. A job without a `product` is
"products" in the summary.

A skill counts as covered (not asked again, and matched) when it is in the profile's `skills`, in the `tools` of
any role or project, or in the `skills` of any highlight. Facts confirmed in Prepare your CV live in the profile,
not in the CV request: the UI saves each one through the profile API as a highlight `{ text, result: "", skills }`
under the role or project the person picks, or as a line of `additional_info`. `CvRequest` only carries what
belongs to this CV.

Dates print as `MM/YYYY – Present`. Bullets print as their text followed by their result. Education prints as
its degree title, else its level and field. The contact line carries the city and country. The ATS check also
looks for month-and-year dates on every role kept, and a location in the header.
