# Jobs API

Service: `services/jobs` (Go). The browser calls `/api/jobs/v1/...`; Traefik and the Vite proxy rewrite it to
`/api/v1/...` on the service. JSON is `snake_case`. Errors use the service's existing error envelope, with
`context.field` on validation errors. No login yet: the dev identity owns the jobs it adds.

The service holds two kinds of jobs: the **catalog** (jobs to discover; for now the seeded sample jobs, later
imports from public job boards) and the jobs a person **adds** from a link or pasted text. It reads job ads with rules
and a skills dictionary only, never with an LLM. Matching against the profile stays in the web app for now.

## Endpoints

| Method | Path (service side) | Body | Response |
| --- | --- | --- | --- |
| GET | `/api/v1/jobs` | query: `q`, `city`, `country`, `include_remote`, `area`, `level` | 200 `{ items: [Job], total }` |
| GET | `/api/v1/jobs/{id}` | | 200 `Job` |
| POST | `/api/v1/jobs/read` | `{ url }` or `{ text, title?, company?, url? }` | 200 `JobDraft` |
| POST | `/api/v1/jobs` | `JobInput` | 201 `Job` |
| DELETE | `/api/v1/jobs/{id}` | | 204 |

- `GET /jobs` returns the catalog plus the caller's own jobs (`added_by_me: true`), newest first; other people's added
  jobs are never visible. Filters: `q` matches title, company and requirements (case-insensitive); `city` matches the
  city (case- and diacritics-insensitive: `Iasi` finds `Iași`); `include_remote=true` also keeps remote jobs when
  `city` is set; `country` matches the country; `area` may repeat; `level` is one of the level values.
- `GET /jobs/{id}` and `DELETE /jobs/{id}`: an unknown id, a non-UUID id or another person's job is 404; deleting a
  catalog job is 404 too.
- `POST /jobs/read` saves nothing. It returns what the rules read, for the person to check and edit.

## Shapes

```text
Job       { id, title, company, city, country, work_mode, level, years_required,
            must_have: [string], nice_to_have: [string], responsibilities: [string],
            about, about_company, product, perks: [string], areas: [string],
            url, source, read_from, sample, added_by_me, created_at }
JobInput  { title, company, city, country, work_mode, level, years_required, must_have, nice_to_have,
            responsibilities, about, url, source, read_from }
JobDraft  JobInput + { host }
```

| Field | Values |
| --- | --- |
| `work_mode` | `""`, `remote`, `hybrid`, `onsite` |
| `level` | `""`, `junior`, `mid`, `senior`, `lead` |
| `years_required` | `null` or 0–40 |
| `source` | `catalog`, `link`, `text` |
| `read_from` | `ats_api` (the job board's public API, see below), `json_ld` (the page's schema.org `JobPosting`), `page_text` (the page's main text), `text` (pasted), `""` for catalog jobs |
| `host` | the link's host name, or `""` |

Strings are never null (`""` when unset), arrays never null (`[]`).

## Rules for `POST /jobs` (422 otherwise)

| Field | Rule |
| --- | --- |
| `title` | required, at most 160 characters |
| `company`, `city`, `country` | at most 120 characters |
| `must_have`, `nice_to_have` | trimmed, empty and case-insensitive duplicates dropped (also across the two lists, `must_have` wins), at most 40 each, each at most 60 characters |
| `responsibilities` | trimmed, empty dropped, at most 30, each at most 400 characters |
| `about` | at most 4000 characters |
| `url` | empty or an `http(s)` URL, at most 2000 characters |
| `source` | `link` or `text` (`catalog` is refused) |

## Reading a job (`POST /jobs/read`)

| Input | Rule |
| --- | --- |
| `url` | required for a link; `http`/`https` only, at most 2000 characters |
| `text` | at least 80 characters, at most 50000 |
| `title`, `company` | optional hints; when set they win over what the rules read |

**Fetching a link** (server side): at most 3 redirects, 10 s, `text/html` (or `application/json` for the board APIs)
only; at most 2 MB for a page and 10 MB for a board API answer (Ashby returns the whole board). Every hop resolves the host and refuses loopback, private, link-local, multicast and unspecified
addresses (no SSRF). The link is read in this order:

1. **A known job board's public API** (`read_from: ats_api`), chosen from the link's shape; no key needed:

   | Link | API call |
   | --- | --- |
   | `boards.greenhouse.io/{board}/jobs/{id}`, `job-boards.greenhouse.io/{board}/jobs/{id}`, any URL with `gh_jid={id}` and a known board | `boards-api.greenhouse.io/v1/boards/{board}/jobs/{id}` |
   | `jobs.lever.co/{site}/{id}` (also `jobs.eu.lever.co`) | `api.lever.co/v0/postings/{site}/{id}` (`api.eu.lever.co` for the EU host) |
   | `jobs.ashbyhq.com/{board}/{id}` | `api.ashbyhq.com/posting-api/job-board/{board}`, the posting with that id |
   | `jobs.smartrecruiters.com/{company}/{id}` | `api.smartrecruiters.com/v1/companies/{company}/postings/{id}` |
   | `apply.workable.com/{slug}/j/{shortcode}` | `apply.workable.com/api/v1/widget/accounts/{slug}?details=true`, the job with that shortcode |

   Title, company, location and remote flags come from the API's fields; the description HTML (with its section
   headings) goes through the text rules below. If the API call fails, fall back to the page.
2. The page's schema.org `JobPosting` JSON-LD (title, `hiringOrganization.name`, `jobLocation.address`,
   `jobLocationType: TELECOMMUTE` → `remote`, `description` HTML turned into lines, list items kept as lines).
3. The main text of the page (scripts, styles, nav, header, footer and forms removed). Title from `<h1>`, else
   `og:title`, else `<title>`; company from `og:site_name`, else a `<title>` of the form `… at {Company}`. Bold lines
   (`<strong>`, `<b>`) and `<h2>`–`<h4>` become heading lines.

The result then goes through the same rules as pasted text.

Failures are 422 with `code: PAGE_UNREADABLE` and `context.reason`:

| `reason` | When |
| --- | --- |
| `login_required` | known login-walled hosts (`linkedin.com`, `glassdoor.*`), or a login page answered |
| `no_job_data` | no `JobPosting` and less than 300 characters of main text |
| `fetch_failed` | network error, timeout, a status other than 2xx, or not HTML |
| `too_large` | more than 2 MB |
| `blocked_address` | the host resolves to a refused address |

**The rules** (English and Romanian):

- Text is normalised first: curly quotes and apostrophes become straight ones, non-breaking spaces become spaces.
- Sections by their heading line (shorter than 60 characters, optional `#`, `*` or trailing `:`), matched on the start
  of the line, case-insensitive:
  - requirements: `Requirements`, `Minimum qualifications`, `Qualifications`, `What we look for`, `What we're looking
    for`, `What you bring`, `What you'll bring`, `What you'll need`, `What you need`, `What we require`, `What makes you a great fit`, `Who you are`, `About you`, `You have`,
    `Skills and experience`, `Must have`, `Cerințe`, `Ce căutăm`, `Profilul candidatului`, `Ce aduci`
  - nice to have: `Nice to have`, `Bonus`, `Preferred`, `Preferred qualifications`, `It's a plus`, `Extra points`,
    `Constituie un avantaj`, `Ar fi un plus`, `Reprezintă un avantaj`
  - responsibilities: `Responsibilities`, `Key responsibilities`, `What you will do`, `What you'll do`, `What you'll be
    doing`, `What you will be doing`, `Your role`, `Your responsibilities`, `In this role`, `The role`,
    `Responsabilități`, `Ce vei face`, `Atribuții`
  - ignored (their lines feed no field): `Benefits`, `What we offer`, `Perks`, `Why you'll`, `Why join`, `About us`,
    `About {company}`, `Our culture`, `Equal opportunity`, `Apply for this job`, `How to apply`, `Ce oferim`,
    `Beneficii`, `Despre noi`
- Skills come from a dictionary with aliases (`K8s` → `Kubernetes`, `IaC` → `Infrastructure as Code`, `Postgres` →
  `PostgreSQL`, `golang` → `Go` and a case-sensitive `Go`, Romanian terms such as `rețele` → `Networking`). Skills found
  in the nice-to-have section are `nice_to_have`; skills found in the requirements and responsibilities sections are
  `must_have`; with no sections at all, every skill in the text is `must_have`.
- `responsibilities`: the lines of the responsibilities section, list markers removed.
- `years_required`: the first `N years`, `N+ years`, `N yrs` or `N ani`.
- `city`: from a list of Romanian cities with and without diacritics (Iași, București/Bucharest, Cluj-Napoca,
  Timișoara, Brașov, Constanța, Craiova, Galați, Oradea, Sibiu, Arad, Ploiești, Târgu Mureș); `country` is `Romania`
  when a Romanian city or the word Romania/România appears.
- `work_mode`: `hybrid`/`hibrid` → `hybrid`, else `on-site`/`la birou`/`office-based` → `onsite`, else `remote`/`de
  acasă` → `remote`.
- `level`: from the title (`Senior`/`Sr` → `senior`, `Junior`/`Jr`/`Entry` → `junior`, `Lead`/`Principal`/`Staff`/`Head`
  → `lead`), else from `years_required` (≥ 5 → `senior`, ≤ 1 → `junior`, otherwise `mid`); `""` when neither exists.
- `title`: the hint, else the JSON-LD title, else the first line. `company`: the hint, else the JSON-LD organisation,
  else a `Company:` / `Firma:` / `Angajator:` line.
- `about`: the body lines before the first section, longer than 40 characters, joined, at most 4000 characters.

## Catalog

The catalog is empty until finding jobs is built (`docs/job-sources.md`): migration `00100_jobs.sql` creates the table
and seeds nothing, so `GET /jobs` returns only the caller's own jobs for now.
