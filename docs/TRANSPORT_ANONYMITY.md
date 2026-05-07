# Transport Anonymity in MURMUR

This document explains MURMUR's transport layer anonymity architecture, the integration of Tor and I2P via `go-i2p/onramp`, and the relationship between Shroud's unlinkability guarantees and transport-level anonymity networks.

**Status**: Production-ready (v0.1). Tor and I2P transports fully integrated. User-facing modes documented in `docs/ANONYMITY_TRANSPORT_MODES.md`.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [The Adapter Model](#the-adapter-model)
3. [go-i2p/onramp Dependency](#go-i2p-onramp-dependency)
4. [Shroud vs. Tor/I2P: Distinct Guarantees](#shroud-vs-tor-i2p-distinct-guarantees)
5. [Transport Modes](#transport-modes)
6. [When to Use Each Mode](#when-to-use-each-mode)
7. [Implementation Details](#implementation-details)
8. [Security Considerations](#security-considerations)
9. [References](#references)

---

## Architecture Overview

MURMUR's transport layer is built on **libp2p**, which provides a pluggable transport system. MURMUR supports four transport types:

1. **Clearnet** (TCP, QUIC, WebSocket, WebRTC) — default, zero dependencies, IP visible to peers
2. **Tor** (.onion hidden services) — requires Tor daemon, hides IP via 3-hop onion routing
3. **I2P** (.i2p destinations) — requires I2P router with SAMv3, hides IP via darknet tunnels
4. **Hybrid** — all above enabled simultaneously, peers choose based on availability

All four transports coexist within libp2p's transport fallback chain. When dialing a peer, libp2p tries all available transports in order and uses the first success.

**Key Insight**: Shroud (MURMUR's internal onion routing network) can layer **on top of** any transport. For example, "Shroud over Tor" provides two layers of unlinkability: Tor hides your IP from the MURMUR network, and Shroud hides which MURMUR identities you're talking to.

---

## The Adapter Model

MURMUR uses **libp2p transport adapters** to integrate Tor and I2P without special-casing in application code. Each adapter implements libp2p's `transport.Transport` interface:

```go
type Transport interface {
    Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (CapableConn, error)
    Listen(laddr ma.Multiaddr) (Listener, error)
    CanDial(addr ma.Multiaddr) bool
    Protocols() []int // Multiaddr protocol codes
    Proxy() bool
}
```

**Tor Adapter** (`pkg/networking/transport/onramp_tor/`):
- Wraps `onramp.Onion` struct for Tor control protocol communication
- `Dial`: Connects to remote `.onion` address via Tor network
- `Listen`: Creates hidden service, returns `.onion` multiaddr
- Multiaddr format: `/onion3/<base32-pubkey>:<port>`
- Protocol code: `ma.P_ONION3` (445)

**I2P Adapter** (`pkg/networking/transport/onramp_i2p/`):
- Wraps `onramp.Garlic` struct for I2P SAM bridge communication
- `Dial`: Connects to remote `.i2p` destination via I2P network
- `Listen`: Creates I2P destination, returns `.i2p` multiaddr
- Multiaddr format: `/garlic64/<base64-destination>:<port>`
- Protocol code: `ma.P_GARLIC64` (446)

**Integration** (`pkg/networking/transport/host.go`):
- `appendAnonymityTransports()` conditionally registers Tor and I2P adapters
- `buildTorTransportOption()` constructs libp2p.Transport option for Tor
- `buildI2PTransportOption()` constructs libp2p.Transport option for I2P
- Config flags: `EnableTor`, `EnableI2P`, `TorControlAddr` (default 127.0.0.1:9051), `I2PSAMAddr` (default 127.0.0.1:7656)

**Multi-Transport Coexistence**:
- Peers advertise all available addresses (TCP, QUIC, onion3, garlic64)
- Dialers try transports in order of preference: QUIC → TCP → WebSocket → WebRTC → Tor → I2P
- No special application logic required — libp2p handles selection automatically

---

## go-i2p/onramp Dependency

**Library**: [`github.com/go-i2p/onramp`](https://github.com/go-i2p/onramp) v0.33.92  
**License**: MIT (permissive)  
**Purpose**: High-level Tor and I2P integration for Go applications

`onramp` provides two key structs:

1. **`Onion`** — Tor hidden service lifecycle management
   - `NewOnion(ctx, controlAddr, options...)` constructs instance
   - `Listen(port)` creates hidden service, returns `.onion` address
   - `Dial(network, addr)` connects via Tor circuit
   - `Close()` releases Tor control port resources
   - Key persistence: Tor daemon manages Ed25519 keypair for hidden service

2. **`Garlic`** — I2P destination lifecycle management
   - `NewGarlic(ctx, tunName, samAddr, options...)` constructs instance
   - `Listen()` creates I2P destination, returns base64-encoded destination
   - `Dial(network, destination)` connects via I2P tunnels
   - `Close()` closes SAM session
   - Key persistence: I2P router manages destination keys (Ed25519 + ElGamal)

**Why onramp?**
- Handles daemon detection, connection management, and protocol negotiation
- Abstracts differences between Tor control protocol and I2P SAMv3
- Production-tested (used in other Go anonymity projects)
- Active maintenance (latest commit 2026-03)

**Runtime Expectations**:
- **Tor**: Requires Tor daemon with control port (9051). Install: `apt install tor` (Linux), `brew install tor` (macOS), or download from torproject.org.
- **I2P**: Requires I2P router with SAMv3 enabled (port 7656). Install: i2pd from i2pd.website or java-i2p from geti2p.net. Config: `sam.enabled=true` (i2pd).
- Both daemons must be running **before** MURMUR starts (fail-fast design, see `pkg/networking/transport/diagnostics/`)

---

## Shroud vs. Tor/I2P: Distinct Guarantees

**Shroud** (MURMUR's internal onion routing network):
- **What it hides**: Which MURMUR identities you're communicating with
- **What it doesn't hide**: Your IP address from direct MURMUR peers (clearnet mode)
- **How it works**: 3-hop onion-encrypted circuits through other MURMUR nodes
- **Threat model**: Network-level observers trying to correlate MURMUR identities with social graphs
- **Does NOT protect against**: Your ISP seeing you use MURMUR, global passive adversary correlating timing

**Tor**:
- **What it hides**: Your IP address from all MURMUR peers and destination services
- **What it doesn't hide**: The fact you're using Tor (traffic fingerprinting), timing correlations with global adversary
- **How it works**: 3-hop onion routing through Tor relay network (separate from MURMUR)
- **Threat model**: State-level censorship, ISP surveillance, network-level metadata collection
- **Does NOT protect against**: Global passive adversary with 51%+ relay control, advanced traffic analysis

**I2P**:
- **What it hides**: Your IP address via darknet design (no exit nodes)
- **What it doesn't hide**: Timing correlations with peers in same I2P network
- **How it works**: Unidirectional tunnels (inbound/outbound separate), encrypted at transport layer
- **Threat model**: Peer-to-peer metadata privacy, resistance to Sybil attacks via NetDB
- **Does NOT protect against**: Global passive adversary, correlation attacks within I2P network

**Layering Shroud over Tor/I2P**:
- **"Shroud over Tor"**: Your IP is hidden from all MURMUR peers (Tor layer), and your MURMUR identity graph is unlinkable (Shroud layer). Two independent anonymity systems.
- **"Shroud over I2P"**: Similar layering, with I2P's darknet properties (no clearnet exit risk).
- **Advantage**: Compromise of Shroud circuit doesn't reveal your IP (Tor/I2P still protect). Compromise of Tor/I2P doesn't reveal which MURMUR identities you talk to (Shroud still protects).
- **Tradeoff**: Higher latency (Tor: 500-2000ms circuit setup, Shroud: +200-600ms/hop).

---

## Transport Modes

MURMUR exposes four user-facing transport modes (see `docs/ANONYMITY_TRANSPORT_MODES.md`):

| Mode | Description | Latency | IP Privacy | Requires |
|------|-------------|---------|------------|----------|
| **A (Default)** | Shroud over clearnet (TCP/QUIC) | 20-200ms | ❌ | Nothing |
| **B (Tor)** | Shroud over Tor (.onion) | 500-2000ms | ✅ | Tor daemon |
| **C (I2P)** | Shroud over I2P (.i2p) | 300-1500ms | ✅ | I2P router |
| **D (Both)** | Shroud over Tor **or** I2P | Worst of B/C | ✅ | Both daemons |

**Mode Selection** (Onboarding Phase 3):
- Users choose privacy level during setup
- Plain-language explanations (no jargon)
- Latency expectations and daemon installation links provided
- Mode persisted in `~/.murmur/config.toml`

**Mode Switching** (Settings Panel):
- Runtime mode change requires application restart
- Confirmation dialog warns about connection reset
- Daemon reachability checked before mode activation (see `pkg/networking/transport/diagnostics/`)

---

## When to Use Each Mode

### Mode A (Default/Clearnet) — Recommended for most users
**Use when**:
- You trust your ISP and government
- Latency matters (gaming, voice chat, real-time collaboration)
- You want MURMUR to work out-of-box with zero setup

**Protection**:
- ✅ Shroud hides which MURMUR identities you talk to
- ✅ End-to-end encryption (Noise protocol)
- ❌ ISP sees you use MURMUR (but not message content or recipient identities)
- ❌ Direct MURMUR peers see your IP

**Example**: Friend group in low-censorship country (US, EU, Canada). Privacy concerns are "hide our conversations from platforms", not "hide from state-level surveillance".

---

### Mode B (Tor) — Recommended for censorship resistance
**Use when**:
- Your government blocks/monitors MURMUR
- Your ISP logs connection metadata
- You want IP-level anonymity from all MURMUR peers
- You can tolerate 2-5× latency increase

**Protection**:
- ✅ All Mode A protections
- ✅ IP hidden from MURMUR peers (they see Tor exit)
- ✅ ISP sees "Tor usage" (not MURMUR specifically)
- ❌ Global passive adversary can correlate timing (requires Tor-level mitigations)

**Example**: Activist in authoritarian country. ISP logs all connections. Needs to hide both MURMUR usage and identity graph from state surveillance.

**Setup**: Install Tor daemon → Enable Mode B → MURMUR auto-verifies Tor on port 9051

---

### Mode C (I2P) — Recommended for I2P community users
**Use when**:
- You already use I2P for other services
- You prefer darknet design (no exit nodes)
- You value better UDP support than Tor
- You can tolerate moderate latency increase

**Protection**:
- ✅ Similar to Mode B (IP hidden)
- ✅ Darknet design: all traffic stays within I2P network
- ✅ Better resistance to traffic analysis (unidirectional tunnels)
- ❌ Smaller relay pool than Tor (fewer users, higher correlation risk)

**Example**: Privacy-conscious user already running I2P for file sharing. Wants MURMUR to integrate with existing I2P setup.

**Setup**: Install I2P router (i2pd or java-i2p) → Enable SAMv3 on port 7656 → Enable Mode C

---

### Mode D (Both) — Recommended for highest threat model
**Use when**:
- Your threat model includes state-level censorship **and** traffic analysis
- You need redundancy (if Tor blocked, fall back to I2P)
- Latency is acceptable tradeoff for maximum anonymity
- You have resources to run both daemons

**Protection**:
- ✅ All Mode B and C protections
- ✅ Censorship resistance: hard to block both Tor **and** I2P simultaneously
- ✅ Redundancy: If one network compromised, other still protects
- ❌ Worst latency experience (slowest network determines perf)

**Example**: Journalist in hostile environment. State-level adversary with advanced traffic analysis. Needs maximum anonymity and censorship resistance.

**Setup**: Install Tor daemon **and** I2P router → Enable Mode D → MURMUR connects via both

---

## Implementation Details

### Host Construction
`pkg/networking/transport/host.go`:`NewHost(ctx, cfg)` builds libp2p host with transports:

```go
func NewHost(ctx context.Context, cfg Config) (host.Host, error) {
    // Base transports: TCP, QUIC, WebSocket, WebRTC
    opts := []libp2p.Option{
        libp2p.Transport(tcp.NewTCPTransport),
        libp2p.Transport(libp2pquic.NewTransport),
        libp2p.Transport(websocket.New),
        libp2p.Transport(libp2pwebrtc.New),
    }

    // Conditionally add Tor and I2P
    opts, err := appendAnonymityTransports(opts, ctx, cfg)
    if err != nil {
        return nil, err
    }

    return libp2p.New(opts...)
}
```

### Startup Diagnostics
`pkg/networking/transport/diagnostics/`:`performDiagnostics(ctx, cfg)` validates daemons before host creation:

```go
func performDiagnostics(ctx context.Context, cfg config.Config) error {
    statuses, err := diagnostics.CheckAll(cfg, 5*time.Second)
    if err != nil {
        return fmt.Errorf("transport diagnostics failed: %w", err)
    }

    for _, status := range statuses {
        if !status.Reachable {
            return fmt.Errorf("%s transport unreachable: %s", status.Name, status.Error)
        }
    }
    return nil
}
```

**Failure Modes**:
- Tor unreachable → "Tor daemon not responding on 127.0.0.1:9051. Install: apt install tor or torproject.org. Ensure daemon is running with control port enabled."
- I2P unreachable → "I2P router not responding on 127.0.0.1:7656. Download i2pd from i2pd.website or java-i2p from geti2p.net. Enable SAM bridge: sam.enabled=true in config."

### Key Persistence
- **Tor**: Tor daemon manages Ed25519 keypair for hidden service. Key file location: `/var/lib/tor/murmur-onion/hs_ed25519_secret_key`. Survives restarts via Tor's HiddenServiceDir.
- **I2P**: I2P router manages destination keys (Ed25519 + ElGamal). Stored in SAM bridge state. MURMUR references by tunnel name (`murmur-i2p`).
- **Future**: Integrate with MURMUR's Argon2id keystore (`pkg/identity/keys/`) for unified key management.

### Testing
`pkg/networking/transport/integration_test.go` (build tag `integration`):
- `TestTorReachability`: Verifies diagnostics.CheckTor() probes Tor control port
- `TestI2PReachability`: Verifies diagnostics.CheckI2P() probes I2P SAM bridge
- `TestHostCreationWithTor`: Creates libp2p host with Tor, validates onion3 multiaddr
- `TestHostCreationWithI2P`: Creates libp2p host with I2P, validates garlic64 multiaddr
- `TestHostCreationWithBoth`: Validates multi-transport coexistence
- `TestFallbackToClearnet`: Validates fail-fast when daemons unreachable

Run with: `go test -tags=integration -short ./pkg/networking/transport/`  
Skip daemon-dependent tests: `-short` flag skips external dependencies

---

## Security Considerations

### Threat Model Alignment
- **Mode A**: Protects against "who-talks-to-whom" correlation within MURMUR. Does NOT protect against ISP-level surveillance.
- **Mode B/C**: Protects against network-level metadata observers, ISP surveillance, censorship. Does NOT protect against global passive adversary with 51%+ relay control.
- **Mode D**: Maximum protection within MURMUR's threat model. Still vulnerable to global passive adversary (requires Tor/I2P-level mitigations).

### Attack Surfaces
1. **Tor Daemon Compromise**: Attacker gains Tor control port access → can read hidden service keys → deanonymizes MURMUR host. Mitigation: Tor control port access control (cookie auth, firewall).
2. **I2P Router Compromise**: Attacker gains SAM bridge access → can read I2P destination keys → deanonymizes MURMUR host. Mitigation: SAM bridge access control (local-only binding).
3. **Timing Correlation**: Global passive adversary observes Tor/I2P entry **and** MURMUR Shroud circuits → correlates timing. Mitigation: Traffic padding (future work), Tor Bridges/I2P tunnels.
4. **Shroud Circuit Compromise**: Attacker controls 3/3 Shroud hops → learns identity graph. Mitigation: Tor/I2P transport hides correlation with IP.

### Key Management
- **Separation**: Tor/I2P keys are **separate** from MURMUR Surface/Specter identity keys. Compromise of one does NOT reveal the other.
- **Backup**: Tor/I2P keys managed by external daemons. MURMUR's BIP-39 recovery does NOT restore .onion/.i2p addresses. Users must backup Tor/I2P keys separately.
- **Rotation**: Tor hidden service rotation not yet implemented. I2P destination rotation requires SAM bridge reconfiguration. Future: Integrate with MURMUR's key rotation (see `docs/KEY_ROTATION.md`).

### Denial of Service
- **Tor Circuit Overload**: Mode B users share global Tor network bandwidth. High MURMUR usage can degrade Tor performance for other users. Mitigation: Rate limiting, circuit rotation.
- **I2P Tunnel Exhaustion**: Mode C users consume I2P tunnel slots. Shroud circuit construction may fail if I2P tunnels saturated. Mitigation: Tunnel count configuration, circuit retry logic.

### Censorship Resistance
- **Tor Bridges**: Mode B supports Tor bridges (pluggable transports) for countries blocking Tor. Config: `TorBridges` in `config.toml`. Future: UI for bridge configuration.
- **I2P Reseed**: Mode C requires initial I2P NetDB reseed. In hostile networks, reseed may be blocked. Mitigation: Manual reseed via I2P router config.
- **Mode D Redundancy**: If one network blocked, other still functions. Requires both daemons available at startup.

---

## References

**MURMUR Documentation**:
- `docs/ANONYMITY_TRANSPORT_MODES.md` — User-facing mode selection guide
- `SECURITY_PRIVACY.md` — Comprehensive threat model and privacy guarantees
- `THREAT_MODEL.md` — Primary/secondary adversary definitions
- `NETWORK_ARCHITECTURE.md` — libp2p foundation, transport layer, peer discovery

**External Resources**:
- Tor Project: [torproject.org](https://www.torproject.org/)
- I2P Project: [geti2p.net](https://geti2p.net/)
- i2pd (lightweight I2P router): [i2pd.website](https://i2pd.website/)
- go-i2p/onramp: [github.com/go-i2p/onramp](https://github.com/go-i2p/onramp)
- libp2p Transport Interface: [libp2p/go-libp2p/core/transport](https://pkg.go.dev/github.com/libp2p/go-libp2p/core/transport)

**Academic Papers**:
- Tor Design: [Dingledine et al., "Tor: The Second-Generation Onion Router", USENIX 2004](https://svn-archive.torproject.org/svn/projects/design-paper/tor-design.pdf)
- I2P Network Database: [Timpanaro et al., "Tracking Hidden Services: Correlation Attacks on the I2P Network", arXiv 2011](https://arxiv.org/abs/1109.1974)
- Traffic Analysis: [Murdoch & Danezis, "Low-Cost Traffic Analysis of Tor", IEEE S&P 2005](https://www.cl.cam.ac.uk/~sjm217/papers/oakland05torta.pdf)

---

**Last Updated**: 2026-05-06  
**Version**: v0.1-foundation  
**Maintainer**: GitHub Copilot CLI (Autonomous Mode)
