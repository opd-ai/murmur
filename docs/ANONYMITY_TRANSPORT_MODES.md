# MURMUR Anonymity Transport Modes

> **Document Status**: Design Specification for PLAN.md §5.6  
> **Last Updated**: 2026-05-06  
> **Implementer**: Autonomous Execution Task Loop

---

## Overview

MURMUR supports four **Anonymity Transport Modes** that determine how the application routes network traffic. Each mode provides different tradeoffs between **anonymity strength**, **latency**, **reachability**, and **failure modes**. The modes control which libp2p transports are enabled and how Shroud onion routing layers over them.

Per PLAN.md §5.5, the transport infrastructure now supports:
- **Clearnet**: TCP, QUIC, WebSocket, WebRTC (standard libp2p transports)
- **Tor**: Onion v3 hidden services via `pkg/networking/transport/onramp_tor`
- **I2P**: Garlic routing via `pkg/networking/transport/onramp_i2p`

Per THREAT_MODEL.md, Shroud's three-hop onion routing provides **intra-MURMUR unlinkability** (who-talks-to-whom within MURMUR is hidden from network observers). Tor and I2P provide **external anonymity** (your IP address is hidden from the internet). Using both provides **defense in depth**.

---

## Mode Definitions

### Mode A: Default (Shroud over Clearnet)

**Configuration**:
```go
Config{
    EnableTor: false,
    EnableI2P: false,
}
```

**Transports Enabled**: TCP, QUIC, WebSocket, WebRTC  
**Shroud**: Enabled (three-hop circuits within MURMUR mesh)  
**IP Visibility**: Your IP is visible to direct MURMUR peers you connect to  
**External Anonymity**: None (your ISP can see you're using MURMUR)  

**Use When**:
- You trust your government/ISP not to surveil MURMUR usage
- Low latency is critical (gaming, real-time chat)
- You're using MURMUR in a privacy-friendly jurisdiction
- Your threat model is limited to "hide who I talk to within MURMUR"

**Tradeoffs**:
- ✅ **Latency**: Excellent (20-200ms peer-to-peer)
- ✅ **Reachability**: Excellent (works behind most NATs via DCUtR)
- ✅ **Simplicity**: No external daemon dependencies
- ❌ **IP Privacy**: Your IP is visible to direct peers
- ❌ **ISP Visibility**: ISP knows you use MURMUR (but not who you talk to)
- ❌ **Global Adversary**: No protection against traffic correlation

**Plain-Language Description**:  
> "Your messages are encrypted and routed through three MURMUR peers so no one can see who you're talking to, but your internet provider can see that you're using MURMUR. Think of it like WhatsApp: your messages are private, but the app isn't hidden."

---

### Mode B: Tor (Shroud over Tor)

**Configuration**:
```go
Config{
    EnableTor: true,
    EnableI2P: false,
    TorControlAddr: "127.0.0.1:9051", // Tor control port
}
```

**Transports Enabled**: TCP, QUIC, Tor (.onion)  
**Shroud**: Enabled (three-hop circuits **over** Tor circuits)  
**IP Visibility**: Your IP is hidden from MURMUR peers (they see the Tor exit node)  
**External Anonymity**: Strong (Tor provides IP anonymity)  

**Use When**:
- You need to hide your MURMUR usage from your ISP
- You're in a jurisdiction with censorship or surveillance
- Your threat model includes network-level correlation attacks
- You can tolerate 500ms-2s added latency per hop

**Tradeoffs**:
- ✅ **IP Privacy**: Your IP is hidden (peers see Tor exit node IP)
- ✅ **ISP Obfuscation**: ISP sees Tor usage, not MURMUR (use bridges for censorship)
- ✅ **Strong Anonymity**: Tor + Shroud = two layers of unlinkability
- ❌ **Latency**: Poor (500ms-2s circuit setup, +200-800ms per hop)
- ❌ **Reachability**: Moderate (some NATs/firewalls block Tor)
- ❌ **Dependency**: Requires Tor daemon running (port 9051)

**Plain-Language Description**:  
> "Your internet traffic goes through Tor, so no one — not your internet provider, not MURMUR peers, not even us — can see your real IP address. This is like using Tor Browser: slower, but much more private. You'll need to run a Tor daemon (we can guide you through this)."

**Failure Modes**:
- **Tor daemon not running**: Application refuses to start and shows actionable error: "Tor mode requires a Tor daemon. Install with: `apt install tor` or download from torproject.org. Expected control port: 127.0.0.1:9051"
- **Tor control port unreachable**: Same as above
- **Circuit construction timeouts**: Automatic retry with exponential backoff (3s, 6s, 12s)

---

### Mode C: I2P (Shroud over I2P)

**Configuration**:
```go
Config{
    EnableTor: false,
    EnableI2P: true,
    I2PSAMAddr: "127.0.0.1:7656", // I2P SAMv3 bridge
}
```

**Transports Enabled**: TCP, QUIC, I2P (.i2p)  
**Shroud**: Enabled (three-hop circuits **over** I2P tunnels)  
**IP Visibility**: Your IP is hidden from MURMUR peers (they see the I2P router)  
**External Anonymity**: Strong (I2P provides IP anonymity)  

**Use When**:
- You prefer I2P's "darknet" design over Tor's "mixnet"
- You want better UDP support than Tor provides
- You're already part of the I2P community
- You can tolerate 300ms-1.5s added latency per hop

**Tradeoffs**:
- ✅ **IP Privacy**: Your IP is hidden (peers see I2P router)
- ✅ **Darknet Design**: I2P is designed for hidden services (not just exit nodes)
- ✅ **UDP Support**: I2P tunnels support both TCP and UDP
- ❌ **Latency**: Moderate (300ms-1.5s tunnel setup, +100-500ms per hop)
- ❌ **Reachability**: Low (smaller I2P network than Tor)
- ❌ **Dependency**: Requires I2P router with SAMv3 enabled (port 7656)

**Plain-Language Description**:  
> "Your internet traffic goes through I2P, a privacy network designed for hidden services. This is similar to Tor but optimized for peer-to-peer applications like MURMUR. Latency is better than Tor for direct connections. You'll need to run an I2P router (we can guide you through this)."

**Failure Modes**:
- **I2P router not running**: Application refuses to start and shows actionable error: "I2P mode requires an I2P router with SAMv3. Download i2pd from i2pd.website or java-i2p from geti2p.net. Enable SAM bridge on port 7656."
- **SAMv3 not enabled**: Same as above with link to SAM documentation
- **Tunnel construction timeouts**: Automatic retry with exponential backoff (2s, 4s, 8s)

---

### Mode D: Belt-and-Suspenders (Tor + I2P)

**Configuration**:
```go
Config{
    EnableTor: true,
    EnableI2P: true,
    TorControlAddr: "127.0.0.1:9051",
    I2PSAMAddr: "127.0.0.1:7656",
}
```

**Transports Enabled**: TCP, QUIC, Tor (.onion), I2P (.i2p)  
**Shroud**: Enabled (three-hop circuits over **either** Tor or I2P)  
**IP Visibility**: Your IP is hidden from MURMUR peers  
**External Anonymity**: Maximum (two independent anonymity networks)  

**Use When**:
- You have the highest threat model (state-level adversary)
- You need redundancy (if Tor is censored, I2P may work)
- You can tolerate the setup complexity and latency
- You're willing to run both Tor and I2P daemons

**Tradeoffs**:
- ✅ **Maximum Anonymity**: Peer can reach you via Tor **or** I2P (you choose)
- ✅ **Redundancy**: If one network is blocked/censored, the other still works
- ✅ **Censorship Resistance**: Very difficult to block both Tor and I2P
- ❌ **Latency**: Worst (whichever network is slowest determines experience)
- ❌ **Complexity**: Requires both Tor and I2P daemons configured correctly
- ❌ **Resource Usage**: Higher CPU/memory/bandwidth (two anonymity networks)

**Plain-Language Description**:  
> "You run both Tor and I2P, giving you maximum privacy and redundancy. Other users can reach you through whichever network they prefer or whichever isn't censored. This is the most secure option but requires running two privacy networks. Only choose this if your safety depends on it."

**Failure Modes**:
- **Neither daemon running**: Application refuses to start with error listing both requirements
- **Only one daemon running**: Application starts but logs warning: "Mode D configured but only [Tor|I2P] is available. Falling back to Mode [B|C]."
- **Partial reachability**: Application continues but logs: "You're reachable via [Tor|I2P] but not [I2P|Tor]. Some peers may not be able to connect."

---

## Mode Selection UI/UX

### First-Time Setup (Onboarding Phase 3)

After identity creation, users are presented with the anonymity mode choice:

```
┌─────────────────────────────────────────────────────────────┐
│  Choose Your Privacy Level                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  [ ] Default (Recommended for most users)                   │
│      Your messages are private, but your internet provider  │
│      can see you're using MURMUR. Fast and easy.            │
│      Latency: Excellent | Setup: None                       │
│                                                              │
│  [ ] Tor (Strong anonymity)                                 │
│      Your IP is hidden from everyone. Slower, but much      │
│      more private. Requires Tor daemon.                     │
│      Latency: Poor | Setup: Install Tor                     │
│                                                              │
│  [ ] I2P (Expert users)                                     │
│      Similar to Tor, optimized for hidden services.         │
│      Requires I2P router with SAM bridge.                   │
│      Latency: Moderate | Setup: Install I2P                 │
│                                                              │
│  [ ] Both (Maximum paranoia)                                │
│      Run Tor and I2P for maximum privacy and redundancy.    │
│      Latency: Worst | Setup: Install both                   │
│                                                              │
│  [Learn More] [Continue]                                    │
└─────────────────────────────────────────────────────────────┘
```

**Learn More** expands to show:
- Latency comparison chart (visual bar graph: Default 20-200ms, Tor 500-2000ms, I2P 300-1500ms)
- Threat model summary (what each mode protects against)
- External resource links (torproject.org, geti2p.net)

### Runtime Mode Switching (Settings Panel)

Users can change modes at any time via Settings → Privacy → Transport Mode. Changing modes requires application restart to rebuild the libp2p host with new transports.

**Confirmation Dialog**:
```
┌─────────────────────────────────────────────────────────────┐
│  Switch to [Mode Name]?                                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Current: [Current Mode]                                     │
│  New: [New Mode]                                             │
│                                                              │
│  This will:                                                  │
│  • Restart MURMUR to apply new transport settings           │
│  • Disconnect all active connections                         │
│  • [Mode-specific requirement, e.g., "Require Tor daemon    │
│    running on port 9051"]                                    │
│                                                              │
│  Your identity, connections, and messages will be preserved. │
│                                                              │
│  [Cancel] [Restart with New Mode]                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Checklist

### Configuration Layer

- [x] **Add config flags** — `EnableTor`, `EnableI2P`, `TorControlAddr`, `I2PSAMAddr` in `pkg/config/config.go` (✓ completed §5.5)
- [x] **Transport registration** — Conditional libp2p transport construction in `pkg/networking/transport/host.go` (✓ completed §5.5)
- [ ] **Mode validation** — Validate that Mode B requires Tor daemon, Mode C requires I2P router, Mode D requires both
- [ ] **Default persistence** — Store selected mode in `~/.murmur/config.toml` so it persists across restarts

### Diagnostics (PLAN.md §5.7)

- [ ] **Startup checks** — Probe Tor control port (9051) and I2P SAM (7656) before host construction
- [ ] **Actionable errors** — Surface specific error messages with installation links if daemons unreachable
- [ ] **Health endpoint** — `/health/transports` endpoint reporting per-transport status (reachable/unreachable)

### UI Implementation

- [ ] **Onboarding screen** — Add "Choose Privacy Level" step to `pkg/onboarding/screens/mode_selection.go`
- [ ] **Settings panel** — Add "Transport Mode" section to `pkg/ui/settings.go`
- [ ] **Confirmation dialog** — Implement restart confirmation with mode-specific warnings
- [ ] **Status indicator** — Show current mode in UI (icon or text, e.g., "Tor Mode Active")

### Testing (PLAN.md §5.8)

- [ ] **Unit tests** — Mock Tor/I2P availability in mode selection tests
- [ ] **Integration tests** — Spin up ephemeral Tor + I2P instances in docker-compose and verify connectivity
- [ ] **Shroud-over-Tor test** — Build 3-hop Shroud circuit over Tor and verify anonymity
- [ ] **Shroud-over-I2P test** — Build 3-hop Shroud circuit over I2P and verify anonymity
- [ ] **Mode switching test** — Verify Mode A → Mode B → Mode C → Mode D transitions work without data loss

### Documentation

- [ ] **User guide** — `docs/USER_GUIDE_ANONYMITY_MODES.md` with step-by-step Tor/I2P installation
- [ ] **FAQ** — Address "Why is Tor slower?", "Can I use Tor bridges?", "What's better, Tor or I2P?"
- [ ] **Threat model** — Update `THREAT_MODEL.md` to clarify what each mode protects against
- [ ] **SECURITY_PRIVACY.md** — Add section on transport modes and their guarantees

---

## Success Criteria

✅ **Mode A (Default)**: Works out-of-box with zero external dependencies  
✅ **Mode B (Tor)**: Fails gracefully with actionable error if Tor daemon unreachable  
✅ **Mode C (I2P)**: Fails gracefully with actionable error if I2P router unreachable  
✅ **Mode D (Both)**: Works if both daemons available, partial fallback if only one available  
✅ **Mode Switching**: User can change modes via Settings with restart confirmation  
✅ **User Understanding**: 80%+ of users can correctly identify which mode they need based on threat model (validated via user testing)  

---

## Future Enhancements (Post-v1.0)

1. **Tor Bridge Support**: Add config option for bridge relays for censorship circumvention
2. **I2P Reseed Integration**: Integrate friend-to-friend reseed (PLAN.md §7.x)
3. **Pluggable Transports**: Support Tor's pluggable transports (obfs4, meek, snowflake)
4. **Auto-Mode Selection**: Detect ISP/country and recommend mode based on threat profile
5. **Mixed-Mode Circuits**: Some Shroud circuits over Tor, others over I2P (load balancing)
6. **Bandwidth Measurement**: Show real-time latency/bandwidth per transport in UI

---

## References

- PLAN.md §5.5 — Transport Registration (completed)
- PLAN.md §5.6 — User-Facing Modes (this document)
- PLAN.md §5.7 — Reachability Diagnostics (next task)
- THREAT_MODEL.md — Primary adversary (network-level observer)
- SECURITY_PRIVACY.md — Cryptographic primitives and guarantees
- pkg/networking/transport/onramp_tor/transport.go — Tor adapter implementation
- pkg/networking/transport/onramp_i2p/transport.go — I2P adapter implementation
