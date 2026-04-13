#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

# Defaults
ITERATIONS=1000
FILTER=""
WARMUP=10
OUTFILE=""

usage() {
    cat <<EOF
Usage: ./run-timing.sh [options]

Runs large-template wall-clock timing benchmarks across all engines.
Unlike run.sh (which uses Go's testing.B micro-benchmarks), this measures
real execution time on production-sized templates.

Options:
  -n, --iterations N   Number of render iterations per engine (default: 1000)
  -f, --filter STR     Only run scenarios containing STR (e.g. "Nested", "Complex")
  -w, --warmup N       Number of warmup renders before measuring (default: 10)
  -o, --output FILE    Save output to FILE
  -h, --help           Show this help

Examples:
  ./run-timing.sh                          # Run all scenarios, 1000 iterations
  ./run-timing.sh -n 500                   # 500 iterations
  ./run-timing.sh -f "Large Loop"          # Only the Large Loop scenario
  ./run-timing.sh -n 2000 -w 20            # 2000 iterations with 20 warmup renders
  ./run-timing.sh -n 2000 -o timing.txt    # 2000 iterations, save output
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -n|--iterations) ITERATIONS="$2"; shift 2 ;;
        -f|--filter)     FILTER="$2"; shift 2 ;;
        -w|--warmup)     WARMUP="$2"; shift 2 ;;
        -o|--output)     OUTFILE="$2"; shift 2 ;;
        -h|--help)       usage ;;
        *) echo "Unknown option: $1"; usage ;;
    esac
done

RESULTS_DIR="results"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
mkdir -p "$RESULTS_DIR"

ARGS=(-n "$ITERATIONS" -warmup "$WARMUP")
if [[ -n "$FILTER" ]]; then
    ARGS+=(-filter "$FILTER")
fi

RESULT_FILE="$RESULTS_DIR/timing-${TIMESTAMP}.txt"
go run ./cmd/timing/ "${ARGS[@]}" | tee "$RESULT_FILE"
ln -sf "timing-${TIMESTAMP}.txt" "$RESULTS_DIR/timing-latest.txt"
echo ""
echo "Results saved to $RESULT_FILE"

if [[ -n "$OUTFILE" ]]; then
    cp "$RESULT_FILE" "$OUTFILE"
    echo "Also saved to $OUTFILE"
fi
