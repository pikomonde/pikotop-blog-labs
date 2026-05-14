#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# hit_human.sh — Simulate human (dashboard) traffic to /v1/telemetry
#
# This path triggers: Redis fetch → JSON unmarshal of ~300KB BigUser blob →
# permission check → response.
# pprof should show json.Unmarshal taking a noticeable but non-fatal CPU slice.
# ─────────────────────────────────────────────────────────────────────────────

BASE_URL="${BASE_URL:-http://localhost:8080}"
PPROF_PORT="${PPROF_PORT:-6060}"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  Human Auth Path  →  BigUser JSON unmarshal anti-pattern    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# ── Single request (sanity check) ────────────────────────────────────────────
echo "▶ Single curl request (sanity check)..."
echo ""
curl -s -X POST "$BASE_URL/v1/telemetry" \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: human" \
  -H "X-User-ID: usr-demo-001" \
  -H "X-Required-Permission: telemetry:read" \
  -H "X-Request-ID: curl-human-001" \
  -d '{
    "timestamp": "2026-05-06T10:00:00Z",
    "metrics": {"dashboard_load_ms": 142.5, "widgets": 8},
    "tags": {"page": "fleet-overview", "session": "ses-abc123"}
  }' | python3 -m json.tool 2>/dev/null || echo "(install python3 for pretty-print)"

echo ""
echo "────────────────────────────────────────────────────────────────"
echo ""

# ── go-wrk load test ─────────────────────────────────────────────────────────
# go-wrk doesn't support custom JSON bodies natively, but we can point it at
# the endpoint with headers. For body we use a simple approach.
#
# Install go-wrk: go install github.com/tsliwowicz/go-wrk@latest
# Or: brew install go-wrk (if available)

echo "▶ go-wrk load test — 10 connections, 60 seconds"
echo "  This will generate load for profiling."
echo "  While this runs, in another terminal execute:"
echo "    ./scripts/run_pprof.sh"
echo ""

if command -v go-wrk &> /dev/null; then
  go-wrk \
    -d 60 \
    -c 10 \
    -M POST \
    -H "Content-Type: application/json" \
    -H "X-Client-Type: human" \
    -H "X-User-ID: usr-demo-001" \
    -H "X-Required-Permission: telemetry:read" \
    -body '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"ping":1},"tags":{}}' \
    "$BASE_URL/v1/telemetry"
else
  echo "  [!] go-wrk not found. Install with:"
  echo "      go install github.com/tsliwowicz/go-wrk@latest"
  echo ""
  echo "  Falling back to curl loop (10 parallel, 200 total requests)..."
  echo ""
  for i in $(seq 1 20); do
    for j in $(seq 1 10); do
      curl -s -X POST "$BASE_URL/v1/telemetry" \
        -H "Content-Type: application/json" \
        -H "X-Client-Type: human" \
        -H "X-User-ID: usr-demo-001" \
        -H "X-Required-Permission: telemetry:read" \
        -d '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"ping":1},"tags":{}}' \
        -o /dev/null &
    done
    wait
    echo "  Batch $i/20 done"
  done
fi

echo ""
echo "Done."
