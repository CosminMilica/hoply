# Plan: users, finance jobs and Romanian CVs

Status: phases 1 to 3 built, 2026-11-28 (phase 2 on feat/finance-jobs, phase 3 on feat/romanian-cv).

Hoply has one profile, finds tech jobs only and writes CVs in English. This plan lets a second person use it: Keycloak
is the identity manager and each person is a tenant of their own, kept apart by Postgres row-level security as in
kartra-ai; Find jobs adds banking and finance areas; a CV can be written in Romanian.

## Decisions

| Question | Proposal |
| --- | --- |
| Identity | Keycloak as in kartra-ai: `include_keycloak`, realm `hoply`, client `hoply-web`; `auth.provider: gatekeeper`; the web app signs in through forge (`include_auth`, `auth.runtime: forge`, a `sign_in` route); accounts are made in Keycloak by the admin |
| Separation | `database.multitenancy: shared_rls`, `database.tenant_resolution: token_claim`: every person is a tenant (a Keycloak organization), rows belong to the tenant and `subject` records who wrote them; row-level security keeps profiles, saved jobs, skips, CVs and added jobs from everyone else; imported jobs stay one shared catalog |
| Existing data | Stays where it is: the realm's organization `hoply` has the old dev tenant's id, so the owner signs in to it |
| Service calls | `auth.service_discovery: true`, as kartra-ai: cv calls profile and jobs with the gatekeeper's `client_credentials` tokens for the tenant it works for (the caller's, or the CV's in the worker) |
| Claude | One Claude for everyone: the owner's subscription token writes every person's CVs (decided 2026-11-27, replacing the owner-only rule of the cv plan) |
| New areas | Banking, Accounting, Finance & control, Risk & compliance, from English and Romanian titles |
| CV language | Per CV: English or Romanian, picked on Prepare your CV; Romanian by default for a Romanian ad |

## Phases

| Phase | What | Done when |
| --- | --- | --- |
| 1. Users | Keycloak, sign-in in the web app, RLS per tenant, service credentials for the cv worker, the data move | Two people sign in and see only their own profile, saved jobs and CVs, also when one asks for the other's ids |
| 2. Finance jobs | The four areas (title rules in both languages), finance skills in the dictionary, area groups in Find jobs, finance employers' boards | Find jobs lists banking and accounting jobs in Iași |
| 3. Romanian CVs | `language` on a CV; Romanian prompts, writing rules, PDF labels and letter; eval cases in Romanian | A Romanian CV passes the same ATS and grounding checks as an English one |

Each phase is its own pull request.

## Phase 1: what the forge dry run showed

| Part | Forge gives | Hoply adds |
| --- | --- | --- |
| Options | read from `stack.yaml`, `.forge/quality.json` and `[forge.options]` in `forge.toml`: all three change | `session_dialog: project`, as kartra-ai |
| Gatekeeper, Keycloak realm, Traefik `auth` middleware, sign-in page | generated | merge into our compose (jobs gzip, importer switch) |
| Go token check (`platform_auth_go_middleware`) | generated, injected into `app.go` and `main.go` | merge: our `app.go` and `main.go` conflict in all four services |
| RLS (`multitenancy_rls_go`) | a policy on the scaffold `items` table only, a `tenantScope` that binds `app.current_tenant` per transaction, a startup guard against owner or BYPASSRLS roles | a policy per table, every repository in a tenant transaction, a non-owner app role |
| Shared catalog | none | jobs policies per command: everyone reads the catalog; without a tenant (the importer) only catalog rows are written; a tenant adds and deletes only its own jobs and may update a catalog job (a shared analysis) |
| cv worker | none | `cv_claim`, `cv_recover` and `cv_tenant` (SECURITY DEFINER) see the whole queue; each CV is changed inside its tenant |
| Browser routes | none | Traefik sends `/auth/login`, `/auth/userinfo`, `/auth/session`, `/callback` and `/logout` on `app.localhost` to the gatekeeper; Keycloak answers at `auth.localhost` (`KC_HOSTNAME`), as kartra-ai |

## Finance employers in Iași (checked 2026-11-27)

| Where | Employers | For phase 2 |
| --- | --- | --- |
| Workday | Genpact: 79 Iași jobs, almost all finance (AP, I2C, R2R, controllership) | add the board; its titles are grade codes ("Specialist - F&A - AP 4A"), so the area rules read F&A, AP, AR, I2C, R2R |
| Recruitee | Veo, tbi bank, Mobilexpense | a `recruitee` platform |
| Workable | Beyond (Qodea) | a `workable` platform (links are read already) |
| SuccessFactors, Taleo, own sites | Capgemini, BCR, BRD, Hornbach, Banca Transilvania, Raiffeisen, CEC | not readable; Add a job from the link or text |

Shared-service ads are in English, often asking for a second language (German first); bank ads are in Romanian
("Consilier financiar bancar", "Contabil", "Referent financiar").

## Shape

```text
deploy/forge/stack.yaml            include_keycloak, keycloak realm and client, auth.provider gatekeeper,
                                   auth.service_discovery, database.multitenancy shared_rls,
                                   database.tenant_resolution token_claim; frontend include_auth and sign_in
services/*/migrations/             one migration per service: the dev subject becomes the owner's
services/jobs/internal/extract/    areas.go (four areas), skills.json (finance skills)
services/jobs/internal/importer/   boards.yaml (finance employers)
services/cv/internal/cvdoc/        language-aware rules: stock words and letter clichés per language
services/cv/prompts/               Romanian prompts beside the English ones
apps/frontend/src/features/jobs/   area groups (Tech, Finance) in the filters
apps/frontend/src/features/cv/     the language choice on Prepare your CV
docs/cv-writing.md                 the Romanian rules and their sources
```
