#!/usr/bin/env bash
# Run 24-hour soak test with monitoring and reporting.
# Per PLAN.md P2: Extended Soak Testing
# Monitors: memory growth, goroutine leaks, GC sweep times, Bbolt DB growth, circuit rotation leaks

set -euo pipefail

# Configuration
DURATION="24h"
TEST_TIMEOUT="25h"
METRICS_DIR="./test/simulation/soak-metrics"
TIMESTAMP=$(date +%s)
LOG_FILE="${METRICS_DIR}/soak-run-${TIMESTAMP}.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "MURMUR 24-Hour Soak Test"
echo "=========================================="
echo "Duration: ${DURATION}"
echo "Timeout: ${TEST_TIMEOUT}"
echo "Metrics: ${METRICS_DIR}"
echo "Log: ${LOG_FILE}"
echo ""

# Create metrics directory
mkdir -p "${METRICS_DIR}"

# Check if go-stats-generator is available (optional, for post-analysis)
if command -v go-stats-generator &> /dev/null; then
    echo "✓ go-stats-generator found (will generate complexity report)"
    HAS_STATS_GEN=1
else
    echo "⚠ go-stats-generator not found (skipping complexity report)"
    HAS_STATS_GEN=0
fi

# Run baseline tests to ensure system is healthy
echo ""
echo "Running pre-soak health check..."
if go test -race -count=1 ./pkg/... -timeout=5m > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Pre-soak tests passed${NC}"
else
    echo -e "${RED}✗ Pre-soak tests failed - aborting${NC}"
    exit 1
fi

# Capture baseline metrics
echo ""
echo "Capturing baseline metrics..."
BASELINE_FILE="${METRICS_DIR}/baseline-${TIMESTAMP}.json"
if [ $HAS_STATS_GEN -eq 1 ]; then
    go-stats-generator analyze . --skip-tests --format json --sections functions,duplication > "${BASELINE_FILE}"
    echo "✓ Baseline metrics saved to: ${BASELINE_FILE}"
fi

# Start system resource monitoring (Linux only)
MONITOR_PID=""
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    MONITOR_FILE="${METRICS_DIR}/system-monitor-${TIMESTAMP}.log"
    echo ""
    echo "Starting system resource monitor..."
    (
        while true; do
            echo "$(date +%s),$(free -m | grep Mem | awk '{print $3}'),$(ps aux | wc -l),$(df -h . | tail -1 | awk '{print $5}')" >> "${MONITOR_FILE}"
            sleep 60
        done
    ) &
    MONITOR_PID=$!
    echo "✓ System monitor started (PID: ${MONITOR_PID})"
    echo "  Logging to: ${MONITOR_FILE}"
fi

# Trap to cleanup on exit
cleanup() {
    echo ""
    echo "Cleaning up..."
    if [ -n "${MONITOR_PID}" ] && kill -0 "${MONITOR_PID}" 2>/dev/null; then
        kill "${MONITOR_PID}" 2>/dev/null || true
        echo "✓ Stopped system monitor"
    fi
}
trap cleanup EXIT INT TERM

# Run soak test
echo ""
echo "=========================================="
echo "Starting 24-hour soak test..."
echo "=========================================="
echo "Start time: $(date)"
echo ""
echo "The test will run for ${DURATION}. You can monitor progress in:"
echo "  ${LOG_FILE}"
echo ""
echo "To monitor in real-time:"
echo "  tail -f ${LOG_FILE}"
echo ""

START_TIMESTAMP=$(date +%s)

# Run test with timeout and capture output (note: requires both soak and simulation tags)
if go test -tags='soak simulation' -timeout="${TEST_TIMEOUT}" ./test/simulation -v -run=TestSoak24Hour 2>&1 | tee "${LOG_FILE}"; then
    TEST_RESULT="PASSED"
    echo -e "${GREEN}"
else
    TEST_RESULT="FAILED"
    echo -e "${RED}"
fi

END_TIMESTAMP=$(date +%s)
ACTUAL_DURATION=$((END_TIMESTAMP - START_TIMESTAMP))

echo "=========================================="
echo "Soak Test Complete"
echo "=========================================="
echo "End time: $(date)"
echo "Actual duration: $((ACTUAL_DURATION / 3600))h $((ACTUAL_DURATION % 3600 / 60))m $((ACTUAL_DURATION % 60))s"
echo "Result: ${TEST_RESULT}"
echo -e "${NC}"

# Capture post-test metrics
echo ""
echo "Capturing post-test metrics..."
POST_FILE="${METRICS_DIR}/post-${TIMESTAMP}.json"
if [ $HAS_STATS_GEN -eq 1 ]; then
    go-stats-generator analyze . --skip-tests --format json --sections functions,duplication > "${POST_FILE}"
    echo "✓ Post-test metrics saved to: ${POST_FILE}"
    
    # Generate diff report
    echo ""
    echo "Generating complexity diff report..."
    DIFF_FILE="${METRICS_DIR}/complexity-diff-${TIMESTAMP}.txt"
    go-stats-generator diff "${BASELINE_FILE}" "${POST_FILE}" > "${DIFF_FILE}" 2>&1 || true
    echo "✓ Diff report saved to: ${DIFF_FILE}"
fi

# Generate summary report
SUMMARY_FILE="${METRICS_DIR}/summary-${TIMESTAMP}.txt"
echo ""
echo "Generating summary report..."
{
    echo "=========================================="
    echo "MURMUR 24-Hour Soak Test Summary"
    echo "=========================================="
    echo ""
    echo "Test Configuration:"
    echo "  Start Time: $(date -d @${START_TIMESTAMP})"
    echo "  End Time: $(date -d @${END_TIMESTAMP})"
    echo "  Duration: $((ACTUAL_DURATION / 3600))h $((ACTUAL_DURATION % 3600 / 60))m $((ACTUAL_DURATION % 60))s"
    echo "  Result: ${TEST_RESULT}"
    echo ""
    echo "Metrics Files:"
    echo "  Soak Metrics: ${METRICS_DIR}/soak-24h-*.json"
    echo "  Test Log: ${LOG_FILE}"
    if [ $HAS_STATS_GEN -eq 1 ]; then
        echo "  Baseline: ${BASELINE_FILE}"
        echo "  Post-Test: ${POST_FILE}"
        echo "  Complexity Diff: ${DIFF_FILE}"
    fi
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "  System Monitor: ${MONITOR_FILE}"
    fi
    echo ""
    echo "Key Findings:"
    echo ""
    # Extract key metrics from test output
    grep -A 20 "SOAK TEST RESULTS" "${LOG_FILE}" || echo "  (Results not found in log)"
    echo ""
    echo "To analyze metrics:"
    echo "  # View metrics JSON:"
    echo "  cat ${METRICS_DIR}/soak-24h-*.json | jq -r '[.timestamp, .heap_alloc_mb, .num_goroutine] | @csv'"
    echo ""
    echo "  # Plot memory over time (requires gnuplot):"
    echo "  cat ${METRICS_DIR}/soak-24h-*.json | jq -r '[.elapsed_seconds, .heap_alloc_mb] | @csv' > /tmp/mem.dat"
    echo "  gnuplot -e 'set terminal png; set output \"memory.png\"; plot \"/tmp/mem.dat\" with lines'"
    echo ""
} > "${SUMMARY_FILE}"

cat "${SUMMARY_FILE}"
echo ""
echo "✓ Summary report saved to: ${SUMMARY_FILE}"

# Return test exit code
if [ "${TEST_RESULT}" = "PASSED" ]; then
    echo ""
    echo -e "${GREEN}✓ Soak test PASSED - system is stable${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}✗ Soak test FAILED - review logs for issues${NC}"
    exit 1
fi
