<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/images/logo-dark.svg">
    <img src="docs/images/logo-light.svg" alt="hoply" width="264">
  </picture>
</p>

<p align="center"><strong>Job search with a CV tailored to each job, written only from what you have done.</strong></p>

<p align="center">
  <a href="https://github.com/CosminMilica/hoply/actions/workflows/ci.yml"><img src="https://github.com/CosminMilica/hoply/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/CosminMilica/hoply/actions/workflows/quality.yml"><img src="https://github.com/CosminMilica/hoply/actions/workflows/quality.yml/badge.svg" alt="Quality gates"></a>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go 1.27">
  <img src="https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs&logoColor=white" alt="Vue 3">
  <img src="https://img.shields.io/badge/CV%20writer-Claude-2A4BFF" alt="CV writer: Claude">
</p>

---

Most CVs are sent as they are, or stuffed with keywords. Hoply does what a good career coach would: it reads the
job ad, works out what the company really needs, picks the strongest proof from your experience, and writes a
one-page CV around it, in the company's own words. It never invents a skill, a number or a result.

## What it does

| | |
| --- | --- |
| **Profile** | Everything a recruiter or a screening system asks for: roles with highlights and results, projects, education, certifications, languages, work preferences. |
| **Add a job** | Paste a link or the ad's text. Hoply reads it, from the public APIs of Greenhouse, Lever, Ashby, SmartRecruiters and Workable, from structured data, or from the page itself, and finds the requirements, the responsibilities, the place and the level. |
| **Match** | Every requirement is matched, close or missing. Matching understands other names and umbrella terms (GitHub Actions proves CI/CD) and tools of the same kind (FluxCD and Argo CD are both GitOps). |
| **Prepare your CV** | Asks only what the job needs and your profile does not show yet. Your answers become notes that the CV is written from. |
| **Tailored CV** | Claude analyzes the job, plans the evidence for each of its priorities, writes the CV and then reviews it as the hiring manager. Every sentence cites the facts it rests on, and code checks them. |
| **ATS check** | Twelve checks a screening system cares about: keywords in sentences, mirrored responsibilities, action verbs, numbers, dates, length. |
| **Cover letter and messages** | From the tailored CV: a cover letter, a LinkedIn note, a message for after you apply and a follow-up a week later, checked against your facts like the CV, editable, and exported in the CV's template. |
| **Export** | A one-column PDF and a Word DOCX with standard headings and real text, readable by any applicant tracking system. |

## How a CV is written

```mermaid
flowchart LR
    ad[Job ad] --> analyze["1. Analyze<br/>priorities, real duties,<br/>traits, terms"]
    profile[Your profile<br/>and notes] --> facts[Numbered facts]
    analyze --> write["2. Plan and write<br/>evidence per priority,<br/>positioning, CV"]
    facts --> write
    write --> review["3. Review<br/>as the hiring manager"]
    review --> check{"Every sentence<br/>checked against<br/>its facts"}
    check -->|passes| cv[Tailored CV]
    check -->|fails| rules[Rewritten from<br/>the cited fact]
    rules --> cv
    cv --> files[PDF · DOCX]
```

A sentence is thrown away when it names a number its facts do not hold, a requirement you do not have, or the
company you are applying to as your employer. A tool you have not used is never named: when the job asks for Argo
CD and you use FluxCD, the CV shows the kind of work instead ("GitOps delivery with FluxCD"). Without Claude, a rules
engine writes the CV, and the page says so. The rules behind the prompts, and their sources, are in
[docs/cv-writing.md](docs/cv-writing.md).

## Architecture

```mermaid
flowchart LR
    web[Web app<br/>Vue 3 · Nuxt UI] --> traefik[Traefik]
    traefik --> profile[profile]
    traefik --> jobs[jobs]
    traefik --> applications[applications]
    traefik --> cv[cv]
    cv --> profile
    cv --> jobs
    cv --> claude[Claude Code CLI]
    cv --> gotenberg[Gotenberg<br/>PDF]
    profile & jobs & applications & cv --> pg[(PostgreSQL)]
```

| Service | Owns | Contract |
| --- | --- | --- |
| `profile` | the person's profile and notes | [docs/api/profile.md](docs/api/profile.md) |
| `jobs` | the jobs a person adds, and reading ads from links and text | [docs/api/jobs.md](docs/api/jobs.md) |
| `applications` | saved, CV ready, applied, skipped | [docs/api/applications.md](docs/api/applications.md) |
| `cv` | questions, job analyses, CVs written asynchronously, PDF and DOCX | [docs/api/cv.md](docs/api/cv.md) |

The services are Go 1.27 with chi, pgx and goose migrations. The web app is Vue 3 with Vite, Nuxt UI, TanStack Query
and zod. The project is generated and kept up to date with [forge](https://github.com/cchifor/forge)
(`deploy/forge/stack.yaml`).

## Run it

You need Docker and, for development, Go 1.27 and Node 26.

```bash
git clone https://github.com/CosminMilica/hoply.git
cd hoply
docker compose -f docker-compose.yml -f docker-compose.dev.yaml up --build -d --wait
```

Open http://app.localhost. Without a Claude token the rules engine writes the CVs.

### Sign in

Keycloak (http://auth.localhost, admin console at http://auth.localhost/admin, `admin` / `admin` locally) holds the
accounts, and the gatekeeper signs the web app in. A person is a Keycloak organization of their own: its id is their
tenant, and row-level security keeps every tenant's profile, saved jobs and CVs from the others. The realm starts
with the organization `hoply` (tenant `00000000-0000-0000-0000-000000000001`) and its user `dev@localhost` /
`devpass`. There is no sign-up: add a person like this.

```bash
kc() { docker compose exec -T keycloak /opt/keycloak/bin/kcadm.sh "$@"; }
kc config credentials --server http://localhost:8080 --realm master --user admin --password admin
org=$(kc create organizations -r hoply -s name=ana -s alias=ana -s enabled=true -i)
user=$(kc create users -r hoply -s username=ana@example.com -s email=ana@example.com -s emailVerified=true \
  -s enabled=true -s firstName=Ana -s lastName=Pop -i)
kc set-password -r hoply --userid "$user" --new-password 'a-first-password' --temporary
kc create "organizations/$org/members" -r hoply -b "\"$user\""
```

### Connect Claude

Hoply runs Claude through the Claude Code CLI with a subscription token, the way you use Claude yourself:

```bash
npx -y @anthropic-ai/claude-code@2.1.322 setup-token
(umask 077 && pbpaste > deploy/compose/secrets/claude_oauth_token)
curl -s http://app.localhost/api/cv/v1/writer    # "state": "ok"
```

The token file is gitignored and read on every CV, so nothing needs a restart. A subscription token is for its
owner's own use: before anyone else uses Hoply, switch to an API key. See [docs/claude-setup.md](docs/claude-setup.md).

## Develop

```text
apps/frontend/        the web app (features: profile, jobs, applications, cv)
services/profile/     Go service: the profile
services/jobs/        Go service: jobs and reading ads
services/applications/ Go service: what you did with each job
services/cv/          Go service: questions, analysis, writing, checks, export
  internal/grounding/prompts/   the prompts and the answer schemas
deploy/               compose, the forge stack, secrets (gitignored)
docs/                 contracts, plans, design, CV writing rules
scripts/forge.sh      runs the pinned forge build
```

Every change passes the same gates locally and in CI: architecture, unit, integration and end-to-end suites in
every service and in the web app, and coverage above 80% on the changed lines.

```bash
scripts/forge.sh --quality architecture --project-path .
scripts/forge.sh --quality test --project-path .
scripts/forge.sh --quality coverage --project-path . --base-ref origin/main
```

Tests never call Claude: a fake CLI returns recorded answers, including the real envelopes of the pinned CLI
version in `services/cv/tests/testdata/claude/`.

## Status

| Done | Next |
| --- | --- |
| Profile, adding jobs from links and text, matching, saved jobs and applications | Search by meaning, and ranking by Claude's reading of each ad |
| Finding jobs in Romania from the job boards of 80 companies (Greenhouse, Lever, Ashby, Workday, SmartRecruiters), and of the companies people follow | |
| A Send to Hoply bookmarklet that brings a job from LinkedIn, eJobs or any site into Add a job | |
| CVs tailored by Claude in three steps, checked sentence by sentence | An evaluation set to tune the prompts on more people and jobs |
| PDF and DOCX export, ATS checks | CVs in Romanian |

The design lives in [docs/design/](docs/design/), and the plan for the cv service in
[docs/plans/2026-11-25-cv-service.md](docs/plans/2026-11-25-cv-service.md).

## Licence

Free for personal and other noncommercial use, under the [PolyForm Noncommercial License 1.0.0](LICENSE). Commercial
use needs a separate agreement with the author. Code generated by forge and third-party packages keep their own
licences: see [NOTICE.md](NOTICE.md).
