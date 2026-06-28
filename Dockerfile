# Stage 1: frontend — built once on the native build platform (the output is
# static JS/CSS, so it doesn't need per-target emulation). Vite writes into
# ../internal/webui/dist.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
# Provide the embed target so the relative outDir resolves during the build.
RUN mkdir -p /app/internal/webui/dist
COPY internal/webui/dist/.gitkeep /app/internal/webui/dist/.gitkeep
RUN npm run build

# Stage 2: Go binary (embeds the built frontend + openapi.yaml). Runs on the
# native build platform and cross-compiles to the target, avoiding QEMU.
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/webui/dist ./internal/webui/dist
ARG VERSION=docker
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} go build \
    -trimpath -ldflags="-s -w -X github.com/t0mer/raptor/internal/version.Version=${VERSION}" \
    -o /app/raptor ./cmd/raptor/

# Pre-create the data dir so the distroless nonroot user can write the SQLite DB
# (the image has no shell to mkdir/chown at runtime).
RUN mkdir -p /data

# Stage 3: runtime — distroless nonroot, shell-free, with CA certs.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/raptor /raptor
COPY --from=builder --chown=nonroot:nonroot /data /data
EXPOSE 8084
EXPOSE 2525
EXPOSE 5354/udp
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/raptor", "--version"]
ENTRYPOINT ["/raptor"]
