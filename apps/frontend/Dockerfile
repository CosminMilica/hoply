# -- Dependencies stage --
FROM node:26-slim AS deps
WORKDIR /app


COPY package.json package-lock.json* ./
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi

# The dev server writes Vite's cache and its bundled config into node_modules
# at startup; those directories (and /app) belong to the unprivileged user, the
# installed packages stay root's.
RUN mkdir -p node_modules/.vite node_modules/.vite-temp \
 && chown node:node /app node_modules node_modules/.vite node_modules/.vite-temp

# -- Source stage --
# The one place the application is copied, so the dev and production images
# cannot differ in what they hold.
FROM deps AS source
COPY --chown=node:node . .

# -- Dev stage --
# docker-compose.dev.yaml builds this target; sync keeps /app in step with the
# project (docker compose up --watch).
FROM source AS dev
USER node
EXPOSE 5173
# The vite binary, not the project's `dev` script: a flag in that script (--port, --host)
# would win over the wrapper's settings and leave the router with nothing to reach.
CMD ["node_modules/.bin/vite", "--host", "0.0.0.0", "--config", "vite.dev.config.ts"]

# -- Build stage --
FROM source AS builder
RUN npm run build

# -- Runtime stage --
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
# nginx.conf is NOT baked into the image — it is supplied at runtime: a
# bind-mount in docker-compose, a ConfigMap in the Helm chart. This keeps the
# routing config a deployment concern (overridable per environment) rather than
# a build-time constant.

EXPOSE 80
