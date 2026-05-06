# MURMUR 24-Hour Soak Test Guide

## Overview

The 24-hour soak test validates MURMUR's long-term stability and resource management per PLAN.md Priority 2 (P2: Extended Soak Testing). It monitors:

1. **Memory growth** — Target: <256 MiB per TECHNICAL_IMPLEMENTATION.md §9
2. **Goroutine leaks** — Target: baseline ±5 goroutines
3. **GC sweep times** — Target: <100ms per sweep
4. **Bbolt database growth** — Target: <50 MiB
5. **Circuit rotation leaks** — Target: No resource leaks during Shroud circuit lifecycle

## Quick Start

### Run the Full 24-Hour Test

```bash
# Run with monitoring and automatic reporting
./scripts/soak-test.sh
```

The script will:
- Run pre-soak health checks
- Capture baseline metrics
- Start system resource monitoring (Linux only)
- Run the 24-hour test
- Generate summary report with all metrics

### Run Manually (Without Script)

```bash
# Build and run with 25h timeout (24h + 1h buffer)
# Note: Requires both 'soak' and 'simulation' build tags
go test -tags='soak simulation' -timeout=25h ./test/simulation -v -run=TestSoak24Hour
```

## Monitored Metrics

The soak test samples metrics every 30 seconds and writes them to a JSON file in `test/simulation/soak-metrics/`. Each sample includes:

```json
{
  "timestamp": "2026-05-06T15:30:00Z",
  "elapsed_seconds": 1800,
  "heap_alloc_mb": 127,
  "heap_inuse_mb": 145,
  "total_alloc_mb": 2048,
  "num_gc": 42,
  "last_gc_pause_micros": 82000,
  "max_gc_pause_micros": 95000,
  "num_goroutine": 58,
  "bbolt_size_bytes": 4194304,
  "active_circuits": 50,
  "waves_received": 12000,
  "waves_published": 12000,
  "memory_warnings": 0,
  "memory_critical": 0,
  "gc_pause_violations": 0,
  "goroutine_leaks": 0
}
```

## Test Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| **Duration** | 24 hours | Total test runtime |
| **Node Count** | 50 | Moderate mesh for sustained operation |
| **Sample Interval** | 30 seconds | Metric collection frequency |
| **Circuit Rotation** | 10 minutes | Shroud circuit lifecycle interval |
| **Waves Per Node** | 10 per interval | Simulated activity load |

## Success Criteria

The test passes if ALL of the following conditions are met:

| Criterion | Target | Assertion |
|-----------|--------|-----------|
| **Memory Critical Events** | <10 in 24h | Rare memory pressure spikes |
| **GC Pause Violations** | <50 in 24h | GC sweeps stay under 100ms |
| **Goroutine Leaks** | Baseline ±10 | No goroutine accumulation |
| **Bbolt DB Size** | <50 MiB total | Database stays within spec |
| **Wave Traffic** | >0 received/published | Network activity sustained |
| **Final Memory State** | Not critical | No memory exhaustion |

## Output Files

After the test completes, the following files are created in `test/simulation/soak-metrics/`:

| File | Description |
|------|-------------|
| `soak-24h-<timestamp>.json` | Full metrics time series (1 sample per 30s) |
| `soak-run-<timestamp>.log` | Complete test output log |
| `summary-<timestamp>.txt` | Human-readable summary report |
| `baseline-<timestamp>.json` | Pre-test code complexity metrics |
| `post-<timestamp>.json` | Post-test code complexity metrics |
| `complexity-diff-<timestamp>.txt` | Complexity regression check |
| `system-monitor-<timestamp>.log` | System-level resource usage (Linux only) |

## Analyzing Results

### View Metrics Time Series

```bash
# Extract heap memory over time (CSV format)
cat test/simulation/soak-metrics/soak-24h-*.json | \
  jq -r '[.elapsed_seconds, .heap_alloc_mb] | @csv' > /tmp/memory.csv

# View in spreadsheet or plot with gnuplot:
gnuplot -p -e "set xlabel 'Time (seconds)'; set ylabel 'Heap Memory (MB)'; \
  plot '/tmp/memory.csv' using 1:2 with lines title 'Heap Allocation'"
```

### Check for Memory Growth Trend

```bash
# Linear regression on heap memory (requires Python + numpy)
cat test/simulation/soak-metrics/soak-24h-*.json | \
  jq -r '[.elapsed_seconds, .heap_alloc_mb] | @csv' | \
  python3 -c "
import sys, numpy as np
data = np.loadtxt(sys.stdin, delimiter=',')
x, y = data[:, 0], data[:, 1]
slope, intercept = np.polyfit(x, y, 1)
print(f'Memory growth rate: {slope:.6f} MB/second = {slope*3600:.2f} MB/hour')
print(f'Projected 24h growth: {slope*86400:.2f} MB')
"
```

### Identify GC Pause Spikes

```bash
# Extract all GC pause violations (>100ms)
cat test/simulation/soak-metrics/soak-24h-*.json | \
  jq -r 'select(.last_gc_pause_micros > 100000) | 
    [.timestamp, .last_gc_pause_micros/1000] | 
    @csv' | \
  column -t -s,
```

### Check Goroutine Stability

```bash
# Plot goroutine count over time
cat test/simulation/soak-metrics/soak-24h-*.json | \
  jq -r '[.elapsed_seconds, .num_goroutine] | @csv' > /tmp/goroutines.csv

# Calculate stddev to check for leaks
cat /tmp/goroutines.csv | \
  python3 -c "
import sys, numpy as np
data = np.loadtxt(sys.stdin, delimiter=',')
x, y = data[:, 0], data[:, 1]
print(f'Goroutine count: mean={y.mean():.1f}, stddev={y.std():.2f}, range=[{y.min():.0f}, {y.max():.0f}]')
print(f'Leak detection: {"NO LEAK" if y.std() < 5 else "POSSIBLE LEAK"}')
"
```

## Troubleshooting

### Test Fails: Memory Critical

**Symptom**: `memory_critical > 10` or final assertion fails  
**Cause**: Memory usage exceeded 256 MiB target

**Actions**:
1. Analyze memory profile: `go tool pprof -http=:8080 heap.prof`
2. Check for allocation hotspots in Wave propagation
3. Verify LRU cache eviction is working (see `pkg/content/propagation/relay.go`)
4. Review Bbolt cache settings

### Test Fails: GC Pause Violations

**Symptom**: `gc_pause_violations > 50` in 24h  
**Cause**: GC sweeps exceeding 100ms target

**Actions**:
1. Reduce heap allocation rate (see `PERFORMANCE_ANALYSIS_1000NODE.md` §3.1)
2. Tune `GOGC` environment variable (lower = more frequent, shorter GC)
3. Consider batch allocation strategies
4. Profile with: `go test -gcflags='-m -m' ./...` to find escape analysis issues

### Test Fails: Goroutine Leak

**Symptom**: Final goroutine count > baseline + 10  
**Cause**: Goroutines not properly cleaned up

**Actions**:
1. Check for missing context cancellation
2. Verify all goroutines have proper lifecycle management
3. Use goroutine profiler: `curl http://localhost:6060/debug/pprof/goroutine?debug=2`
4. Review circuit rotation logic in `pkg/anonymous/shroud/`

### Test Fails: Database Growth

**Symptom**: Bbolt database size > 50 MiB  
**Cause**: Insufficient TTL enforcement or eviction

**Actions**:
1. Check Wave expiry logic in `pkg/content/storage/`
2. Verify GC goroutine is running (see `TECHNICAL_IMPLEMENTATION.md` §8)
3. Review TTL settings for Waves (default 7 days, max 30)
4. Check if eviction is triggered correctly

## CI Integration

To run in CI with shorter duration (for smoke testing):

```yaml
# .github/workflows/soak-test.yml
name: Soak Test (Shortened)
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday
jobs:
  soak:
    runs-on: ubuntu-latest
    timeout-minutes: 360  # 6 hours
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Run 6-hour soak test
        run: |
          # Modify duration in test file temporarily
          sed -i 's/24 \* time.Hour/6 * time.Hour/' test/simulation/soak_test.go
          go test -tags=soak -timeout=7h ./test/simulation -v -run=TestSoak24Hour
      - name: Upload metrics
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: soak-metrics
          path: test/simulation/soak-metrics/
```

## Performance Baselines (Expected Values)

Based on 1000-node simulation results (see `PERFORMANCE_VALIDATION_2026-05-06.md`):

| Metric | Expected Range | Notes |
|--------|----------------|-------|
| Heap Memory | 100-200 MB | Should stabilize after warmup |
| GC Pause (p50) | 20-50 ms | Most pauses well under 100ms |
| GC Pause (p99) | 80-100 ms | Rare spikes acceptable |
| Goroutines | 50-65 | ~8 persistent + per-node overhead |
| Bbolt DB | 5-30 MB | Depends on Wave retention |
| Wave Propagation | <50ms p50 | Should stay under 500ms target |

## References

- **PLAN.md** — Task definition (P2: Extended Soak Testing)
- **TECHNICAL_IMPLEMENTATION.md §9** — Performance targets
- **PERFORMANCE_ANALYSIS_1000NODE.md** — 1000-node baseline results
- **PERFORMANCE_VALIDATION_2026-05-06.md** — Target validation report
- **pkg/resources/monitor.go** — Memory monitoring implementation
- **pkg/content/propagation/relay.go** — LRU cache eviction logic
