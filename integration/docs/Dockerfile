# Multi-stage build for the new-api-docs-v1 (Atius-branded) documentation site.
#
# Stage 1: install deps + run `prebuild` (generates MDX from content/) +
#            `next build` (compiles the Next.js app).
# Stage 2: minimal standalone Next.js runtime (output: 'standalone' from
#            next.config.mjs produces a self-contained server in .next/standalone).
#
# Why standalone: same pattern as router-ai-atius-login. ~150MB image
# instead of ~600MB. Includes only the files Next.js needs at runtime.
#
# The content/ directory (MDX docs) is 5.9MB and the public/ assets
# are 68MB — these are baked into the image. To update docs, rebuild.
#
# Build context: integration/docs/ (the cloned new-api-docs-v1 repo).
# See podman-compose service `router-ai-atius-docs` for the runtime.

# ---- Stage 1: deps + build ----
# Bun is the package manager (matches web/default + router-ai-atius-login
# in this project). node:20-bookworm-slim ships with unzip + the
# C toolchain Bun needs to bootstrap.
FROM docker.io/library/node:20-bookworm-slim AS builder

# Install Bun (the official install script uses $HOME/.bun, NOT
# /usr/local/bun which is what the older docs say).
ENV BUN_INSTALL=/root/.bun
RUN apt-get update && apt-get install -y --no-install-recommends curl unzip ca-certificates \
    && curl -fsSL https://bun.sh/install | bash \
    && rm -rf /var/lib/apt/lists/*
ENV PATH="/root/.bun/bin:${PATH}"

WORKDIR /app

# Install deps with bun (lockfile-friendly, faster than npm).
# Use --ignore-scripts because fumadocs-mdx postinstall tries to import
# ESM modules with relative paths that fail in a clean container — the
# same step runs in the prebuild script below with proper resolution.
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile --ignore-scripts

# Copy source + content
COPY . .

# Run the prebuild script (which re-runs fumadocs-mdx with proper paths).
# This generates the .source/ directory that next build needs.
ENV NEXT_TELEMETRY_DISABLED=1
ENV NEXT_PUBLIC_NEW_API_URL="https://router.atius.com.br"
RUN bun run prebuild && bun run build

# ---- Stage 2: minimal runtime ----
# We do NOT use Next.js standalone output here because the upstream
# new-api-docs-v1 repo doesn't enable it. Instead we copy the full
# .next build output + node_modules (production) + the standalone
# runtime. Image is bigger (~600MB) but matches what next start needs.
FROM docker.io/library/node:20-bookworm-slim AS runner

WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3003
ENV HOSTNAME=0.0.0.0

# Install production-only deps (no devDependencies, no postinstall scripts).
# Note: --omit=dev + --ignore-scripts to skip fumadocs-mdx postinstall
# (same workaround as Stage 1).
COPY package.json bun.lock ./
RUN apt-get update && apt-get install -y --no-install-recommends curl unzip ca-certificates \
    && curl -fsSL https://bun.sh/install | bash \
    && rm -rf /var/lib/apt/lists/* \
    && export PATH="/root/.bun/bin:${PATH}" \
    && bun install --omit=dev --frozen-lockfile --ignore-scripts

# Copy full build output (build/ is .next/, public/ is user uploads).
COPY --from=builder --chown=node:node /app/.next ./.next
COPY --from=builder --chown=node:node /app/public ./public

USER node
EXPOSE 3003

# Healthcheck: GET / via the standalone server (always 200 for Next.js).
# Use wget (curl may not be present; python:3.12-slim ships it).
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:3003/ || exit 1

CMD ["node", "node_modules/next/dist/bin/next", "start", "-p", "3003", "-H", "0.0.0.0"]
