# Jobs API

Service: `services/jobs` (Go). The browser calls `/api/jobs/v1/...`; Traefik and the Vite proxy rewrite it to
`/api/v1/...` on the service. JSON is `snake_case`. Errors use the service's existing error envelope, with
`context.field` on validation errors. Every request is signed in through the gatekeeper (Keycloak); a person is a tenant, and row-level security keeps each tenant's rows from every other. The jobs a person adds are theirs; the catalog is everyone's.

The service holds three kinds of jobs: the **catalog** (empty for now), the jobs it **imports** from company job
boards (see Importing below), and the jobs a person **adds** from a link or pasted text. It reads job ads with rules
and a skills dictionary only, never with an LLM. Matching against the profile stays in the web app for now.

## Endpoints

| Method | Path (service side) | Body | Response |
| --- | --- | --- | --- |
| GET | `/api/v1/jobs` | query: `q`, `city`, `country`, `include_remote`, `area`, `domain`, `level`, `ids`, `limit` | 200 `{ items: [JobCard], total }` |
| GET | `/api/v1/jobs/{id}` | | 200 `Job` |
| POST | `/api/v1/jobs/read` | `{ url }` or `{ text, title?, company?, url? }` | 200 `JobDraft` |
| POST | `/api/v1/jobs` | `JobInput` | 201 `Job` |
| PUT | `/api/v1/jobs/{id}/requirements` | `{ must_have, nice_to_have }` | 200 `Job` |
| DELETE | `/api/v1/jobs/{id}` | | 204 |
| GET | `/api/v1/boards/of` | query: `url` | 200 `BoardOf`; 404 when the link is on no board Hoply reads |
| POST | `/api/v1/boards` | `{ url, company }` | 201 `Board` |

- `GET /jobs` returns the catalog, the imported jobs still open and the caller's own jobs (`added_by_me: true`),
  newest first (by `posted_at`, else `created_at`), at most `limit` (1–2000, 2000 by default); other people's added jobs
  are never visible. Filters: `q` matches title, company and requirements (case-insensitive); `city` matches the city
  (case- and diacritics-insensitive: `Iasi` finds `Iași`); `include_remote=true` also keeps remote jobs when `city` is
  set; `country` matches the country; `area` may repeat; `domain` may repeat, each one of `it`, `banking`, `accounting`,
  `finance`, `insurance`, `hr`, and keeps the jobs in any of them plus the caller's added jobs in none; `level` is one of
  the level values. `ids` (comma-separated, at
  most 200) returns those jobs only, closed ones included: Saved uses it. `total` is how many jobs the filters find,
  also when `items` stops at `limit`. Each item is a `JobCard`: a `Job` without the whole ad (`qualifications`,
  `bonuses`, `about_company`, `product`, `perks`, `read_from`), with its first 3 `responsibilities` and the first 300
  characters of `about`; `GET /jobs/{id}` has the whole ad. The response is gzip-compressed when the client accepts it.
- `GET /jobs/{id}` and `DELETE /jobs/{id}`: an unknown id, a non-UUID id or another person's job is 404; deleting a
  catalog job is 404 too.
- `PUT /jobs/{id}/requirements` replaces the requirements of a job the caller sees with the cv service's analysis of
  the ad, cleaned by the same rules as `POST /jobs`, and sets `requirements_from: analysis`. A catalog or imported
  job's ad is the same for everyone, so its analysis is shared. Another person's job is 404.
- `POST /jobs/read` saves nothing. It returns what the rules read, for the person to check and edit.
- `POST /boards` follows the board of a job link (company required, at most 120 characters; a link on no board Hoply
  reads is 422 on `url`). The importer reads followed boards with the listed ones, and imports a newly followed board
  at once in the background. Following a listed or followed board again returns it as it is and imports nothing. Followed boards are
  shared, like the jobs imported from them.

## Shapes

```text
Job       { id, title, company, city, country, work_mode, level, years_required,
            must_have: [string], nice_to_have: [string], responsibilities: [string],
            qualifications: [string], bonuses: [string],
            about, about_company, product, perks: [string], areas: [string], domains: [string],
            url, source, read_from, requirements_from, sample, added_by_me, created_at,
            origin, posted_at, closed_at }
JobInput  { title, company, city, country, work_mode, level, years_required, must_have, nice_to_have,
            responsibilities, qualifications?, bonuses?, about, about_company?, perks?, url, source, read_from }
JobDraft  JobInput + { host }
BoardOf   { platform, board, company, imported }
Board     { company, platform, board, followed_at }
```

| Field | Values |
| --- | --- |
| `work_mode` | `""`, `remote`, `hybrid`, `onsite` |
| `level` | `""`, `junior`, `mid`, `senior`, `lead` |
| `years_required` | `null` or 0–40 |
| `source` | `catalog`, `import`, `link`, `text` |
| `origin` | the job board platform of an imported job (`greenhouse`, `lever`, `lever-eu`, `ashby`, `workday`, `smartrecruiters`, `bamboohr`, `recruitee`, `workable`), else `""` |
| `posted_at` | when the board published an imported job, else `null` |
| `closed_at` | when the board stopped listing an imported job, else `null` |
| `read_from` | `ats_api` (the job board's public API, see below), `json_ld` (the page's schema.org `JobPosting`), `page_text` (the page's main text), `text` (pasted), `""` for catalog jobs |
| `requirements_from` | `rules` (read when the job was added) or `analysis` (Claude's reading of the ad, through the cv service) |
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
   | `boards.greenhouse.io/{board}/jobs/{id}`, `job-boards.greenhouse.io/{board}/jobs/{id}` (also their `.eu.` hosts), any URL with `gh_jid={id}` and a known board | `boards-api.greenhouse.io/v1/boards/{board}/jobs/{id}` |
   | `jobs.lever.co/{site}/{id}` (also `jobs.eu.lever.co`) | `api.lever.co/v0/postings/{site}/{id}` (`api.eu.lever.co` for the EU host) |
   | `jobs.ashbyhq.com/{board}/{id}` | `api.ashbyhq.com/posting-api/job-board/{board}`, the posting with that id |
   | `jobs.smartrecruiters.com/{company}/{id}` | `api.smartrecruiters.com/v1/companies/{company}/postings/{id}` |
   | `{company}.bamboohr.com/careers/{id}` | `{company}.bamboohr.com/careers/{id}/detail` |
   | `{company}.recruitee.com/o/{slug}` | `{company}.recruitee.com/api/offers/`, the offer with that slug |
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
  - about the company (`about_company`): `About us`, `About the company`, `About {company}`, `Company description`,
    `Who we are`, `Despre noi`, `Despre companie`
  - perks (`perks`): `Benefits`, `What we offer`, `Perks`, `Why you'll`, `Why join`, `Ce oferim`, `Beneficii`
  - ignored: `Our culture`, `Equal opportunity`, `Apply for this job`, `How to apply`

  Lines about the company and its perks feed no other field: a city or a skill named there is not the job's.
- Skills come from a dictionary with aliases (`K8s` → `Kubernetes`, `IaC` → `Infrastructure as Code`, `Postgres` →
  `PostgreSQL`, `golang` → `Go` and a case-sensitive `Go`, Romanian terms such as `rețele` → `Networking`). Skills found
  in the nice-to-have section are `nice_to_have`; skills found in the requirements and responsibilities sections are
  `must_have`; with no sections at all, every skill in the text is `must_have`.
- `responsibilities`, `qualifications`, `bonuses`, `perks`: the lines of the responsibilities, requirements,
  nice-to-have and perks sections, list markers removed, at most 30 each, each at most 400 characters. The job page
  shows `qualifications` as "What we look for" when an ad has them, else a line per skill.
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
  `about_company`: the lines of the about-the-company section, joined, at most 4000 characters.

## Catalog

Migration `00100_jobs.sql` creates the table and seeds nothing, so the catalog is empty; Find jobs shows the imported
jobs. The sample jobs exist only in the web app's test doubles.

## Importing

With `JOBS_IMPORT=on` (Docker Compose sets it), the service reads the boards listed in
`services/jobs/internal/importer/boards.yaml` at start, then every `JOBS_IMPORT_INTERVAL` (6 h by default, at least
1 m), one board after another with a second between them. A Postgres advisory lock keeps it to one replica.

| Platform | List call (public, no key) |
| --- | --- |
| `greenhouse` | `boards-api.greenhouse.io/v1/boards/{board}/jobs?content=true` (EU boards answer here too) |
| `lever`, `lever-eu` | `api.lever.co/v0/postings/{board}?mode=json` (`api.eu.lever.co` for `lever-eu`) |
| `ashby` | `api.ashbyhq.com/posting-api/job-board/{board}` (unlisted jobs skipped) |
| `smartrecruiters` | `api.smartrecruiters.com/v1/companies/{board}/postings?country=ro`, 100 a page; each wanted job's description from `/postings/{id}` |
| `bamboohr` | `{board}.bamboohr.com/careers/list`, the call the careers page makes (not a documented API), with each job's city and state; each wanted job's description, country and posting date from `/careers/{id}/detail` |
| `recruitee` | `{board}.recruitee.com/api/offers/`: every published offer with its description and requirements |
| `workable` | `apply.workable.com/api/v1/widget/accounts/{board}?details=true`, the widget its careers pages embed: every job with its description |
| `anofm` | ANOFM's job matching portal: `POST mediere.anofm.ro/api/entity/vw_public_job_posting` with the county's id, 100 a page, the call its public search makes (not a documented API); a wanted job's responsibilities, requirements and terms from its page `/app/module/mediere/job/{id}`, without the contact person, email, phone or address. A board is a county code (`IS`); each job keeps its own employer, and its level comes from the experience asked |
| `workday` | `POST {host}/wday/cxs/{tenant}/{site}/jobs`, the call the career site makes (not a documented API): first to find the site's country facet named Romania (sites name it differently), else the values of its location facet that are in Romania ("Bucharest - Dacia One"), then 20 a page with them; a site with neither lists none; each wanted job's description from `{host}/wday/cxs/{tenant}/{site}{externalPath}` |

- A job is kept when it is based in Romania (a Romanian city or the word Romania in any of its locations, the
  country code `RO`, or a list the platform kept to Romania) and its title names one of the domains (`extract.Classify`,
  on English and Romanian titles and shared-service process codes). A job's `domains` and `areas`:

  | Domain | Areas |
  | --- | --- |
  | `it` | `DevOps`, `Backend`, `Frontend`, `Full stack`, `Mobile`, `Data`, `AI`, `QA`, `Security`, `Embedded`, `Sysadmin`, `Product`, `Design` |
  | `banking` | `Front office`, `Lending`, `Bank operations`, `Corporate banking` |
  | `accounting` | `General accounting`, `Payables`, `Receivables`, `Payroll`, `Tax` |
  | `finance` | `FP&A & controlling`, `Treasury`, `Risk & compliance`, `Audit` |
  | `insurance` | `Underwriting`, `Claims`, `Insurance sales`, `Actuarial` |
  | `hr` | `Recruitment`, `HR generalist`, `Compensation & benefits`, `Training & development`, `HR administration` |

  A title is in IT when it names an IT area (a generic developer title is `Backend` unless it names a narrower area; an
  "administrator" title in another domain is not `Sysadmin`); in another domain when it names the domain, and then in
  the areas it names. An accounting or HR title that names none of its areas is `General accounting` or `HR generalist`;
  in the other domains it has the domain and no area. Payroll titles are in both `accounting` and `hr`. A job a person
  adds gets its domains and areas from its title the same way. Where the list has no descriptions,
  only those jobs are read, a quarter of a second apart, and a description read in the last 7 days is not read again
  (the job is only marked as still listed). Its text goes through the same rules as a link; the company is the one in `boards.yaml`.
- A job is keyed by its platform, its board and the platform's id: a later run updates it and keeps requirements Claude's
  analysis wrote. A job the board no longer lists is closed (`closed_at`); listed again, it opens again. A board that
  answers with no jobs, whose list stopped at its cap (Workday 400, SmartRecruiters 500), or whose
  descriptions could not all be read, closes nothing. A run that read the followed
  boards also closes the jobs of every board no longer listed or followed.
- A board that fails is logged and skipped; the others still run. Answers are capped at 32 MB and 30 s.

