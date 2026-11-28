# Profile API

Service: `services/profile` (Go). The browser calls `/api/profile/v1/...`; Traefik and the Vite proxy rewrite it to
`/api/v1/...` on the service. JSON is `snake_case`. Errors use the service's existing error envelope.

Every request is signed in through the gatekeeper (Keycloak). A person is a tenant, so "me" is the one profile of
the caller's tenant, kept from every other tenant by row-level security. The profile is created empty on the first
`GET /me`.

## Endpoints

| Method | Path (service side) | Body | Response |
| --- | --- | --- | --- |
| GET | `/api/v1/profiles/me` | | 200 `Profile` |
| PATCH | `/api/v1/profiles/me` | `ProfilePatch` | 200 `Profile` |
| PUT | `/api/v1/profiles/me/skills` | `{ "skills": [string] }` | 200 `Profile` |
| POST | `/api/v1/profiles/me/experience` | `ExperienceInput` | 201 `Experience` |
| PUT | `/api/v1/profiles/me/experience/{id}` | `ExperienceInput` | 200 `Experience` |
| DELETE | `/api/v1/profiles/me/experience/{id}` | | 204 |
| POST | `/api/v1/profiles/me/projects` | `ProjectInput` | 201 `Project` |
| PUT | `/api/v1/profiles/me/projects/{id}` | `ProjectInput` | 200 `Project` |
| DELETE | `/api/v1/profiles/me/projects/{id}` | | 204 |
| POST | `/api/v1/profiles/me/education` | `EducationInput` | 201 `Education` |
| PUT | `/api/v1/profiles/me/education/{id}` | `EducationInput` | 200 `Education` |
| DELETE | `/api/v1/profiles/me/education/{id}` | | 204 |
| POST | `/api/v1/profiles/me/certifications` | `CertificationInput` | 201 `Certification` |
| PUT | `/api/v1/profiles/me/certifications/{id}` | `CertificationInput` | 200 `Certification` |
| DELETE | `/api/v1/profiles/me/certifications/{id}` | | 204 |
| POST | `/api/v1/profiles/me/languages` | `LanguageInput` | 201 `Language` |
| DELETE | `/api/v1/profiles/me/languages/{id}` | | 204 |

An unknown `{id}`, or one belonging to another profile, is 404. PUT is a full replace and keeps the entry's place.
PUT and DELETE never create the profile; PATCH, PUT skills and POST do.

## Shapes

```text
Profile            { id, name, headline, about, email, phone, github, linkedin,
                     city, country, work_modes, open_to_relocation,
                     work_authorization, work_authorization_region, notice_period_weeks, available_from,
                     target_roles, target_seniority, employment_types, salary_min, salary_currency, salary_period,
                     app_language, domains: [string], additional_info: [string], notes: [Note],
                     skills: [string], experience: [Experience], projects: [Project], education: [Education],
                     certifications: [Certification], languages: [Language], created_at, updated_at }
ProfilePatch       every Profile field above from name to notes, each optional: only the fields sent change;
                   an array replaces the stored one; null clears notice_period_weeks or salary_min
Bullet             { text, result, skills: [string] }
Note               { text, skills: [string] }
Experience         { id, role, company, location, employment_type, industry, team_size, people_managed,
                     start, end, tools: [string], bullets: [Bullet], notes: [Note] }
ExperienceInput    Experience without id; `notes` left out keeps the stored notes
Project            { id, name, role, url, start, end, tools: [string], bullets: [Bullet], notes: [Note] }
ProjectInput       Project without id; `notes` left out keeps the stored notes
Education          { id, level, field, degree, school, start_year, end_year }
EducationInput     Education without id
Certification      { id, name, issuer, issued, expires, credential_id, url }
CertificationInput Certification without id
Language           { id, name, level }
LanguageInput      { name, level }
```

Strings are never null: an unset string is `""`, an unset array is `[]`. `notice_period_weeks`, `salary_min`,
`team_size` and `people_managed` are a number or `null`. An empty `end` (experience, project) or `end_year`
(education) means "ongoing".

## Rules

| Field | Rule (422 otherwise, with `context.field` naming it) |
| --- | --- |
| `name` (profile), `headline` | at most 120 characters |
| `about` | at most 280 characters |
| `email` | empty or a valid address |
| `github`, `linkedin`, `phone`, every `url` | at most 200 characters |
| `city`, `country`, `work_authorization_region`, `industry` | at most 60 characters |
| `work_modes` | each one of `remote`, `hybrid`, `onsite`; duplicates dropped |
| `open_to_relocation` | boolean |
| `work_authorization` | empty or one of `citizen`, `permanent_resident`, `work_permit`, `needs_sponsorship` |
| `notice_period_weeks` | `null` or an integer from 0 to 52 |
| `available_from` | empty or a month |
| `target_roles` | same cleaning as `skills`, at most 10, each at most 120 characters |
| `target_seniority` | each one of `intern`, `junior`, `mid`, `senior`, `lead`, `principal`; duplicates dropped |
| `employment_types` | each one of `full_time`, `part_time`, `contract`, `freelance`, `internship`; duplicates dropped |
| `employment_type` | empty or one of the `employment_types` values |
| `salary_min` | `null` or an integer from 0 to 100000000 |
| `salary_currency` | empty or three letters, stored upper-case |
| `salary_period` | empty or one of `year`, `month`, `hour` |
| `app_language` | empty (not picked yet) or one of `en`, `ro`: the language the person uses the app in |
| `domains` | each one of `it`, `banking`, `accounting`, `finance`, `insurance`, `hr` (the jobs service's domains); duplicates dropped. Empty until the person is through onboarding |
| `additional_info` | trimmed, empty entries dropped, at most 10, each at most 300 characters |
| `skills` | trimmed, empty entries dropped, duplicates dropped case-insensitively (first spelling kept), at most 50, each at most 60 characters |
| `role` (experience), `name` (project, certification, language) | required, at most 120 characters |
| `role` (project), `company`, `location`, `school`, `field`, `degree`, `issuer`, `credential_id` | at most 120 characters |
| `team_size`, `people_managed` | `null` or an integer from 0 to 100000 |
| `start`, `end`, `issued`, `expires` | empty or a month `YYYY-MM` (year 1950–2100); `end` not before `start`, `expires` not before `issued` |
| `start_year`, `end_year` | empty or four digits (1950–2100); `end_year` not before `start_year` |
| `level` (education) | one of `high_school`, `associate`, `bachelor`, `master`, `doctorate`, `other` |
| `tools`, `Bullet.skills` | same cleaning as `skills`; `tools` at most 30, `Bullet.skills` at most 10 |
| `bullets` | at most 10; entries with an empty `text` dropped; `text` at most 300, `result` at most 200 characters |
| `notes` | at most 30; entries with an empty `text` dropped; `text` at most 1000 characters; `Note.skills` cleaned as `skills`, at most 10 |
| `level` (language) | one of `A1`, `A2`, `B1`, `B2`, `C1`, `C2`, `Native` |

`Bullet.result` is the measurable outcome ("cut release time from 2 hours to 15 minutes"); `Bullet.skills` tags the
skills the bullet proves, so a confirmed fact from Prepare CV is remembered under the skill it answered.

## Order

`skills`, `tools`, arrays of enums and `bullets` keep the order sent. `experience` and `projects` are newest-added
first. `education`, `certifications` and `languages` keep the order they were added in.

## Migration of existing data (`00101`)

| Before | After |
| --- | --- |
| experience `start_year`, `end_year` | `start`, `end` = `YYYY-01` (or `""`) |
| experience `bullets: [string]` | `[{ text, result: "", skills: [] }]` |
| education `degree` | kept as `degree`; `level` guessed from it (`BSc`/`Bachelor`/`Licen` → `bachelor`, `MSc`/`Master` → `master`, `PhD`/`Doctor` → `doctorate`, otherwise `other`); `field` empty |
| education `years` ("2016 – 2020") | `start_year`, `end_year` from the first and last four-digit numbers, else empty |

## Notes

Highlights (`bullets`) are the person's own CV lines. Notes are what they tell Prepare your CV about a role, a
project (`experience[].notes`, `projects[].notes`) or themselves (`notes`), in their own words and as long as they
like. The cv service gives both to Claude, which writes each CV's bullets from them; a note is never printed as it is.
The skills of a note count as skills the person shows.
