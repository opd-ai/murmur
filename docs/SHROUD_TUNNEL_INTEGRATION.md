# Shroud-Based Tunneling — Phase 6.4 Implementation Plan

**Status**: Design Phase  
**Date**: 2026-05-06  
**Prerequisites**: Phase 6.3 complete (single-hop prototype validated)

---

## Overview

Phase 6.4 extends the minimal single-hop tunneling prototype to use Shroud's three-hop onion routing infrastructure. This transforms tunnels from direct TCP connections into anonymity-preserving circuits, preventing exit relays from learning operator IP addresses.

### Goals

1. **Reuse Shroud infrastructure** — Leverage existing `Circuit` type from `pkg/anonymous/shroud`
2. **Operator IP anonymity** — Exit relay cannot learn initiator's real IP
3. **Traffic accounting separation** — Tunnel traffic tracked separately from social Waves/Specters
4. **Graceful fallback** — Single-hop mode available when Shroud relays insufficient
5. **Performance targets** — <5s tunnel setup, <1s p50 latency for HTTP requests

---

## Architecture Changes

### Current (Phase 6.3): Single-Hop TCP

```
Initiator → (TCP) → Exit Relay → (HTTP forward) → Client
          REGISTER <tunnel-id>
```

- Direct TCP connection
- Exit relay sees initiator's IP
- ~200ms setup, ~50ms latency

### Target (Phase 6.4): Three-Hop Shroud

```
Initiator → Hop1 → Hop2 → Hop3 (Exit) → Client
          [Shroud Circuit with XChaCha20-Poly1305 layers]
          TunnelRegisterCell → TunnelDataCell streams
```

- Shroud circuit with 3-hop onion routing
- Exit relay sees only Hop2's IP
- ~3s setup (circuit construction), ~600ms latency (3 hops + encryption overhead)

---

## Implementation Phases

### Phase 6.4.1: Shroud Circuit Integration (Core)

**Duration**: 3 days  
**Files Modified**: `pkg/tunneling/initiator/initiator.go`, `pkg/tunneling/relay/relay.go`

#### Changes

1. **Replace direct TCP dial with circuit construction**
   ```go
   // OLD: conn, err := dialer.DialContext(ctx, "tcp", i.config.ExitRelayAddr)
   // NEW: circuit, err := shroud.BuildCircuitToRelay(ctx, exitRelayPeerID)
   ```

2. **Wrap tunnel protocol in Shroud cells**
   - `TunnelRegisterCell`: Tunnel registration (replaces `REGISTER <id>\n`)
   - `TunnelDataCell`: HTTP request/response forwarding (replaces raw TCP)
   - `TunnelTeardownCell`: Graceful disconnect (replaces `UNREGISTER <id>\n`)

3. **Circuit lifecycle management**
   - Detect circuit failures (hop timeouts, decryption errors)
   - Automatic circuit rebuild on failure
   - Graceful degradation to single-hop if <3 relays available

#### Dependencies

- `pkg/anonymous/shroud.Beacon` (relay discovery)
- `pkg/anonymous/shroud.Circuit` (onion encryption)
- libp2p stream protocol `/murmur/tunnel/1` (circuit-to-circuit communication)

#### Testing

- Unit tests: Circuit construction with mock relays
- Integration tests: End-to-end tunnel via in-memory Shroud circuits
- Latency benchmarks: Compare 1-hop vs 3-hop overhead

---

### Phase 6.4.2: Traffic Accounting (Separation)

**Duration**: 2 days  
**Files New**: `pkg/tunneling/accounting/`, `pkg/tunneling/accounting/accounting.go`

#### Changes

1. **Per-tunnel traffic counters**
   ```go
   type TunnelMetrics struct {
       BytesSent     uint64
       BytesReceived uint64
       RequestCount  uint64
       ErrorCount    uint64
   }
   ```

2. **Separate Prometheus metrics**
   - `tunnel_bytes_sent_total{tunnel_id}` (distinct from Wave bytes)
   - `tunnel_requests_total{tunnel_id}`
   - `tunnel_errors_total{tunnel_id, error_type}`
   - `tunnel_circuit_rebuild_total{tunnel_id}`

3. **Bandwidth quota enforcement**
   - Per-tunnel daily limits (default 500 MB/day per TUNNEL_ABUSE_POLICY.md)
   - Graceful teardown when exceeded (send quota warning to client)
   - Operator-configurable limits via config

#### Testing

- Unit tests: Accounting logic with mock traffic
- Integration tests: Quota enforcement triggers teardown
- Metrics validation: Prometheus scrape includes tunnel metrics

---

### Phase 6.4.3: Protobuf Cell Definitions

**Duration**: 1 day  
**Files New**: `proto/tunnel.proto`

#### Protobuf Schema

```protobuf
syntax = "proto3";
package murmur.tunnel;

// TunnelRegisterCell initiates a tunnel at the exit relay.
message TunnelRegisterCell {
    bytes tunnel_id = 1;           // 16-byte tunnel ID (BLAKE3 derived)
    bytes operator_pubkey = 2;     // Ed25519 public key (32 bytes)
    bytes signature = 3;           // Ed25519 signature over (tunnel_id || timestamp)
    int64 timestamp_unix = 4;      // Unix timestamp (replay protection)
    uint64 bandwidth_limit = 5;    // Requested daily bandwidth quota (bytes)
}

// TunnelDataCell carries HTTP request/response data.
message TunnelDataCell {
    bytes tunnel_id = 1;           // Which tunnel this data belongs to
    bytes payload = 2;             // Raw HTTP message bytes
    uint32 sequence = 3;           // Sequence number (ordering)
    bool is_final = 4;             // Last fragment in stream
}

// TunnelTeardownCell closes a tunnel gracefully.
message TunnelTeardownCell {
    bytes tunnel_id = 1;           // Tunnel to close
    TeardownReason reason = 2;     // Why closing
    string message = 3;            // Human-readable reason
}

enum TeardownReason {
    OPERATOR_REQUEST = 0;          // Normal shutdown
    QUOTA_EXCEEDED = 1;            // Bandwidth limit hit
    ABUSE_DETECTED = 2;            // Policy violation
    CIRCUIT_FAILURE = 3;           // Shroud circuit collapsed
}
```

#### Changes

- Generate Go code: `protoc --go_out=. proto/tunnel.proto`
- Check in generated `proto/tunnel.pb.go`
- Update `Makefile` with protobuf generation target

#### Testing

- Unit tests: Protobuf serialization round-trips
- Integration tests: Cell marshaling/unmarshaling over circuits

---

### Phase 6.4.4: libp2p Stream Protocol

**Duration**: 2 days  
**Files New**: `pkg/tunneling/protocol/`, `pkg/tunneling/protocol/stream.go`

#### Changes

1. **Define stream protocol `/murmur/tunnel/1`**
   - Bidirectional streams for tunnel data
   - One stream per tunnel connection
   - Multiplexed via yamux over libp2p connection

2. **Stream lifecycle**
   ```go
   // Initiator side
   stream, err := host.NewStream(ctx, exitRelayPeerID, protocol.ID("/murmur/tunnel/1"))
   // Send TunnelRegisterCell
   // Stream TunnelDataCell bidirectionally
   // Send TunnelTeardownCell on close
   
   // Exit relay side
   host.SetStreamHandler(protocol.ID("/murmur/tunnel/1"), handleTunnelStream)
   // Read TunnelRegisterCell
   // Forward TunnelDataCell to localhost
   // Handle TunnelTeardownCell
   ```

3. **Error handling**
   - Stream reset detection → circuit rebuild
   - Timeout detection → send error cell to client
   - Backpressure handling (yamux flow control)

#### Testing

- Unit tests: Stream lifecycle with mock libp2p hosts
- Integration tests: Multi-stream concurrency (10 simultaneous tunnels)
- Performance tests: Stream overhead vs raw TCP

---

### Phase 6.4.5: Fallback & Error Recovery

**Duration**: 1 day  
**Files Modified**: `pkg/tunneling/initiator/initiator.go`

#### Changes

1. **Relay count detection**
   ```go
   relayCount := shroud.Beacon.RelayCount()
   if relayCount < 3 {
       log.Warn("Insufficient relays for Shroud circuit, using single-hop mode")
       return initiateSingleHop(ctx)
   }
   ```

2. **Circuit failure recovery**
   - Detect `ErrCircuitClosed`, `ErrDecryptionFailed`, `ErrRelayFailure`
   - Attempt rebuild with different relays (max 3 retries)
   - Fallback to single-hop after exhausting retries
   - Notify client of degraded anonymity

3. **Graceful degradation**
   - Log warnings when falling back
   - Update tunnel metrics (circuit_rebuild_total)
   - Client receives warning: "Tunnel using single-hop mode (exit relay can see your IP)"

#### Testing

- Unit tests: Fallback triggers correctly
- Integration tests: Recover from relay failures
- Chaos tests: Random relay crashes during tunnel operation

---

## Performance Estimates

| Metric | Single-Hop (6.3) | Three-Hop (6.4 Target) | Overhead |
|--------|------------------|------------------------|----------|
| Setup time | ~200ms | ~3s | 15x |
| Latency (p50) | ~50ms | ~600ms | 12x |
| Latency (p99) | ~100ms | ~1200ms | 12x |
| Throughput | ~100 Mbps | ~10 Mbps | 10x |

**Rationale**: Each hop adds ~200ms latency (relay processing + network RTT). Encryption overhead (XChaCha20-Poly1305) is ~10% throughput penalty per layer.

---

## Success Criteria

- [ ] Initiator successfully constructs Shroud circuit to exit relay
- [ ] Tunnel registration via TunnelRegisterCell protobuf
- [ ] End-to-end HTTP request/response through 3-hop circuit
- [ ] Exit relay cannot learn initiator IP (verified via packet capture)
- [ ] Traffic accounting separates tunnel bytes from social bytes
- [ ] Graceful fallback to single-hop when <3 relays available
- [ ] Circuit rebuild on relay failure
- [ ] All tests pass with `-race` detector (zero race conditions)
- [ ] Latency p50 < 1s, p99 < 2s (acceptable for tunneling use case)

---

## Open Questions

1. **DHT integration timing**: When should tunnel IDs be published to DHT? (Deferred to Phase 6.5)
2. **Relay incentives**: Should tunnel operators receive Resonance rewards? (Deferred to Phase 6.5)
3. **Multi-circuit load balancing**: Use multiple circuits for high-bandwidth tunnels? (Deferred to v1.1)
4. **Tor/I2P interop**: Can tunnels route over Tor/I2P transports? (Deferred to Phase 6.6)

---

## Implementation Sequence

1. ✅ Phase 6.3: Single-hop prototype (complete)
2. **Phase 6.4.1**: Shroud circuit integration (3 days) ← START HERE
3. **Phase 6.4.2**: Traffic accounting (2 days)
4. **Phase 6.4.3**: Protobuf cell definitions (1 day)
5. **Phase 6.4.4**: libp2p stream protocol (2 days)
6. **Phase 6.4.5**: Fallback & error recovery (1 day)
7. **Phase 6.4.6**: End-to-end validation (1 day)

**Total**: ~10 days (1 engineer, full-time)

---

## References

- `TUNNEL_DESIGN.md` — High-level tunneling architecture
- `TUNNEL_ABUSE_POLICY.md` — Abuse prevention requirements
- `pkg/anonymous/shroud/circuit.go` — Shroud circuit implementation
- `SECURITY_PRIVACY.md §4` — Shroud anonymity guarantees
- `TECHNICAL_IMPLEMENTATION.md §4.5` — Shroud circuit lifecycle
