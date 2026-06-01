FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder

WORKDIR /build
COPY web/default/package.json .
COPY web/default/bun.lock .
RUN bun install
# Clear rsbuild/bun cache before build
RUN rm -rf ~/.cache ~/.bun/install/cache/node_modules/.cache .rsbuild .next .nuxt
COPY ./web/default .
COPY ./VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder-classic

WORKDIR /build
COPY web/classic/package.json .
COPY web/classic/bun.lock .
RUN bun install
COPY ./web/classic .
COPY ./VERSION .
RUN VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=builder /build/dist ./web/default/dist
COPY --from=builder-classic /build/dist ./web/classic/dist

# ATIUS BRANDING: replace ALL embedded assets in dist AFTER builder copy
# Remove old embedded PNG/logo files from the dist, keep JS/CSS, then copy Atius assets
RUN find ./web/default/dist -type f \( -name "*.png" -o -name "*.ico" -o -name "*.svg" -o -name "*.jpg" -o -name "*.jpeg" -o -name "*.gif" -o -name "*.webp" \) -delete 2>/dev/null || true
COPY web/default/public/logo.png ./web/default/dist/logo.png
COPY web/default/public/logo.svg ./web/default/dist/logo.svg
COPY web/default/public/favicon.ico ./web/default/dist/favicon.ico
RUN echo "=== DIST FILES BEFORE GO BUILD ===" && find ./web/default/dist -type f | wc -l && find ./web/default/dist -name "*.png" -exec ls -lh {} \;

RUN go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/new-api /
<<<<<<< HEAD

=======
COPY LICENSE NOTICE THIRD-PARTY-LICENSES.md /licenses/
>>>>>>> upstream/main
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/new-api"]
