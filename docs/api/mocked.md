# Mocked services

None. Every service the web app calls is real: profile (`docs/api/profile.md`), jobs (`docs/api/jobs.md`),
applications (`docs/api/applications.md`) and cv (`docs/api/cv.md`).

MSW stays in the web app for a service while it is being built. Its handlers go in
`apps/frontend/src/custom/mocks/handlers.ts`, and they run on the Vite dev server only
(`VITE_MSW_ENABLED=true` in `apps/frontend/.env.development`). A production build has neither the handlers nor
the worker script.

The tests have their own in-memory services in `apps/frontend/src/test/`: `profile-service.ts`,
`jobs-service.ts`, `applications-service.ts` and `cv-service.ts`. Vitest serves them through MSW, and the
Playwright suite through `page.route` (`apps/frontend/tests/e2e/`). `cv-service.ts` writes CVs with the rules
engine the web app used before the cv service existed (`src/test/cv-engine/`). The Go engine in `services/cv` is
its port, and the Go tests hold the same cases.
