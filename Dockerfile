# Stage 1: frontend — Vite writes its output into ../internal/webui/dist.
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci --silent
COPY web/ ./
# Provide the embed target so the relative outDir resolves during the build.
RUN mkdir -p /app/internal/webui/dist
COPY internal/webui/dist/.gitkeep /app/internal/webui/dist/.gitkeep
RUN npm run build

# Stage 2: Go binary (embeds the built frontend + openapi.yaml).
FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOTOOLCHAIN=local
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/internal/webui/dist ./internal/webui/dist
ARG VERSION=docker
RUN CGO_ENABLED=0 go build \
    -trimpath -ldflags="-s -w -X github.com/t0mer/raptor/internal/version.Version=${VERSION}" \
    -o /app/raptor ./cmd/raptor/

# Stage 3: runtime — distroless nonroot, shell-free, with CA certs.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /app/raptor /raptor
EXPOSE 8084
EXPOSE 2525
EXPOSE 5354/udp
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/raptor", "--version"]
ENTRYPOINT ["/raptor"]
