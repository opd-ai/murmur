# libp2p Transport Adapter Boundary

## Overview (PLAN.md §5.2)

MURMUR uses libp2p for its networking layer. Both Tor and I2P transport adapters MUST implement the `libp2p/go-libp2p/core/transport.Transport` interface so they can compose alongside existing transports (TCP, QUIC, Noise) without special-casing in application code.

## Interface Requirements

### libp2p transport.Transport Interface

```go
type Transport interface {
    // Dial dials a remote peer via the transport-specific addressing scheme.
    // Returns a CapableConn (already upgraded with Noise encryption + yamux multiplexing).
    Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (CapableConn, error)

    // CanDial returns true if this transport can dial the given multiaddr.
    // Used by libp2p's connection manager to filter unsuitable transports before dialing.
    CanDial(addr ma.Multiaddr) bool

    // Listen creates a listener on the given multiaddr.
    // Returns a Listener that yields CapableConn on Accept().
    Listen(laddr ma.Multiaddr) (Listener, error)

    // Protocols returns protocol codes handled by this transport (for multiaddr matching).
    Protocols() []int

    // Proxy returns true if this is a proxy transport (both Tor and I2P are).
    Proxy() bool
}
```

### CapableConn Interface

Connections returned by `Dial()` and accepted from `Listen()` must implement `transport.CapableConn`, which extends `network.Conn` with:
- Stream multiplexing (yamux)
- Connection security (Noise XX handshake)
- Connection capabilities metadata

**IMPORTANT**: The underlying `net.Conn` from onramp must be wrapped in libp2p's upgrader to add Noise encryption and yamux multiplexing. onramp provides raw TCP connections; libp2p expects secure, multiplexed connections.

## Multiaddr Mapping

### Tor (Onion)

**Format**: `/onion3/<base32>:<port>`

Example: `/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:9001`

- `<base32>`: 56-character base32-encoded v3 onion address (from ed25519 public key)
- `<port>`: Hidden service port (0-65535)

**Multiaddr Protocol Code**: `444` (onion3)

### I2P (Garlic)

**Format**: `/garlic64/<base64>:<port>` or `/garlic32/<base32>:<port>`

Example: `/garlic64/JhGVjdXFqc3JRamZMWkd4K3E5VVFndDYxT1d0MnJ6dz06Om0xNXk5d2Y0bTYyMjZueHNrbGZndG14:9001`

- `<base64>`: base64-encoded I2P destination (516+ bytes, contains public key + certificate)
- `<port>`: Virtual port within I2P tunnel (0-65535)

**Multiaddr Protocol Codes**: 
- `garlic64`: `456` (proposed, not yet standardized)
- `garlic32`: `457` (shorter base32 encoding for I2P addresses)

**Note**: Multiaddr protocol codes for I2P are not yet standardized in go-multiaddr. We may need to register custom codecs or use experimental codes. See `github.com/multiformats/multicodec` for registration.

## Dial and Listen Semantics

### Dial Flow (Tor Example)

1. **Input**: `ma.Multiaddr` (e.g., `/onion3/<addr>:<port>`), `peer.ID`
2. **Parse**: Extract onion address and port from multiaddr
3. **Delegate**: Call `onion.Dial(ctx, "onion3_addr.onion:port")`
4. **Upgrade**: Wrap returned `net.Conn` in libp2p's upgrader:
   ```go
   rawConn, err := onion.Dial(ctx, targetAddr)
   secureConn, err := upgrader.Upgrade(ctx, rawConn, network.DirOutbound, peerID)
   ```
5. **Return**: `transport.CapableConn` (secure, multiplexed connection)

### Listen Flow (Tor Example)

1. **Input**: `ma.Multiaddr` (e.g., `/onion3/0.0.0.0:0` to auto-assign)
2. **Delegate**: Call `onion.Listen(ctx, port)`
3. **Translate**: Convert returned hidden service address to `/onion3/<addr>:<port>` multiaddr
4. **Return**: `transport.Listener` that:
   - Accepts raw `net.Conn` from onion listener
   - Upgrades each connection via libp2p upgrader
   - Yields `transport.CapableConn` on `Accept()`

### Latency Considerations

**Tor**: High latency (~500ms-2s for circuit construction, ~200-800ms per hop for data)
- Initial dial may take several seconds (Tor circuit construction + libp2p Noise handshake + yamux setup)
- Subsequent dials via existing circuits are faster (reuse Tor stream multiplexing)

**I2P**: Moderate-to-high latency (~300ms-1.5s for tunnel construction, ~100-500ms per hop)
- Initial tunnel construction can take up to 60 seconds (I2P exploratory tunnels + garlic routing setup)
- Once tunnels are established, latency is comparable to Tor

**Implication**: Dial timeouts must be generous (30-60s for initial dial, 10-15s for subsequent). libp2p's default dial timeout (15s) may be too aggressive for initial Tor/I2P connections.

## Transport Coexistence

### Multi-Transport Strategy

MURMUR will register multiple transports simultaneously:
- **Clearnet**: TCP + QUIC (existing, via libp2p defaults)
- **Tor**: onion3 transport (via onramp Onion adapter)
- **I2P**: garlic transport (via onramp Garlic adapter)

libp2p's connection manager will:
1. Filter candidate transports via `CanDial(addr ma.Multiaddr)`
2. Attempt dials in priority order (configured per privacy mode)
3. Fall back to next transport on failure

### Multiaddr Selection Logic

Per PLAN.md §5.5:
- **Default (Shroud over clearnet)**: Prefer TCP/QUIC for surface traffic; Shroud adds onion routing on top
- **Tor Mode**: All outbound dials use onion3 transport; inbound via Tor hidden service
- **I2P Mode**: All outbound dials use garlic transport; inbound via I2P destination
- **Dual Mode**: Register both onion3 and garlic; libp2p tries both (first to connect wins)

**Implementation**: Use libp2p's `DialRanker` to prioritize transports based on user's privacy mode.

## Security Boundary

### What onramp Provides
- Raw TCP connection to Tor SOCKS proxy or I2P SAM bridge
- Tor circuit construction (via control port commands)
- I2P tunnel management (via SAM protocol)

### What onramp Does NOT Provide
- End-to-end encryption (payload is cleartext within Tor/I2P network)
- Stream multiplexing (separate Tor streams share circuit, but no libp2p-level multiplexing)
- Peer authentication (no libp2p peer ID verification at Tor/I2P layer)

### What libp2p Upgrader Adds
- **Noise XX encryption**: End-to-end encryption between libp2p peers (independent of Tor/I2P)
- **yamux multiplexing**: Multiple logical streams over single Tor/I2P connection
- **Peer ID authentication**: Cryptographically verifies remote peer's identity via Noise handshake

**Defense-in-Depth**: Shroud onion routing (MURMUR layer) + Tor onion routing (Tor layer) + Noise encryption (libp2p layer) provides three layers of protection. Compromise of any single layer does not expose user identity or payload content.

## Key Persistence Strategy

Per PLAN.md §5.3:
- **Tor**: Hidden service private key (ed25519) must persist to maintain stable .onion address
- **I2P**: Destination private key (elgamal/ecdsa) must persist to maintain stable .i2p address

**Storage Location**: `~/.config/murmur/keys/transport/`
- `tor_onion.key`: Encrypted with Argon2id (same passphrase as Surface identity keystore)
- `i2p_destination.key`: Encrypted with Argon2id

**Lifecycle**:
1. On first transport creation, generate new key and persist
2. On subsequent startups, load existing key
3. On key rotation (future feature), create new key and store continuity mapping

## Testing Strategy

### Unit Tests
- Mock onramp `Onion` and `Garlic` interfaces
- Test multiaddr parsing (valid `/onion3/<addr>:<port>`, invalid formats)
- Test `CanDial` logic (accept onion3/garlic, reject others)
- Test key persistence (write, read, encryption round-trip)

### Integration Tests
- Use ephemeral Tor instance (via `github.com/cretz/bine` embedded mode) or mock
- Use ephemeral I2P router (if onramp supports embedded) or mock
- Test full dial/listen cycle:
  1. Start transport with embedded Tor/I2P
  2. Listen on auto-assigned onion/garlic address
  3. Dial localhost via onion/garlic multiaddr
  4. Verify connection succeeds and Noise handshake completes
  5. Send test data, verify receipt
  6. Close connection cleanly

### Interop Tests (PLAN.md §5.8)
- Cross-transport: clearnet peer dials Tor peer, Tor peer dials I2P peer, etc.
- Shroud over Tor: 3-hop Shroud circuit where all hops use onion3 transport
- Fallback: attempt onion3 dial (fails because Tor daemon not running), fall back to clearnet
- Race detector enabled for all tests

## Implementation Checklist

- [x] Add go-i2p/onramp dependency (§5.1)
- [x] Define transport adapter boundary (§5.2 — this document)
- [ ] Implement Tor transport adapter (§5.3)
  - [ ] `NewTransport(ctx, keystore)` constructor
  - [ ] `Dial(ctx, raddr, peerID)` with upgrader
  - [ ] `Listen(laddr)` with hidden service setup
  - [ ] `CanDial(addr)` onion3 detection
  - [ ] Key persistence (tor_onion.key)
  - [ ] Unit tests (mock onramp.Onion)
- [ ] Implement I2P transport adapter (§5.4)
  - [ ] `NewTransport(ctx, keystore)` constructor
  - [ ] `Dial(ctx, raddr, peerID)` with upgrader
  - [ ] `Listen(laddr)` with destination setup
  - [ ] `CanDial(addr)` garlic detection
  - [ ] Key persistence (i2p_destination.key)
  - [ ] Unit tests (mock onramp.Garlic)
- [ ] Register transports in libp2p host builder (§5.5)
  - [ ] Conditional construction based on config flags (tor_enabled, i2p_enabled)
  - [ ] Transport priority configuration per privacy mode
- [ ] User-facing mode selection UI (§5.6)
- [ ] Reachability diagnostics (§5.7)
- [ ] Integration tests with ephemeral Tor/I2P (§5.8)
- [ ] Documentation (TRANSPORT_ANONYMITY.md) (§5.9)
