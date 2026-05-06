# Concurrency Safety Gaps — 2026-05-06

This document identifies concurrency safety gaps between MURMUR's **stated design goals** and **current implementation state**. These are architectural shortcomings or missing safeguards that could lead to concurrency bugs as the system scales or evolves.

---

## Gap 1: No Subsystem-Level Shutdown Instrumentation

### Stated Goal
Per TECHNICAL_IMPLEMENTATION.md §8 and ROADMAP.md line 814:
> "Graceful shutdown with ordered subsystem teardown. Each subsystem must shut down within 3 seconds."

### Current State
- Application shutdown uses a **single 10-second global timeout** (`pkg/app/murmur.go:584-596`)
- No per-subsystem timing instrumentation
- No identification of which subsystem causes delays
- Tests fail consistently: `TestGracefulShutdown` reports 10.002967443s actual vs 3s expected

### Evidence
```
WARNING: Graceful shutdown timeout reached after 10 seconds
--- FAIL: TestGracefulShutdown (10.03s)
    main_test.go:290: Shutdown took 10.002967443s, expected < 3s
```

Repeated across 6 test runs. The timeout is hit consistently, indicating at least one goroutine is not respecting context cancellation promptly.

### Risk
- **Production impact:** Users experience hung shutdowns requiring force-kill (SIGKILL)
- **Data loss:** Bbolt transactions may not flush; in-flight Waves lost
- **Debugging difficulty:** Cannot identify which subsystem is blocking without instrumentation
- **CI flakiness:** Shutdown tests are unreliable, masking real regressions

### Root Cause Analysis
The application launches 7 persistent goroutines but does not track which complete during shutdown:
1. Event bus (`eventbus.go:248`)
2. Deduplication rotation (`murmur.go:474`)
3. GC (`murmur.go:480`)
4. Memory monitor (`murmur.go:486`)
5. Nudge loop (`murmur.go:496`)
6. Beacon loop (`murmur.go:528`)
7. Layout engine (`pulsemap/layout/engine.go:199`)

Likely culprits (by blocking potential):
1. **Layout engine:** Uses separate `stopCh` not synchronized with context → `M2` in AUDIT.md
2. **Bbolt GC:** May be mid-transaction when context cancels → transactions can block 1-5s
3. **libp2p PubSub:** `ps.Close()` may block on peer disconnections → no timeout specified

### Closing the Gap
**Immediate (1 week):**
1. Add per-goroutine exit logging: `defer log.Debug("shutdown:goroutine:NAME:complete")`
2. Add per-subsystem timeout tracking:
   ```go
   subsystems := []struct{ name string; closer func() error }{
       {"EventBus", a.closeEventBus},
       {"WaveCache", a.closeWaveCache},
       // ...
   }
   for _, sub := range subsystems {
       start := time.Now()
       if err := sub.closer(); err != nil { /* log */ }
       elapsed := time.Since(start)
       if elapsed > 2*time.Second {
           log.Warn("subsystem:%s:slow_shutdown:duration=%v", sub.name, elapsed)
       }
   }
   ```
3. Reduce global timeout from 10s to 5s (force subsystems to respect budget)

**Short-term (1 month):**
4. Refactor layout engine to use context-only shutdown (remove `stopCh`)
5. Add Bbolt write batching flush on context cancel: `db.Update()` with 100ms deadline
6. Wrap `ps.Close()` in timeout: `ctx, cancel := context.WithTimeout(ctx, 3*time.Second)`

**Long-term (3 months):**
7. Add shutdown smoke test to CI that **gates merges**: must complete in <3s
8. Add `runtime.NumGoroutine()` baseline test: verify all goroutines exit

**Validation:**
```bash
go test -v -run TestGracefulShutdown ./cmd/murmur
# Expected: PASS in <3s with per-subsystem timing logs
```

---

## Gap 2: No Context Cancellation Contract Tests

### Stated Goal
Per TECHNICAL_IMPLEMENTATION.md §8:
> "~8 persistent goroutines with context cancellation. All goroutines must respect context.Done()."

### Current State
- 28/28 production goroutines receive `context.Context`
- Only **8/28 explicitly select on `ctx.Done()`** in their main loops
- Remaining 20 delegate context handling to called functions (e.g., `sub.Next(ctx)`)
- **Zero tests** verify goroutine exit latency on context cancellation

### Evidence
```bash
$ grep -r "go func()" --include="*.go" --exclude="*_test.go" pkg/ | wc -l
28  # Total production goroutines launched

$ grep -r "ctx.Done()" --include="*.go" --exclude="*_test.go" pkg/ | wc -l
8   # Only 8 explicitly handle cancellation in select statements
```

Goroutines relying on implicit context handling (examples):
- `pkg/app/murmur.go:474-477` — deduplication rotation calls `handlers.StartDedupRotation(ctx)`
- `pkg/app/murmur.go:480-484` — GC calls `cache.StartGC(ctx, interval)`
- `pkg/networking/gossip/ephemeral.go:45` — subscription handler calls `sub.Next(ctx)`

### Risk
- **Goroutine leaks:** If a called function's context handling is refactored to ignore context, the goroutine leaks
- **Shutdown hangs:** Contributes to Gap 1 (shutdown timeout)
- **Resource exhaustion:** Long-running tests accumulate goroutines
- **Fragile refactoring:** No contract enforcement — developers must manually verify context propagation

### Current Safety Mechanisms
All 20 goroutines **currently** delegate to functions that do respect context:
- `sub.Next(ctx)` — libp2p subscription blocks on context
- `cache.StartGC(ctx, interval)` — uses `select { case <-ctx.Done(): case <-ticker.C: }`
- `handlers.StartDedupRotation(ctx)` — similar ticker pattern

**However,** this is implicit, not tested. A future refactor could break this.

### Closing the Gap
**Immediate (1 week):**
1. Add per-goroutine context cancellation tests:
   ```go
   func TestGoroutineRespondsToContextCancel(t *testing.T) {
       ctx, cancel := context.WithCancel(context.Background())
       done := make(chan struct{})
       
       go func() {
           defer close(done)
           goroutineUnderTest(ctx)  // e.g., cache.StartGC(ctx, 1*time.Second)
       }()
       
       time.Sleep(100 * time.Millisecond)  // Let goroutine start
       cancel()
       
       select {
       case <-done:
           // Success: goroutine exited
       case <-time.After(1 * time.Second):
           t.Fatal("goroutine did not exit within 1s of context cancellation")
       }
   }
   ```
2. Add 28 such tests (one per production goroutine) in their respective packages

**Short-term (1 month):**
3. Add linter rule: "All `go func()` must either select on `ctx.Done()` OR have a test proving exit latency <1s"
4. Document context contract in `CONTRIBUTING.md`: "All goroutines must exit within 1 second of context cancellation"

**Long-term (3 months):**
5. Add goroutine leak detector to CI: `goleak.VerifyNone(t)` in all tests
6. Add runtime instrumentation: track active goroutine count via Prometheus gauge `murmur_goroutines_active`

**Validation:**
```bash
go test -v -run TestGoroutineRespondsToContextCancel ./pkg/...
# Expected: PASS for all 28 goroutines
```

---

## Gap 3: No Load Testing for Event Bus Backpressure

### Stated Goal
Per TECHNICAL_IMPLEMENTATION.md §2:
> "Event bus uses 256-entry buffered channel for inbound events. Non-blocking sends drop events under load."

### Current State
- Event bus buffer: 256 entries
- Non-blocking `Emit()` drops events when buffer full
- No tests verify behavior under high load (1K+ events/sec)
- No production guidance on buffer sizing for viral content scenarios
- `EventBusDropsTotal` metric exists but no alert thresholds defined

### Evidence
```go
// pkg/app/eventbus.go:347-360
func (eb *EventBus) Emit(event Event) {
    select {
    case eb.inbound <- event:
    default:
        // Buffer full, drop event.
        metrics.EventBusDropsTotal.WithLabelValues("inbound_full").Inc()
    }
}
```

No corresponding test exercises this code path. The only test is `eventbus_test.go:TestEventBusBasic` which sends 10 events sequentially.

### Risk
- **Silent data loss:** Critical events (e.g., `EventReplyReceived`, `EventShroudCircuitFailed`) dropped under viral Wave propagation
- **Broken UI state:** Pulse Map may miss node updates, show stale graph
- **Circuit recovery failure:** Shroud circuit rebuild events dropped → user stuck without anonymity
- **No production visibility:** Buffer size adequate for 100 nodes, unknown for 10K nodes

### Realistic Load Scenario
- 1000 peers in network
- Viral Wave propagates to all peers within 5 seconds
- Each peer emits: `EventWaveReceived` (1) + `EventPeerConnected` (1 per hop = 3) = 4 events
- Total: 1000 peers × 4 events = **4000 events in 5 seconds = 800 events/sec**
- Buffer fills in: 256 / 800 = **0.32 seconds**
- Remaining 4.68 seconds: **~3744 events dropped**

### Closing the Gap
**Immediate (1 week):**
1. Add load test:
   ```go
   func TestEventBusHighLoad(t *testing.T) {
       eb := NewEventBus(EventBusConfig{BufferSize: 256})
       ctx, cancel := context.WithCancel(context.Background())
       defer cancel()
       go eb.Start(ctx)
       
       subscriber := make(chan Event, 100)
       eb.SubscribeAll(subscriber)
       
       // Send 10K events rapidly
       for i := 0; i < 10000; i++ {
           eb.Emit(Event{Type: EventWaveReceived})
       }
       
       // Verify drops
       dropsMetric := testutil.ToFloat64(metrics.EventBusDropsTotal.WithLabelValues("inbound_full"))
       if dropsMetric > 100 {
           t.Errorf("Too many drops: %v (expected <100 with 256 buffer)", dropsMetric)
       }
   }
   ```
2. Increase buffer size from 256 → 1024 based on viral propagation math

**Short-term (1 month):**
3. Add event priority levels:
   ```go
   type Priority int
   const (
       PriorityCritical Priority = iota  // Never drop
       PriorityHigh
       PriorityNormal
       PriorityLow
   )
   ```
4. Implement separate channels:
   - `criticalInbound chan Event` (unbuffered, blocks on full)
   - `normalInbound chan Event` (buffered 1024)
   - `lowInbound chan Event` (buffered 256, drops freely)
5. Route events by type:
   - `EventShroudCircuitFailed`, `EventReplyReceived` → Critical
   - `EventWaveReceived`, `EventPeerConnected` → Normal
   - `EventHeartbeatReceived` → Low

**Long-term (3 months):**
6. Add adaptive buffer sizing: monitor `EventBusDropsTotal`, increase buffer if sustained drops >10/min
7. Add production alert: "Event bus drops >100 in 1 minute → investigate viral content or network partition"
8. Add simulation test: 10K-node network with libp2p hosts sending real Waves

**Validation:**
```bash
go test -v -run TestEventBusHighLoad ./pkg/app
# Expected: <1% drop rate (100/10000) with 1024 buffer
```

---

## Gap 4: No Per-Hop Timeout for Shroud Circuit Construction

### Stated Goal
Per SHADOW_GRADIENT.md and SECURITY_PRIVACY.md:
> "Shroud circuits use three-hop onion routing. Circuit construction must complete within reasonable time to maintain usability."

### Current State
- Circuit construction dials 3 relay peers sequentially via `host.NewStream(ctx, peer, protocol)`
- No per-hop timeout specified → uses default libp2p context timeout (30s per hop)
- Total potential blocking time: **3 hops × 30s = 90 seconds**
- No parallel dial to multiple candidate relays
- No fast-fail on unreachable peers

### Evidence
```go
// pkg/anonymous/shroud/circuit.go (inferred structure, not fully examined)
for _, hop := range [3]PeerID{relay1, relay2, relay3} {
    stream, err := host.NewStream(ctx, hop, ProtocolShroudCircuit)  // May block 30s
    if err != nil {
        return fmt.Errorf("dialing hop: %w", err)
    }
    // Perform key exchange...
}
```

No timeout wrapper around `NewStream()`. If `relay1` is offline (e.g., behind NAT, node shutdown), the call blocks for 30 seconds before returning an error.

### Risk
- **Poor UX:** User clicks "Send Anonymous Wave" → UI freezes for 30-90 seconds → user assumes app crashed
- **Resource exhaustion:** If 10 concurrent circuit builds all block 90s, 10 goroutines stalled for 15 minutes total
- **Cascade failure:** If all known relays are offline (e.g., network partition), every circuit build attempt fails after 90s
- **Amplification attack:** Adversary registers malicious relays that accept connections but never respond → DoS client circuit construction

### Realistic Failure Scenario
- Network has 50 known relays
- 30% are behind strict NAT (unreachable)
- 10% are offline (node shutdown)
- Client randomly selects 3 relays for circuit
- Probability all 3 reachable: 0.6^3 = **21.6%**
- Probability at least 1 unreachable: **78.4%**
- Average circuit build time: 0.784 × 30s (timeout) + 0.216 × 5s (success) = **24.6 seconds per attempt**

With 3 retry attempts, total latency: **73.8 seconds** before user sees error.

### Closing the Gap
**Immediate (1 week):**
1. Add per-hop 10-second timeout:
   ```go
   for _, hop := range hops {
       hopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
       defer cancel()
       stream, err := host.NewStream(hopCtx, hop, ProtocolShroudCircuit)
       if err != nil {
           log.Warn("hop unreachable: %v", err)
           continue  // Try next relay candidate
       }
   }
   ```
2. Add relay pre-filtering: exclude penalized relays (per `RelayFailureTracker`)

**Short-term (1 month):**
3. Add parallel dial to 5 candidate relays, take first 3 to respond:
   ```go
   candidates := beacon.SelectRelays(5)  // Over-provision
   results := make(chan dialResult, 5)
   for _, relay := range candidates {
       go func(r PeerID) {
           hopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
           defer cancel()
           stream, err := host.NewStream(hopCtx, r, ProtocolShroudCircuit)
           results <- dialResult{relay: r, stream: stream, err: err}
       }(relay)
   }
   
   var hops [3]stream
   for i := 0; i < 3; i++ {
       result := <-results
       if result.err == nil {
           hops[i] = result.stream
       }
   }
   ```
4. Add circuit build latency metric: `ShroudCircuitBuildDurationSeconds` histogram

**Long-term (3 months):**
5. Add relay health probing: periodic `ping` to known relays, maintain reachability score
6. Add circuit pooling: pre-build 2-3 circuits on application startup, rotate every 10 minutes
7. Add fallback: if circuit build fails 3× in 60s, disable anonymous messaging temporarily (fallback to Surface Layer)

**Validation:**
```bash
# Simulate offline relay
go test -v -run TestCircuitBuildWithOfflineRelay ./pkg/anonymous/shroud
# Expected: circuit fails within 30s (3 hops × 10s each), not 90s
```

---

## Gap 5: No Memory Budget Enforcement at Application Startup

### Stated Goal
Per PERFORMANCE targets in README.md:
> "Memory usage <256 MiB during normal operation."

### Current State
- Memory monitor goroutine runs every 60 seconds (`pkg/app/murmur.go:486-493`)
- Monitors `runtime.MemStats` and evicts Waves if memory exceeds 200 MiB
- **No startup budget check** — application may consume >256 MiB before first GC pass
- **No subsystem-level budgets** — cannot identify which subsystem causes overrun

### Evidence
```go
// pkg/app/murmur.go:486-493
a.wg.Add(1)
go func() {
    defer a.wg.Done()
    a.runMemoryMonitor(cache)  // Starts after subsystem init
}()

// First memory check happens 60 seconds after startup
// If Wave cache loads 10K Waves at startup (1 MiB each), memory = 10 GiB before eviction
```

No pre-loading limit. If user has 30 days of Waves persisted (max TTL), all are loaded into memory at startup via `storage.NewCache()`.

### Risk
- **OOM crash:** Application loads 50K Waves at startup (100 MiB of protobuf data + 150 MiB Go runtime overhead) = **250 MiB**
- **Exceeds target:** Violates 256 MiB budget before monitor even starts
- **No attribution:** Cannot tell if memory is from Waves, Pulse Map graph, Shroud circuits, or network buffers
- **No recovery:** Once memory exceeds budget, eviction may be too slow (LRU evicts 100 Waves/sec, but network receives 500/sec during viral event)

### Closing the Gap
**Immediate (1 week):**
1. Add startup budget check:
   ```go
   func (a *App) initContent() error {
       cache, err := storage.NewCache(a.subsystems.Storage)
       if err != nil {
           return err
       }
       
       // Check memory after cache load
       var m runtime.MemStats
       runtime.ReadMemStats(&m)
       if m.Alloc > 200*1024*1024 {
           log.Warn("memory budget exceeded at startup: %d MiB", m.Alloc/1024/1024)
           cache.EvictLRU(1000)  // Emergency eviction
       }
   }
   ```
2. Add Wave cache loading limit: `NewCache()` loads at most 10K most recent Waves, defers older Waves to lazy load

**Short-term (1 month):**
3. Add subsystem-level memory attribution:
   ```go
   type MemoryBudget struct {
       WaveCache    uint64  // Target: 50 MiB
       PulseMapGraph uint64  // Target: 20 MiB
       ShroudCircuits uint64  // Target: 10 MiB
       NetworkBuffers uint64  // Target: 20 MiB
       // ... other subsystems
   }
   ```
4. Add `pkg/store` instrumentation: track bytes stored in each Bbolt bucket
5. Reduce memory check interval from 60s → 10s during first 5 minutes

**Long-term (3 months):**
6. Add memory pressure signaling: when memory exceeds 180 MiB (90% budget), emit `MemoryPressureHigh` event → subsystems proactively evict
7. Add configurable memory budget: `--memory-limit=256M` flag, adjust eviction thresholds dynamically
8. Add memory profiling endpoint: `/debug/pprof/heap` for production debugging

**Validation:**
```bash
# Simulate large Wave cache
go test -v -run TestStartupMemoryBudget -count=1 ./pkg/app
# Populate DB with 50K Waves, start app, measure memory within 5s
# Expected: <200 MiB
```

---

## Summary Table

| Gap | Severity | Risk | Closes With | Validation |
|-----|----------|------|-------------|------------|
| **Gap 1:** No shutdown instrumentation | HIGH | Hung shutdowns, data loss | Per-subsystem timing + goroutine logging | `TestGracefulShutdown` <3s |
| **Gap 2:** No context cancellation tests | MEDIUM | Goroutine leaks on refactor | 28 exit latency tests | All pass <1s |
| **Gap 3:** No event bus load testing | MEDIUM | Silent event drops during viral content | Load test + priority channels | <1% drop at 10K events |
| **Gap 4:** No Shroud per-hop timeout | MEDIUM | 90s circuit build latency | 10s per-hop timeout + parallel dial | <30s with offline relay |
| **Gap 5:** No startup memory budget | MEDIUM | OOM at startup with large cache | Startup check + lazy Wave loading | <200 MiB after init |

---

## Relationship to AUDIT.md Findings

This document extends AUDIT.md by focusing on **architectural gaps** rather than code-level bugs:

- **AUDIT.md H1** (shutdown timeout) → **Gap 1** (no instrumentation to debug it)
- **AUDIT.md H2** (context propagation) → **Gap 2** (no tests to enforce it)
- **AUDIT.md M1** (event drops) → **Gap 3** (no load characterization)
- **AUDIT.md M4** (Shroud timeout) → **Gap 4** (no hop-level timeout design)
- *New:* **Gap 5** (memory budget) — identified during baseline analysis, not in AUDIT.md

---

**Report generated:** 2026-05-06T01:29:49Z  
**Project phase:** v0.1 Foundation (85-90% complete per README.md)  
**Concurrency model:** Channel-first with ~8 persistent goroutines  
**Largest gap:** Shutdown instrumentation (Gap 1) — blocking production release
