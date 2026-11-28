# crashr

crashr - a Vue 3 SPA on forge's application shell.

## Tech Stack

- **Framework**: Vue 3 + TypeScript + Vite
- **UI**: Nuxt UI 4 (Reka UI + Tailwind CSS 4)
- **State**: Pinia (client) + TanStack Vue Query (server)
- **HTTP**: Ky + Zod runtime validation
- **Testing**: Vitest + Playwright

## Application shell

forge renders the shell, the menus and the routing contracts from the project's configuration (`frontend.layouts`, `frontend.routing`, `frontend.auth`, `frontend.extras`) into `src/shared/application/`, `src/shared/layouts/` and `src/shared/system/`. Those files are regenerated, never edited: change the configuration and run `forge --update`. The application's own pages live in `src/features/`; compose their routes, layout wrappers and menu actions in `src/app/router/index.ts`.

## Scripts

| Script | Description |
|---|---|
| `npm run dev` | Start dev server |
| `npm run build` | Production build + type-check |
| `npm run test` | Run unit tests with Vitest |
| `npm run lint` | Lint with ESLint |
| `npm run type-check` | TypeScript type checking |

## Getting Started

```bash
cd frontend
npm install
npm run dev
```

## Hot reload in Docker Compose

This app also runs as a Vite dev server in the project's Compose stack, behind the same
Traefik router and host as the built app. From the project root:

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yaml up --build --watch
```

- Needs Docker Compose 2.39.4 or newer.
- Edits under `apps/frontend` sync into the container; a change to `package.json` or the
  lockfile rebuilds the image.
- No content-security policy applies and nginx is not in the path, so check security
  headers on the built image (`docker compose up --build`).
- The generated stack does not route the sign-in paths (`/auth/*`) on any frontend. With auth
  on, the app redirects to `/auth/login` as it does in the built stack; generate with auth off
  for a hot-reload loop without sign-in, or route the gatekeeper in your own Compose overlay.
- Turn it off with `dev_server: false` and `forge --update`.
