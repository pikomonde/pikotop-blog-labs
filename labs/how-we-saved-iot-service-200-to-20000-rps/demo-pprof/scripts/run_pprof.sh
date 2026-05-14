#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# run_pprof.sh — Capture a CPU profile from the running service
#
# Run this WHILE hit_human.sh or hit_machine.sh is running so there is
# actual load to profile. An idle server produces a boring flamegraph.
#
# Usage:
#   ./scripts/run_pprof.sh                  # default: 30s CPU profile
#   SECONDS=60 ./scripts/run_pprof.sh       # custom duration
#   PROFILE=heap ./scripts/run_pprof.sh     # heap profile instead
# ─────────────────────────────────────────────────────────────────────────────

PPROF_HOST="${PPROF_HOST:-localhost}"
PPROF_PORT="${PPROF_PORT:-6060}"
PROFILE="${PROFILE:-profile}"        # profile | heap | goroutine | trace | allocs
DURATION="${SECONDS:-30}"            # seconds to sample CPU
OUTPUT_DIR="./pprof-output"

mkdir -p "$OUTPUT_DIR"

BASE="http://${PPROF_HOST}:${PPROF_PORT}/debug/pprof"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  pprof — Capturing profile from running service             ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  pprof endpoint : $BASE"
echo "  Profile type   : $PROFILE"
echo "  Duration       : ${DURATION}s (for CPU profile)"
echo "  Output dir     : $OUTPUT_DIR/"
echo ""

# ── Verify the pprof server is reachable ─────────────────────────────────────
if ! curl -sf "$BASE/" -o /dev/null; then
  echo "  ✗ Cannot reach pprof server at $BASE"
  echo "    Make sure the service is running: go run ."
  exit 1
fi
echo "  ✓ pprof server is reachable"
echo ""

# ── Capture the profile ───────────────────────────────────────────────────────
OUTPUT_FILE="$OUTPUT_DIR/${PROFILE}_${TIMESTAMP}.pb.gz"

case "$PROFILE" in
  profile)
    echo "  Sampling CPU for ${DURATION}s — keep the load test running!"
    echo "  URL: $BASE/profile?seconds=$DURATION"
    echo ""
    curl -sf "$BASE/profile?seconds=$DURATION" -o "$OUTPUT_FILE"
    ;;
  heap)
    echo "  Capturing heap snapshot..."
    curl -sf "$BASE/heap" -o "$OUTPUT_FILE"
    ;;
  goroutine)
    echo "  Capturing goroutine dump..."
    curl -sf "$BASE/goroutine?debug=1" -o "$OUTPUT_FILE"
    ;;
  allocs)
    echo "  Capturing allocation profile..."
    curl -sf "$BASE/allocs" -o "$OUTPUT_FILE"
    ;;
  trace)
    echo "  Capturing execution trace for ${DURATION}s..."
    curl -sf "$BASE/trace?seconds=$DURATION" -o "$OUTPUT_FILE"
    # Trace uses a different viewer — not go tool pprof
    echo ""
    echo "  View trace with:"
    echo "    go tool trace $OUTPUT_FILE"
    exit 0
    ;;
  *)
    echo "  Unknown profile type: $PROFILE"
    echo "  Valid types: profile | heap | goroutine | allocs | trace"
    exit 1
    ;;
esac

if [ $? -ne 0 ] || [ ! -s "$OUTPUT_FILE" ]; then
  echo "  ✗ Profile capture failed or file is empty."
  exit 1
fi

echo ""
echo "  ✓ Profile saved: $OUTPUT_FILE"
echo "  Size: $(du -sh "$OUTPUT_FILE" | cut -f1)"
echo ""

# ── Convenience: write the latest file path for gen_html.sh ──────────────────
echo "$OUTPUT_FILE" > "$OUTPUT_DIR/.latest"

echo "────────────────────────────────────────────────────────────────"
echo ""
echo "  Next steps:"
echo ""
echo "  1. Interactive CLI explorer:"
echo "     go tool pprof $OUTPUT_FILE"
echo "     Then type: top10    (top 10 functions by CPU)"
echo "                web      (open SVG flamegraph in browser — requires graphviz)"
echo "                list bcrypt  (source-level annotation for bcrypt)"
echo "                list json    (source-level annotation for JSON)"
echo ""
echo "  2. Generate standalone HTML flamegraph:"
echo "     ./scripts/gen_html.sh"
echo ""
echo "  3. Open interactive web UI (recommended):"
echo "     go tool pprof -http=:8888 $OUTPUT_FILE"
echo "     Then open http://localhost:8888"
echo "       → Flame Graph tab shows the call tree"
echo "       → Top tab shows hottest functions"
echo ""
