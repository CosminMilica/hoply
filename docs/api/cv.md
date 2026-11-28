# CV API

Service: `services/cv` (Go). The browser calls `/api/cv/v1/...`; Traefik and the Vite proxy rewrite it to
`/api/v1/...` on the service. JSON is `snake_case`. Errors use the service's error envelope, with `context.field` on
validation errors. Every request is signed in through the gatekeeper (Keycloak); a person is a tenant, and row-level security keeps each tenant's rows from every other. The cv service calls profile and jobs with the gatekeeper's service tokens for the
tenant it works for.

The service asks the questions for a job and writes CVs tailored to it. It reads the profile from the profile service
(`GET /api/v1/profiles/me`) and the job from the jobs service (`GET /api/v1/jobs/{id}`), and keeps a snapshot of both
in each CV, so later edits never change a CV already made. Writing is asynchronous: a CV is queued, written by a worker
in the service, then checked. The worker writes with Claude when the service is connected (`docs/claude-setup.md`),
else with the rules engine (`docs/plans/2026-11-25-cv-service.md`, §4–6).

## Endpoints

| Method | Path (service side) | Body | Response |
| --- | --- | --- | --- |
| GET | `/api/v1/jobs/{job_id}/questions` | | 200 `PrepQuestions`; 404 unknown job; 503 profile or jobs unavailable |
| POST | `/api/v1/jobs/{job_id}/analysis` | | 202 `{ status: pending \| unavailable }`; 404 unknown job; 503 jobs unavailable |
| GET | `/api/v1/jobs/{job_id}/probe` | | 200 `Probe`; 404 unknown job; 503 profile or jobs unavailable |
| GET | `/api/v1/probe` | | 200 `Probe` for the whole profile; 503 profile unavailable |
| POST | `/api/v1/cvs` | `CvRequest` | 202 `Cv` with `status: queued`; 404 unknown job; 422 invalid body; 503 profile or jobs unavailable |
| GET | `/api/v1/cvs` | query `job_id` (optional) | 200 `{ items: [Cv] }`, newest first |
| GET | `/api/v1/cvs/{id}` | | 200 `Cv`; 404 |
| GET | `/api/v1/cvs/{id}/file` | query `format` (`pdf` or `docx`), `version` (`tailored`, the default, or `profile`), `template` (`classic`, the default, `modern` or `elegant`) | 200 the file as an attachment named `file_name` (`.docx` for DOCX); 409 not ready; 422 bad `format`, `version` or `template`; 503 PDF printer unavailable |
| GET | `/api/v1/cvs/{id}/letter` | | 200 `Letter`; 404 |
| POST | `/api/v1/cvs/{id}/letter` | `{ recruiter? }` (at most 80 characters) | 202 `Letter` with `status: writing`, or `failed` without Claude; 404; 409 not ready; 422 bad `recruiter` |
| POST | `/api/v1/cvs/{id}/letter/file` | `{ text }` (1–8000 characters), query `format` (`pdf` or `docx`), `template` | 200 the cover letter as an attachment named like `file_name` with `Cover_Letter` for `CV`; 404; 409 not ready; 422 bad `format`, `template` or `text`; 503 PDF printer unavailable |
| DELETE | `/api/v1/cvs/{id}` | | 204; 404 |
| GET | `/api/v1/writer` | | 200 `WriterStatus` |

`{job_id}`, `{id}`, `job_id` and `CvRequest.job_id` must be UUIDs (422 with `context.field` otherwise). Every CV
belongs to the caller; another person's CVs are never visible or changed.

## Shapes

```text
PrepQuestions { job_id, base_score, covered: [skill], gaps: [{ skill, ask, example }], related: [{ skill, via: [skill], note }],
                responsibilities: [{ index, text, question, prefill }] }
CvRequest     { job_id, skip_roles: [experience id], focus: [skill] (at most 3; empty takes the first three matched),
                confirmed: [int] (indexes of the key responsibilities the person confirmed), language: "en" | "ro" (default "en") }
Cv            { id, job_id, status, created_at, updated_at, job_title, company, file_name,
                highlights: [skill] | null, profile: CvDocument | null, tailored: CvDocument | null,
                ats: { passed, total, checks: [{ ok, label, hint }] } | null, preview: { from, to } | null,
                written_by: "claude" | "rules" | null, model, refused, rules_reason, limit_until | null }
WriterStatus  { writer: "claude" | "rules", model, state, limit_until | null }
Probe         { status: "ready" | "pending" | "unavailable", questions: [{ theme, question, why, example, about, place, choices: [string] }] }
CvDocument    { name, headline, contact: [string], links: [string], summary, skills: [string],
                experience: [{ role, meta, bullets: [string] }], projects: [{ name, meta, bullets: [string] }],
                education: [{ degree, meta }], certifications: [string], languages, additional: [string],
                language: "en" | "ro" (its headings follow it) }
```

| Field | Rule |
| --- | --- |
| `status` | `queued`, `writing`, `checking`, `ready` or `failed` |
| `highlights`, `profile`, `tailored`, `ats` | set when `status` is `ready`; `null` before |
| `preview` | the first highlight of the first role kept, before and after the rewrite; set from `checking` on, `null` when no role has a highlight |
| `written_by` | who wrote the tailored version, once `ready`. The profile version is always the rules' |
| `model` | the Claude model, when `written_by` is `claude` |
| `refused` | how many of Claude's sentences the checks replaced with the rules' rewrite of the facts they cited (left out on a Romanian CV, as the rules write English) |
| `rules_reason` | why the rules wrote it: `off`, `no_token`, `login_needed`, `usage_limit` or `claude_failed`; empty for Claude |
| `limit_until` | with `usage_limit`: when Claude is available again |
| `WriterStatus.state` | `ok`, `off`, `no_token`, `login_needed` or `usage_limit` (`docs/claude-setup.md`) |
| `file_name` | `<Name>_CV_<Company>.pdf`, every run of characters other than letters and digits replaced by `_`; `CV_<Company>.pdf` without a name |

A CV left in `writing` or `checking` when the service stops goes back to `queued` when it starts again. A CV that
cannot be written is `failed`; the web app offers to try again.

## Reading the job

The requirements to write around are `must_have` then `nice_to_have`. With Claude, they are the analysis's: it reads
the skills the ad asks for as a person lists them on a CV, `must` or `nice` (a plus, preferred, or only part of the
stack), and leaves out practices, qualities and the field itself. A new analysis writes them back to the job
(`PUT /jobs/{id}/requirements` in the jobs service), so the match and Prepare your CV use them too. The web app asks
for the analysis in the background when a job is added (`POST /jobs/{job_id}/analysis`); otherwise it runs with the
first CV or the first questions. An analysis kept before it read requirements is read again. The key responsibilities asked about are the
job's first four `responsibilities`. A job without a `product` is "products" in the summary.

The person's skills are the profile's `skills`, the `tools` of every role and project, and the `skills` of every
highlight and note. A requirement is matched against them through `services/cv/internal/cvdoc/skillmap.json` (the web
app holds an identical copy, checked by a test):

- covered when the person has it under any of its names ("Argo CD", "ArgoCD") or has a tool that proves it
  (GitHub Actions proves CI/CD, Docker proves Containers, Terraform proves Infrastructure as Code);
- related when they have a tool of the same kind (FluxCD for Argo CD: both GitOps). Related requirements are not
  asked as gaps, and the CV never names them: it shows their kind ("GitOps delivery with FluxCD");
- missing otherwise.

The score weighs a must-have twice a nice-to-have, and a related requirement half.

## Writing

The CV comes in two versions from one snapshot: `profile` as the profile reads, and `tailored`:

- the summary names the headline, the years of experience (from the earliest role's start month), the focus skills,
  the second sentence of `about`, and the job and company;
- the headline adds the focus skills;
- the skills list starts with the matched skills;
- each bullet uses the job's terms and verbs and names at most two matched tools of that role that fit its wording;
  bullets are ordered by keyword hits, then by having a number, and the first role keeps 7, the others 3;
- roles in `skip_roles` are left out.

Facts the person confirms in Prepare your CV live in the profile (highlights or additional information), not in the
request. Dates print as `MM/YYYY – Present`, bullets as their text followed by their result, education as its degree
title, else its level and field. The contact line carries the city and country.

## ATS checks

Twelve checks run on the tailored version: layout and headings, email and phone, location, month-and-year dates on
every role kept, the job title in the summary, the job's keywords present and used in sentences, responsibilities
mirrored, action verbs, results with numbers, and length.

## Written by Claude

Claude gets the job as data and the profile as numbered facts (`E1.B2` is the second highlight of the first role
kept). It returns the summary, the bullets of each role and project and the order of the skills. Every sentence
cites the facts it rests on. The code keeps a sentence only when:

- it cites facts that exist and, for a bullet, belong to that role or project;
- every number in it appears in the facts it cites;
- it names none of the job's requirements the profile does not show, nor a tool the person knows only an
  equivalent of;
- a bullet does not name the company being applied to;
- it is at most 300 characters.

A sentence that fails is replaced with the rules' rewrite of the fact it cites, or dropped. A role or project
Claude leaves out keeps the rules' bullets. Name, contact, dates, companies, education and the profile version
always come from the profile. The ATS checks run on the final text.

## Files

The PDF is the CV as HTML (`internal/render/html.go`) printed by Gotenberg's Chromium on A4 (`GOTENBERG_URL`).
The DOCX is written by the service (`internal/render/docx.go`), with Word's own Title, Heading 1, Heading 2 and List
Bullet styles. Both are one column with standard headings, real text, and no tables, images or text boxes, in the
order of the web app's CV paper. A section with nothing in it is left out.

The templates (`internal/render/templates.go`) change only type, color and alignment, never the text or its order:

| Template | Look | PDF font | DOCX font |
| --- | --- | --- | --- |
| `classic` | black, uppercase headings over a rule | Liberation Sans (Arial) | Arial |
| `modern` | blue name and headings, a bar beside each heading | Carlito (Calibri) | Calibri |
| `elegant` | serif, centered header, headings between two thin rules | Caladea (Cambria) | Cambria |

The web app shows the CV in the template the person picked and remembers it in the browser.

## Claude's questions

`GET /jobs/{job_id}/probe` returns what Claude would ask to make the CV stronger for the job: questions about the
person's facts that best prove the job's priorities but lack a result or scale, about priorities the facts barely
touch, and about the traits the ad stresses (`prompts/probe-system.md`). The first call starts asking in the
background and answers `pending`; the web app asks again every two seconds until `ready`. The questions are kept per
person and job for the facts as they were, and asked again when the facts change. `unavailable` means Claude cannot
be asked now (no token, a usage limit, a failure); it is tried again after five minutes. `about` is the fact or role
the question is about, `place` where an answer goes (`experience:<id>`, `projects:<id>` or `other`); the web app saves
answers as notes there.

`choices` are up to four answers the person can pick instead of typing, offered when the answer is a count, a size,
a frequency or a before and after: non-overlapping ranges in the person's field ("Covers 5–20 AWS accounts"). People
rarely know the exact figure, and an honest range is a fact the CV can use; a picked choice is saved as a note like a
typed answer.

`GET /probe` asks the same way for the whole profile, once for every job (`prompts/profile-probe-system.md`): results,
scale or frequency for the facts every CV leans on (the current role's highlights and notes, the strongest projects).
They are kept like a job's questions, under no job, and the web app shows them on the profile page as "Make every CV
stronger".

## Cover letter and messages

`POST /cvs/{id}/letter` asks Claude, in the background, for four texts written from a ready CV: a cover letter, a
LinkedIn connection note, a message to the recruiter for right after applying, and a follow-up for a week later
(`prompts/letter-system.md`). The web app asks again every two seconds while it is `writing`.

```text
Letter      { status: none | writing | ready | failed, recruiter, texts: LetterTexts | null, model, refused,
              reason: "" | off | no_token | login_needed | usage_limit | claude_failed, limit_until, updated_at }
LetterTexts { cover_letter, connection_note, message: { subject, body }, follow_up: { subject, body } }
```

- One Claude call reads the job's kept analysis, the CV as it will be sent, how it was tailored and the numbered
  facts, plus what the ad says about the role and the company as facts `J1`, `J2`…
- Each sentence passes the CV's checks (cited facts, their numbers, no missing or related requirement named), with
  the company allowed and the job title never counted as a claim. Clichés ("passionate", "team player", "I am
  writing to"…) and exclamation marks fail. A failed sentence is left out; a subject that fails is replaced by
  "Application for <title>". `refused` counts what was left out.
- The greetings and sign-offs are the code's: "Dear <recruiter>," or "Dear <company> hiring team,", "Hi <first
  name>," in the messages, and the person's name. The connection note is cut, sentence by sentence, to LinkedIn's
  300 characters.
- A letter left `writing` by a service that stopped reads as `failed` after five minutes. Without Claude there is no
  letter: a rules letter would read as a template.
- The person can edit every text on the page; the edits are not saved. `POST /cvs/{id}/letter/file` prints the
  letter they send with the CV's header, the date and the CV's template.

