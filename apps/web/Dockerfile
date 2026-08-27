FROM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS build
WORKDIR /repo

RUN npm install --global pnpm@9.15.0
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/package.json
COPY scripts/install-hooks scripts/install-hooks
RUN pnpm install --frozen-lockfile

COPY contracts/ contracts/
COPY apps/web/ apps/web/
ARG VITE_APP_NAME="Micro1 Template"
ARG VITE_API_BASE_URL="http://localhost:8080/api/v1"
ARG VITE_API_TIMEOUT_MS="10000"
ENV VITE_APP_NAME=$VITE_APP_NAME \
    VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_API_TIMEOUT_MS=$VITE_API_TIMEOUT_MS
RUN pnpm --filter @micro1/web build && pnpm --filter @micro1/web bundle:check

FROM caddy:2.10-alpine@sha256:4c6e91c6ed0e2fa03efd5b44747b625fec79bc9cd06ac5235a779726618e530d AS runtime
RUN mkdir -p /config/caddy /data/caddy && chown -R 1000:1000 /config /data
COPY apps/web/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /repo/apps/web/dist /srv
USER 1000:1000
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1:8080/health/live || exit 1
