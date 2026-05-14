#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# gen_html.sh — Generate HTML flamegraph + SVG from a captured pprof profile
#
# Usage:
#   ./scripts/gen_html.sh                          # uses last captured profile
#   ./scripts/gen_html.sh pprof-output/profile_*.pb.gz  # explicit file
#
# Requirements:
#   - Go toolchain (go tool pprof)
#   - Graphviz (for SVG/PDF): brew install graphviz  |  apt install graphviz
# ─────────────────────────────────────────────────────────────────────────────

OUTPUT_DIR="./pprof-output"
mkdir -p "$OUTPUT_DIR"

# ── Resolve input file ────────────────────────────────────────────────────────
if [ -n "$1" ]; then
  PROFILE_FILE="$1"
elif [ -f "$OUTPUT_DIR/.latest" ]; then
  PROFILE_FILE=$(cat "$OUTPUT_DIR/.latest")
else
  # Try to find the most recent profile file
  PROFILE_FILE=$(ls -t "$OUTPUT_DIR"/*.pb.gz 2>/dev/null | head -1)
fi

if [ -z "$PROFILE_FILE" ] || [ ! -f "$PROFILE_FILE" ]; then
  echo "  ✗ No profile file found."
  echo "    Run ./scripts/run_pprof.sh first, or pass the file as an argument:"
  echo "    ./scripts/gen_html.sh pprof-output/profile_20260506_120000.pb.gz"
  exit 1
fi

BASENAME=$(basename "$PROFILE_FILE" .pb.gz)
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  pprof — Generating HTML flamegraph                         ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  Input  : $PROFILE_FILE"
echo "  Output : $OUTPUT_DIR/"
echo ""

# ── 1. Interactive web UI (opens browser) ────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Option A: Interactive web UI (RECOMMENDED for flamegraph)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  Run this command in a separate terminal:"
echo ""
echo "    go tool pprof -http=:8888 $PROFILE_FILE"
echo ""
echo "  Then open: http://localhost:8888"
echo "    - Flame Graph → visual call stack (what you want for the article)"
echo "    - Top         → hottest functions ranked"
echo "    - Graph       → call graph with edge weights"
echo "    - Source      → annotated source code"
echo ""

# ── 2. Generate SVG (requires graphviz) ──────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Option B: Generate static SVG (requires Graphviz)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

SVG_OUT="$OUTPUT_DIR/${BASENAME}_graph.svg"

if command -v dot &> /dev/null; then
  echo "  Generating SVG call graph..."
  go tool pprof -svg "$PROFILE_FILE" > "$SVG_OUT" 2>/dev/null
  if [ -s "$SVG_OUT" ]; then
    echo "  ✓ SVG saved: $SVG_OUT"
    # Try to open in browser
    if command -v open &> /dev/null; then open "$SVG_OUT"; fi
    if command -v xdg-open &> /dev/null; then xdg-open "$SVG_OUT" 2>/dev/null; fi
  else
    echo "  ✗ SVG generation failed (empty output)"
  fi
else
  echo "  [!] Graphviz not found — skipping SVG generation."
  echo "      Install with:"
  echo "        macOS  : brew install graphviz"
  echo "        Ubuntu : sudo apt install graphviz"
  echo "        Windows: choco install graphviz"
fi

echo ""

# ── 3. Generate plain text top-N report ──────────────────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Option C: Plain text top-20 report"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

TEXT_OUT="$OUTPUT_DIR/${BASENAME}_top20.txt"
go tool pprof -top -nodecount=20 "$PROFILE_FILE" > "$TEXT_OUT" 2>&1
if [ -s "$TEXT_OUT" ]; then
  echo "  ✓ Text report saved: $TEXT_OUT"
  echo ""
  cat "$TEXT_OUT"
else
  echo "  ✗ Text report generation failed"
fi

echo ""

# ── 4. Generate annotated source for known suspects ──────────────────────────
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Bonus: Source annotation for hot functions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

for sym in bcrypt json Unmarshal; do
  SRC_OUT="$OUTPUT_DIR/${BASENAME}_src_${sym}.txt"
  go tool pprof -list "$sym" "$PROFILE_FILE" > "$SRC_OUT" 2>&1
  if [ -s "$SRC_OUT" ] && grep -q "." "$SRC_OUT"; then
    echo "  ✓ Source annotation for '$sym': $SRC_OUT"
  else
    rm -f "$SRC_OUT"
    echo "  - '$sym' not found in profile (not hot enough or not present)"
  fi
done

echo ""
echo "────────────────────────────────────────────────────────────────"
echo ""
echo "  Summary of generated files in $OUTPUT_DIR/:"
ls -lh "$OUTPUT_DIR"/ 2>/dev/null | grep -v "^total" | grep -v ".latest"
echo ""
echo "  TIP: For the cleanest flamegraph for your article screenshot,"
echo "       use the web UI → Flame Graph tab:"
echo "       go tool pprof -http=:8888 $PROFILE_FILE"
echo ""
