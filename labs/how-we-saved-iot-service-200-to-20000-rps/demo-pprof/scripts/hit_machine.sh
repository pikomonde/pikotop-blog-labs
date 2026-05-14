#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# hit_machine.sh — Simulate machine (IoT device) traffic to /v1/telemetry
#
# This path triggers: bcrypt.CompareHashAndPassword on EVERY request.
# Bcrypt cost 12 ≈ 300ms per op. Under 10 concurrent connections this
# saturates the CPU completely — the flamegraph will show >90% in bcrypt.
# ─────────────────────────────────────────────────────────────────────────────

BASE_URL="${BASE_URL:-http://localhost:8080}"
PPROF_PORT="${PPROF_PORT:-6060}"

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  Machine Auth Path  →  bcrypt on hot path anti-pattern      ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  ⚠  This will peg your CPU. That is the point."
echo "  Expected behaviour: ~5-20 RPS, CPU near 100%"
echo ""

# ── Single request (sanity check) ────────────────────────────────────────────
echo "▶ Single curl request (sanity check)..."
echo ""
curl -s -X POST "$BASE_URL/v1/telemetry" \
  -H "Content-Type: application/json" \
  -H "X-Client-Type: machine" \
  -H "X-Device-ID: device-iot-001" \
  -H "X-Device-Token: s3cr3t-t0k3n-device-iot-001" \
  -H "X-Request-ID: curl-machine-001" \
  -d '{
    "timestamp": "2026-05-06T10:00:00Z",
    "metrics": {
      "speed_kmh": 87.5,
      "rpm": 2340,
      "fuel_pct": 62.1,
      "battery_v": 12.6,
      "coolant_temp_c": 91.2,
      "gps_lat": -6.2088,
      "gps_lng": 106.8456
    },
    "tags": {
      "vehicle_id": "VH-00421",
      "fleet": "fleet-jakarta-north",
      "firmware": "v3.2.1"
    }
  }' | python3 -m json.tool 2>/dev/null || echo "(install python3 for pretty-print)"

echo ""
echo "────────────────────────────────────────────────────────────────"
echo ""

# ── go-wrk load test ─────────────────────────────────────────────────────────
echo "▶ go-wrk load test — 10 connections, 60 seconds"
echo "  ⚠  bcrypt will make this VERY slow. CPU will spike immediately."
echo ""
echo "  While this runs, in another terminal execute:"
echo "    ./scripts/run_pprof.sh"
echo ""
echo "  Then after the profile is captured:"
echo "    ./scripts/gen_html.sh"
echo ""

if command -v go-wrk &> /dev/null; then
  go-wrk \
    -d 60 \
    -c 10 \
    -M POST \
    -H "Content-Type: application/json" \
    -H "X-Client-Type: machine" \
    -H "X-Device-ID: device-iot-001" \
    -H "X-Device-Token: s3cr3t-t0k3n-device-iot-001" \
    -body '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"speed_kmh":87.5,"rpm":2340},"tags":{"vehicle_id":"VH-00421"}}' \
    "$BASE_URL/v1/telemetry"
else
  echo "  [!] go-wrk not found. Install with:"
  echo "      go install github.com/tsliwowicz/go-wrk@latest"
  echo ""
  echo "  Falling back to curl loop (10 parallel, 100 total requests)..."
  echo "  Note: bcrypt is slow — this will take a while."
  echo ""
  for i in $(seq 1 10); do
    for j in $(seq 1 10); do
      curl -s -X POST "$BASE_URL/v1/telemetry" \
        -H "Content-Type: application/json" \
        -H "X-Client-Type: machine" \
        -H "X-Device-ID: device-iot-00$((j % 3 + 1))" \
        -H "X-Device-Token: s3cr3t-t0k3n-device-iot-00$((j % 3 + 1))" \
        -d '{"timestamp":"2026-05-06T10:00:00Z","metrics":{"speed_kmh":87.5},"tags":{}}' \
        -o /dev/null &
    done
    wait
    echo "  Batch $i/10 done"
  done
fi

echo ""
echo "Done. Check your terminal for RPS — it should be very low vs /health."
