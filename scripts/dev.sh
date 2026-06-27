#!/usr/bin/env bash
# Local development: run the Go backend and the Vite dev server together with
# hot reload. The Vite server proxies /api and /health to the backend.
set -euo pipefail
cd "$(dirname "$0")/.."

cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

mkdir -p data

echo "starting backend on :8084"
go run ./cmd/raptor --data ./data --base-url http://localhost:5173 --log-level debug &

echo "starting frontend dev server on :5173"
(cd web && npm install --silent && npm run dev) &

wait
