# MURMUR Tunneling Primitive — Design Specification

> **Version**: 0.1  
> **Status**: Design Phase  
> **Last Updated**: 2026-05-06  
> **Prerequisites**: Phase 4 (Anti-Abuse Framework) complete, Phase 5 (Tor/I2P Transport) complete

---

## Overview

MURMUR's tunneling primitive enables developers to expose localhost services to the internet through the Shroud onion routing network, similar to ngrok or PageKite, but with built-in anonymity guarantees. Tunnels leverage MURMUR's existing three-hop circuit infrastructure, treating tunnel traffic as a specialized form of anonymous content propagation.

### Design Principles

1. **Reuse Shroud infrastructure** — Tunnels are built on existing circuit construction, no new routing layer
2. **Operator anonymity by default** — Tunnel endpoints do not reveal the operator's real IP unless explicitly configured
3. **Abuse-aware from day one** — Content-type restrictions, bandwidth accounting, and takedown protocols integrated from v1
4. **Explicit opt-in** — Relay operators must actively enable tunnel support; not required for social MURMUR usage
5. **Clear threat model** — Users understand what tunnels protect (IP linkability) vs. what they don't (application-layer deanonymization)

---

## Use Cases

### Primary: Developer Localhost Exposure

**Scenario**: Alice is building a webhook-enabled app locally and needs a public URL for testing.

**Flow**:
1. Alice runs `murmur tunnel start --port 8080 --name alice-dev`
2. MURMUR constructs a three-hop Shroud circuit to a tunnel exit relay
3. Exit relay assigns a stable address: `murmur://tunnel/alice-dev-a3f2b9`
4. Alice shares the address; incoming HTTP requests route through Shroud to localhost:8080
5. Alice's real IP remains hidden from both the exit relay and external clients

**Benefit**: Same simplicity as ngrok, but with Shroud's IP unlinkability guarantees.

---

### Secondary: Friend-to-Friend Reseed Bootstrap

**Scenario**: Bob's country blocks MURMUR bootstrap nodes. Charlie (Bob's trusted friend) runs a tunnel to bridge Bob into the network.

**Flow**:
1. Charlie runs `murmur tunnel start --reseed --whitelist bob-identity-pubkey`
2. Bob receives Charlie's tunnel address out-of-band (QR code, encrypted message)
3. Bob's client connects via the tunnel, fetches peer list and bootstrap keys
4. Bob joins the network without hitting public bootstrap infrastructure

**Benefit**: Censorship-resistant onboarding without requiring Tor/I2P external dependencies.

---

### Tertiary: Private Service Hosting

**Scenario**: A friend group wants to host a collaborative document editor accessible only to group members, without renting a server.

**Flow**:
1. One group member runs the editor on localhost:3000
2. They create a tunnel with `--whitelist-identity` set to group member public keys
3. Group members access `murmur://tunnel/group-docs-xyz` — requests are authenticated at entry via Ed25519 signatures
4. Exit relay enforces the whitelist, rejecting non-member connections

**Benefit**: Zero-cost private hosting with group-level access control.

---

## Addressing Scheme

### Tunnel Address Format

```
murmur://tunnel/<tunnel-id>
```

- **Protocol**: `murmur://` (distinct from `http://` or `https://`)
- **Type**: `tunnel` (reserved type for tunneling primitive)
- **Tunnel ID**: 16-character base32 string derived from the tunnel operator's public key + tunnel name

**Example**: `murmur://tunnel/alice-dev-a3f2b9`

### Address Resolution

Clients resolve tunnel addresses via **Kademlia DHT lookup**:

1. Client queries DHT for `tunnel/<tunnel-id>`
2. DHT returns the current exit relay's peer ID and Shroud circuit metadata
3. Client constructs a Shroud circuit to the exit relay
4. Client sends tunnel connection request with tunnel ID
5. Exit relay forwards traffic to the operator's localhost service

### Address Stability

- **Persistent tunnels**: Tunnel ID remains constant across restarts (derived from operator's Ed25519 keypair)
- **Ephemeral tunnels**: Tunnel ID includes a timestamp; tunnel expires after 24 hours or on operator disconnect
- **Exit relay changes**: Tunnel ID does not change if the exit relay changes (DHT record updates point to new relay)

---

## Anonymity Model

### Operator Anonymity

**Guaranteed**:
- Exit relay **does not** learn the operator's real IP address (traffic routed through three-hop Shroud circuit)
- External clients **do not** learn the operator's IP address (only see the exit relay's IP)
- DHT queries for the tunnel address do not reveal the operator's identity (queries are standard Kademlia lookups)

**Not Guaranteed**:
- **Application-layer deanonymization**: If the localhost service leaks identifying information (e.g., default error pages, server headers, response timing), adversaries can correlate the tunnel with the operator
- **Global passive adversary**: A state-level adversary controlling a majority of Shroud relays can perform timing correlation attacks
- **Exit relay compromise**: A malicious exit relay can inspect plaintext HTTP traffic (mitigated by using HTTPS tunnels)

**Recommendation**: For high-risk scenarios, operators should route MURMUR itself through Tor/I2P (Mode B or Mode C from ANONYMITY_TRANSPORT_MODES.md), layering Shroud on top for double-hop protection.

---

### Client Anonymity

**Guaranteed**:
- Clients connecting to a tunnel through Shroud circuits are IP-unlinkable from the tunnel operator's perspective (operator sees only the exit relay's IP)
- External observers cannot correlate client identities with tunnel access (traffic is onion-routed)

**Not Guaranteed**:
- **Malicious tunnel operator**: The operator controls the localhost service and can log application-layer data (e.g., session cookies, user-agent strings)
- **Exit relay logging**: A malicious exit relay can correlate connection times and traffic patterns across multiple clients

**Recommendation**: Clients should only connect to tunnels operated by trusted parties, or use HTTPS to prevent exit relay inspection.

---

### Anonymity vs. Tor/I2P Hidden Services

| Property | MURMUR Tunnels | Tor Hidden Services | I2P Eepsites |
|---|---|---|---|
| **Operator IP hidden** | ✅ Yes (3-hop Shroud) | ✅ Yes (6-hop circuit) | ✅ Yes (multi-hop garlic) |
| **Exit relay knows content** | ⚠️ Yes (unless HTTPS) | ❌ No (E2E encrypted) | ❌ No (E2E encrypted) |
| **Resistant to global adversary** | ❌ No | ✅ Yes (with guards) | ✅ Yes (netDB isolation) |
| **Setup time** | ~3s (circuit construction) | ~60s (descriptor publish) | ~120s (tunnel build) |
| **Bandwidth overhead** | 3× (three hops) | 6× (six hops) | 5-8× (multi-hop garlic) |
| **Friend-to-friend capable** | ✅ Yes (whitelist support) | ⚠️ Via .onion ACLs | ⚠️ Via addressbook |

**Use Tunnels When**: Speed and MURMUR integration matter more than absolute anonymity (e.g., development, trusted friend groups).

**Use Tor/I2P When**: The threat model includes state-level adversaries or the service handles sensitive data requiring E2E encryption.

---

## Technical Architecture

### Components

1. **Tunnel Initiator** (`pkg/tunneling/initiator/`)
   - Constructs Shroud circuits to exit relays
   - Publishes tunnel metadata to DHT
   - Forwards localhost traffic into Shroud circuits

2. **Tunnel Exit Relay** (`pkg/tunneling/relay/`)
   - Accepts incoming tunnel circuits
   - Enforces content-type and hostname allowlists (per Phase 4 abuse policy)
   - Forwards plaintext traffic to external clients
   - Reports bandwidth accounting to initiator

3. **Tunnel Client** (`pkg/tunneling/client/`)
   - Resolves tunnel addresses via DHT
   - Constructs Shroud circuits to exit relays
   - Sends HTTP/WebSocket requests through circuits

4. **DHT Tunnel Registry** (extension of `pkg/networking/discovery/`)
   - Stores `(tunnel-id → exit-relay-peer-id)` mappings
   - Supports 24-hour expiry for ephemeral tunnels
   - Rate-limits DHT publish operations to prevent spam

---

### Tunnel Lifecycle

#### 1. Tunnel Creation

```go
// Operator runs: murmur tunnel start --port 8080 --name alice-dev
initiator := tunneling.NewInitiator(config.TunnelConfig{
    LocalPort:  8080,
    TunnelName: "alice-dev",
    Ephemeral:  false,
})
tunnelID := initiator.Start(ctx) // Returns "alice-dev-a3f2b9"
```

**Steps**:
1. Generate tunnel ID from operator's Ed25519 public key + tunnel name (BLAKE3 hash)
2. Construct a three-hop Shroud circuit to an available exit relay
3. Publish DHT record: `tunnel/alice-dev-a3f2b9 → exit-relay-peer-id`
4. Start localhost proxy: bind to `127.0.0.1:8080`, forward traffic into Shroud cells

---

#### 2. Client Connection

```go
// Client runs: murmur tunnel connect murmur://tunnel/alice-dev-a3f2b9
client := tunneling.NewClient("murmur://tunnel/alice-dev-a3f2b9")
resp, err := client.Get(ctx, "/api/status")
```

**Steps**:
1. Parse tunnel ID from address
2. Query DHT for `tunnel/alice-dev-a3f2b9` → resolve to exit relay peer ID
3. Construct Shroud circuit to exit relay
4. Send tunnel connection request (includes tunnel ID, HTTP method, path, headers)
5. Exit relay validates tunnel ID, looks up local registry, forwards to operator's circuit
6. Operator's localhost service processes request, sends response back through circuit

---

#### 3. Traffic Flow

```
[Client]
   ↓ HTTP GET /api/status
   ↓ (wrap in Shroud cell)
[Hop 1 Relay]
   ↓ (decrypt outer layer)
[Hop 2 Relay]
   ↓ (decrypt middle layer)
[Exit Relay]
   ↓ (decrypt inner layer, validate tunnel ID)
   ↓ (lookup operator's circuit by tunnel ID)
[Hop 3 Relay (operator's entry)]
   ↓ (encrypt response, send back)
[Hop 2 Relay (operator's middle)]
   ↓ (encrypt response)
[Hop 1 Relay (operator's exit → exit relay)]
   ↓ (encrypt response)
[Exit Relay]
   ↓ (re-encrypt for client circuit)
[Client]
   ← HTTP 200 OK
```

**Total hops**: 6 (three for client-to-exit, three for exit-to-operator)

**Latency**: ~1.5–4 seconds (3× Shroud circuit latency, depends on relay locations)

---

#### 4. Tunnel Shutdown

```go
initiator.Stop(ctx) // Gracefully closes circuit, removes DHT record
```

**Steps**:
1. Send graceful shutdown signal to exit relay
2. Close all active connections with 30-second drain period
3. Remove DHT record: `DELETE tunnel/alice-dev-a3f2b9`
4. Tear down Shroud circuit

---

### Wire Protocol

Tunnel requests use a specialized **Shroud cell type** (`CellType = 0x08 TUNNEL_REQUEST`):

```protobuf
message TunnelRequestCell {
  bytes tunnel_id = 1;           // 16-byte tunnel identifier
  string http_method = 2;        // GET, POST, etc.
  string http_path = 3;          // /api/status
  map<string, string> headers = 4; // HTTP headers
  bytes body = 5;                // Request body (chunked for large payloads)
  uint32 request_id = 6;         // Unique request ID (for response correlation)
}

message TunnelResponseCell {
  uint32 request_id = 1;         // Matches TunnelRequestCell.request_id
  uint32 http_status = 2;        // 200, 404, etc.
  map<string, string> headers = 3;
  bytes body = 4;                // Response body (chunked)
  bool final = 5;                // True if this is the last chunk
}
```

---

## Abuse Prevention (Phase 4 Integration)

### Content-Type Allowlists

Exit relays enforce **default-deny** policies for executable content:

```json
{
  "allowed_content_types": [
    "text/html",
    "text/plain",
    "application/json",
    "image/*"
  ],
  "blocked_content_types": [
    "application/x-executable",
    "application/octet-stream",
    "application/x-msdownload"
  ]
}
```

**Operators can opt-in** to serve executables by publishing a signed policy extension:

```json
{
  "operator_policy_override": {
    "allow_executables": true,
    "signed_by": "0x...",  // Operator's Ed25519 signature
    "valid_until": 1735689600  // Unix timestamp
  }
}
```

**Exit relays validate the signature** before accepting executable content. This creates an audit trail for abuse investigations.

---

### Hostname Allowlists (Reseed Use Case)

For reseed tunnels, operators can restrict allowed destination hostnames:

```json
{
  "allowed_hostnames": [
    "bootstrap.murmur.network",
    "fallback.murmur.network"
  ]
}
```

Exit relays reject requests to other hostnames (e.g., `malware-c2.example.com`).

---

### Bandwidth Accounting

Exit relays enforce **per-tunnel bandwidth caps**:

```json
{
  "max_bandwidth_per_tunnel_mb": 500,  // 500 MB per 24-hour period
  "burst_limit_mb": 10  // 10 MB burst before rate limiting
}
```

**Operator incentive**: Exit relays that accept tunnels can publish their capacity and earn **Resonance rewards** (see ABUSE_MODEL.md §4 for incentive design).

**Abuse mitigation**: Exit relays automatically terminate tunnels exceeding the cap and publish a **bandwidth abuse report** to the DHT (anonymized, includes tunnel ID but not operator identity).

---

### Automated Takedown Protocol

**Scenario**: An exit relay detects malware C2 traffic (periodic beacons, known malware signatures).

**Protocol**:
1. Exit relay generates a **takedown request** (includes tunnel ID, abuse category, evidence hash)
2. Exit relay signs the request with its peer ID and publishes to DHT under `takedown/<tunnel-id>`
3. Other exit relays query the DHT and **refuse to serve the tunnel** if the request is valid (signature verification, evidence plausibility)
4. Operator's tunnel becomes unreachable across the network within ~60 seconds

**Operator anonymity preserved**: The takedown request does not include the operator's IP or real identity, only the tunnel ID (which is already public via DHT).

**Appeal process**: Operators can publish a **dispute claim** with counter-evidence; exit relays re-evaluate after 24 hours.

---

## Security Considerations

### Threat 1: Malicious Exit Relay Logging

**Risk**: Exit relay operator logs all plaintext HTTP traffic passing through their node.

**Mitigation**:
- **Mandatory HTTPS**: Tunnel clients should always use HTTPS when connecting to untrusted tunnels (E2E encryption, exit relay sees only encrypted blobs)
- **Exit relay reputation**: Future work (v1.1+) will track exit relay uptime, abuse reports, and operator Resonance as a trust signal
- **Distributed exit selection**: Clients can specify multiple exit relays in a round-robin pool, reducing single-relay exposure

**Residual Risk**: An exit relay can still log connection metadata (tunnel ID, connection times, traffic volume). This is inherent to the exit relay role.

---

### Threat 2: Tunnel Operator Deanonymization via Application Fingerprints

**Risk**: A localhost service leaks identifying information (e.g., default CMS error pages, server version headers, unique typos in HTML).

**Mitigation**:
- **Documentation warning**: TUNNEL_DESIGN.md and CLI help text must explicitly state that tunnels hide IP, not application-layer identifiers
- **Header scrubbing**: MURMUR tunnel initiator should strip `Server`, `X-Powered-By`, and similar headers by default (opt-out via `--preserve-headers`)
- **Response normalization**: Future work (v1.2+) could add a `--sanitize` mode that rewrites HTML to remove fingerprints

**Residual Risk**: Sophisticated adversaries can still fingerprint via response timing, content size patterns, or JavaScript execution behavior.

---

### Threat 3: DHT Pollution (Spam Tunnel IDs)

**Risk**: Attacker publishes thousands of fake tunnel IDs to DHT, causing lookup failures or resource exhaustion.

**Mitigation**:
- **PoW gating**: DHT publish operations for tunnel records require PoW (difficulty 20, same as Wave creation)
- **Resonance gating**: Only Specters with Resonance ≥50 (Wraith tier) can publish tunnel records
- **Rate limiting**: DHT nodes limit tunnel publishes to 10 per hour per peer ID

**Residual Risk**: An attacker with high Resonance or distributed bot network can still publish spam, but economic cost (PoW + Resonance) makes it non-trivial.

---

### Threat 4: Circuit Correlation (Global Passive Adversary)

**Risk**: State-level adversary monitors all Shroud relay traffic, correlates tunnel connection times with operator activity times.

**Mitigation**:
- **Tor/I2P transport mode**: Operators in high-risk scenarios should run MURMUR in Mode B (Tor) or Mode C (I2P), adding 3–6 additional hops before Shroud circuits
- **Decoy traffic**: Future work (v1.3+) could add decoy tunnel requests (random noise at random intervals) to obfuscate real traffic patterns

**Residual Risk**: If the adversary controls both the client's ISP and the operator's ISP, correlation is still possible (Shroud alone does not protect against global passive adversaries).

---

## Performance Benchmarks (Estimates)

| Metric | ngrok (clearnet) | MURMUR Tunnel (Shroud only) | MURMUR Tunnel (Shroud over Tor) |
|---|---|---|---|
| **Setup time** | ~1s | ~3s | ~8s |
| **Latency (p50)** | 50ms | 600ms | 2000ms |
| **Latency (p99)** | 200ms | 2000ms | 6000ms |
| **Throughput** | ~100 Mbps | ~10 Mbps | ~2 Mbps |
| **Connection overhead** | 1× | 3× (three hops) | 9× (three Shroud + six Tor) |

**Conclusion**: MURMUR tunnels trade performance for anonymity. Suitable for development, webhooks, and low-traffic services. Not suitable for video streaming or high-frequency trading.

---

## Open Questions (Deferred to Implementation Phase)

1. **WebSocket support**: Should tunnels support persistent WebSocket connections, or only HTTP request/response? (WebSockets complicate circuit lifecycle management)

2. **Custom port forwarding**: Should tunnels support arbitrary TCP/UDP port forwarding (e.g., SSH, database connections), or restrict to HTTP/HTTPS? (Non-HTTP increases abuse risk)

3. **Multi-operator tunnels**: Can multiple operators share a single tunnel ID for load balancing? (Requires new DHT record structure and exit relay round-robin logic)

4. **Exit relay incentives**: Should exit relays earn Resonance for serving tunnels? If so, how to prevent Sybil attacks (operator spins up fake exit relay, routes to self, farms Resonance)? (See ABUSE_MODEL.md §4.5)

5. **IPv6 support**: Should exit relays support IPv6 destinations? (Most localhost services are IPv4-only, but future-proofing may be necessary)

6. **Circuit reuse**: Can a single Shroud circuit serve multiple tunnel requests from the same client? (Reduces circuit construction overhead but increases correlation risk)

7. **Tunnel discovery UI**: Should the Pulse Map show active tunnels as a separate layer/overlay? (UX question: tunnels are not social content, but visualizing them helps operators debug)

---

## Implementation Roadmap

### Phase 1: Core Tunnel Infrastructure (Weeks 1–4)

- [ ] `pkg/tunneling/initiator/` — Tunnel creation, localhost proxy, DHT publish
- [ ] `pkg/tunneling/relay/` — Exit relay, content-type enforcement, bandwidth accounting
- [ ] `pkg/tunneling/client/` — Tunnel address resolution, HTTP request tunneling
- [ ] `proto/tunnel.proto` — TunnelRequestCell, TunnelResponseCell, TunnelMetadata
- [ ] DHT extension for tunnel records (publish, lookup, expiry)
- [ ] CLI commands: `murmur tunnel start`, `murmur tunnel stop`, `murmur tunnel connect`

### Phase 2: Abuse Prevention Integration (Weeks 5–6)

- [ ] Content-type allowlist enforcement
- [ ] Hostname allowlist for reseed mode
- [ ] Bandwidth accounting and cap enforcement
- [ ] Takedown protocol (DHT-based, anonymity-preserving)
- [ ] Exit relay policy JSON schema and validation
- [ ] Operator policy override (signed executable allowlist)

### Phase 3: Testing & Documentation (Weeks 7–8)

- [ ] Unit tests: Tunnel lifecycle, cell encoding/decoding, DHT publish/lookup
- [ ] Integration tests: End-to-end tunnel creation, HTTP request/response, abuse policy enforcement
- [ ] Performance benchmarks: Latency p50/p99, throughput, circuit construction overhead
- [ ] User documentation: TUNNEL.md (user-facing guide), TUNNEL_OPERATOR.md (exit relay setup)
- [ ] Security documentation: Update THREAT_MODEL.md, SECURITY_PRIVACY.md with tunnel-specific threats

### Phase 4: Production Hardening (Weeks 9–10)

- [ ] Exit relay discovery (DHT query for relays with `tunnel_support=true` flag)
- [ ] Circuit failover (automatic retry on exit relay disconnect)
- [ ] Graceful degradation (fallback to direct HTTPS if Shroud circuit fails)
- [ ] Monitoring: Tunnel uptime, exit relay health, abuse report dashboard
- [ ] HTTPS enforcement (warn users when connecting to non-HTTPS tunnels)

---

## Success Criteria

1. **Functional**: A developer can expose `localhost:8080` and receive external HTTP requests in <60 seconds
2. **Secure**: Exit relay cannot learn operator's real IP address (verified via network trace analysis)
3. **Abuse-resistant**: Default-deny executable content blocks 95%+ of malware C2 attempts (based on simulated attack dataset)
4. **Performance**: p50 latency <1 second for HTTP GET requests (acceptable for webhooks, unacceptable for real-time apps)
5. **Adoption**: At least 50 active tunnels per month within three months of v1.0 release (indicates real-world utility)

---

## References

- **ABUSE_MODEL.md** — Tunnel abuse categories, mitigation levers, takedown protocol
- **SECURITY_PRIVACY.md** — Threat model, anonymity guarantees, attack surfaces
- **ANONYMITY_TRANSPORT_MODES.md** — Tor/I2P integration (Mode B/C for high-risk operators)
- **NETWORK_ARCHITECTURE.md** — Shroud circuit construction, relay selection, cell types
- **TECHNICAL_IMPLEMENTATION.md** — Concurrency model, goroutine lifecycle, performance targets

---

## Changelog

- **2026-05-06**: Initial draft (Phase 6.1 deliverable per PLAN.md)

---

**Status**: ✅ Design complete, ready for Phase 6.2 (abuse policy) and Phase 6.3 (prototype implementation)
