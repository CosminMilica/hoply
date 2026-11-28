# Job sources

Status: research, 2026-11-25. Endpoints marked "to verify" come from third-party descriptions, not the vendor's docs.

Where the `jobs` service can get real jobs, and on what terms. "Full text" means the full ad, which matching and
Prepare CV need; a snippet is not enough.

## 1. Company job boards (ATS public APIs): the main source

One call per company, no key, full text. We keep a list of companies hiring in Romania or remote in the EU.

| Platform | Endpoint | Full text | Notes |
| --- | --- | --- | --- |
| Greenhouse | `boards-api.greenhouse.io/v1/boards/{token}/jobs?content=true` | yes | official, public |
| Lever | `api.lever.co/v0/postings/{site}?mode=json` | yes | official, public; about 1 request/s |
| Ashby | `api.ashbyhq.com/posting-api/job-board/{board}` | yes | official, public; `includeCompensation=true` for salary |
| SmartRecruiters | `api.smartrecruiters.com/v1/companies/{id}/postings` | per job | the list has no description; one more call per job. Common with large employers in Romania |
| BambooHR | `{company}.bamboohr.com/careers/list` | per job | the JSON its careers page loads (not a documented API); the list has city and state, `/careers/{id}/detail` the description and country. Used by smaller companies |
| Workable | `apply.workable.com/api/v1/widget/accounts/{slug}?details=true` | yes | the public widget endpoint, descriptions included |
| Recruitee | `{slug}.recruitee.com/api/offers/` | yes | keyless, every published offer with its description and requirements; used by Romanian banks and shared-service centres |
| Personio | `{company}.jobs.personio.de/xml?language=en` | yes | official XML feed; mostly DACH companies |
| Teamtailor | career site feed | to verify | used by Romanian companies (it publishes to eJobs) |
| Any career page | schema.org `JobPosting` in the page (and its sitemap) | yes | the link reader we already build |

## 2. Aggregators with an API

| Source | Access | Full text | Notes |
| --- | --- | --- | --- |
| Jooble | key, by form on `jooble.org/api/about` | snippet | has a Romanian site; coverage to confirm with the key |
| Careerjet | affiliate ID (`careerjet.com/partners`) | snippet | every call needs the end user's IP and user agent, so live search only, no background import |
| Adzuna | key | snippet | Romania not confirmed in its country list |
| Arbeitnow | none | yes | free; mostly Germany and remote Europe |

For a snippet-only source, we open the job's own page with the link reader to get the full text.

## 3. Remote-job boards

| Source | Access | Use |
| --- | --- | --- |
| Remote OK | public API, 24 h delay | yes, with attribution and a link to the original |
| Jobicy | public API, up to 200 per call | yes, with a link to the original |
| We Work Remotely | RSS | yes, after checking its terms |
| Hacker News "Who is hiring" | Algolia API | yes; free text, mostly remote and US |
| Remotive | public API | no: its terms forbid passing jobs to other services, about 4 calls a day |
| Himalayas | public API | no: its terms forbid aggregation and commercial reuse |

## 4. Romanian job boards: link or paste only

| Source | Why not imported |
| --- | --- |
| eJobs.ro | no public API (partner tokens are for posting jobs); scraping breaks its terms |
| BestJobs, Hipo.ro, OLX Locuri de muncă | no public API or feed found |
| LinkedIn | no public API; job pages need a login |

People bring these jobs in with **Add a job** (link or text).

## 5. Not now

| Source | Why |
| --- | --- |
| EURES | data exchange for EURES members and partners, not a public API; portal terms are personal use |
| ANOFM (Romania's public employment agency) | no API found; to check |
| Google for Jobs via SerpAPI, paid job datasets (TheirStack, Coresignal and similar) | paid; consider once the free sources are not enough |

## Plan

See [plans/2026-11-26-find-jobs.md](plans/2026-11-26-find-jobs.md).
