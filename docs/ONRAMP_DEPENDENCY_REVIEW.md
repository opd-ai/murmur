# Onramp Dependency Review

## Dependencies Added (PLAN.md §5.1)

### github.com/go-i2p/onramp v0.33.92
- **Purpose**: Provides Onion (Tor) and Garlic (I2P) transport adapters for anonymous networking
- **Version**: v0.33.92 (2024-11-16)
- **License**: MIT (standard permissive open-source license)
- **API Stability**: Active development, follows semantic versioning
- **Key Structs**:
  - `onramp.Onion`: Tor hidden service lifecycle management
  - `onramp.Garlic`: I2P destination lifecycle management
- **Lifecycle Methods**:
  - `NewOnion()`: Creates new Tor onion instance
  - `NewGarlic()`: Creates new I2P garlic instance
  - `Listen()`: Creates listener for incoming connections
  - `Dial()`: Dials outbound connections
  - `Close()`: Cleans up resources

### github.com/cretz/bine v0.2.0
- **Purpose**: Go Tor controller library (transitive dependency of onramp for Tor functionality)
- **Version**: v0.2.0 (2021-05-31)
- **License**: MIT
- **Stability**: Stable, widely used in Go Tor projects

## Runtime Expectations (PLAN.md §5.1)

### For Tor (Onion)
- **Requirement**: Tor daemon with control port accessible
- **Alternatives**: 
  - Embedded Tor instance via bine (included in dependency tree)
  - External system Tor daemon (tor(1)) with control port enabled
- **Default**: Attempt embedded first, fall back to system daemon

### For I2P (Garlic)
- **Requirement**: I2P router with SAMv3 enabled
- **Alternatives**:
  - External I2P router (i2pd or java-i2p) with SAM bridge enabled
  - Embedded I2P instance (if onramp provides embedded support)
- **Default**: Connect to localhost:7656 (standard SAMv3 port)

## Security Considerations

1. **Key Persistence**: Both Tor and I2P identities (hidden service keys, destination keys) must persist across restarts to maintain stable .onion/.i2p addresses
2. **Key Storage**: Keys will be stored in MURMUR's existing keystore directory (`~/.config/murmur/keys/`) with Argon2id encryption for consistency
3. **Control Port Security**: Tor control port access must be authenticated (cookie or password authentication)
4. **SAM Bridge Security**: I2P SAM bridge should bind to localhost only to prevent unauthorized access

## API Stability Assessment

- **onramp**: v0.33.x series is pre-1.0, expect potential breaking changes but semantic versioning respected
- **bine**: v0.2.0 is stable, widely deployed
- **Recommendation**: Pin exact versions in go.mod (already done), monitor for security updates

## Implementation Notes

Per PLAN.md §5.2–5.3:
- Transport adapters must implement `libp2p/go-libp2p/core/transport.Transport` interface
- Multiaddr mapping: `/onion3/<base32>:<port>` for Tor, `/garlic64/<base64>:<port>` for I2P
- Lifecycle: construct once per host, reuse for process lifetime
- Key persistence: integrate with MURMUR's existing keystore (Argon2id encryption)
