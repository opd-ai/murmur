# RED TEAM AUDIT — 2026-05-08

## Threat Model

### Deployment Model
MURMUR is a peer-to-peer desktop/mobile application. Every participant device is
simultaneously a client and a server node in a libp2p mesh. There is no central
server. Nodes expose:
- A libp2p swarm port (TCP/UDP, random by default) reachable from the internet
- An optional HTTP health/metrics endpoint (`pkg/networking/health`)
- An optional TCP tunnel relay port (`pkg/tunneling/relay`)
- Optional Tor/I2P overlay transports

### Trust Boundary Map
| Layer | Boundary | Notes |
|-------|----------|-------|
| Internet | Outermost | libp2p swarm, health HTTP, tunnel relay all bind to 0.0.0.0 |
| Peer mesh | Semi-trusted | Noise XX encrypts transport, but gossip payload validity is the only access control |
| Local process | Trusted | Keystore on disk, bbolt DB, in-memory key material |
| User passphrase | Trusted | Protects keystore via Argon2id + XChaCha20-Poly1305 |

### Attacker Profile
- **Class 1 — Network Peer:** Controls one or more libp2p nodes; can send arbitrary gossip messages
- **Class 2 — Passive Observer:** Can observe libp2p swarm traffic; cannot decrypt Noise XX
- **Class 3 — Relay Operator:** Runs a Shroud relay; sees one hop of onion traffic
- **Class 4 — Physical / Memory Access:** Can read process memory or keystore files on disk

### Sensitive Assets
- Ed25519 Surface private key (identity, signing)
- Curve25519 Specter private key (anonymity layer, all Shroud circuits)
- Argon2id-derived keystore encryption key (transient, in memory during unlock)
- Shamir shares (recovery contacts hold encrypted shares of master key seed)
- Abyssal Wave nonces (stored locally; link nonces to one-time keys)
- bbolt database (waves, threads, Resonance, peer table)

---

## Attack Surface Map

| Entry Point | Trust Level | Input Type | Downstream Sinks | Risk |
|-------------|-------------|------------|------------------|------|
| GossipSub (`/murmur/waves/1`) | Untrusted (Internet) | Protobuf bytes | PoW validator, timestamp check, Ed25519 verify, bbolt store | MEDIUM |
| GossipSub (`/murmur/identity/1`) | Untrusted (Internet) | Protobuf bytes | Timestamp check, Ed25519 verify, bbolt store | MEDIUM |
| GossipSub (`/murmur/shroud/1`) | Untrusted (Internet) | Protobuf bytes | Relay advertisement parser, Beacon relay table | MEDIUM |
| Shroud circuit construction | Untrusted (Peer) | Curve25519 public keys from `Beacon.AddRelay` | `deriveHopKey`, XChaCha20-Poly1305 | HIGH |
| Tunnel relay TCP port | Untrusted (Internet) | Raw TCP bytes | HTTP request forwarding, tunnel registry map | HIGH |
| Health HTTP endpoint (`/health`, `/metrics`) | Untrusted (Internet) | HTTP GET | JSON response with peer ID, topology; Prometheus metrics | MEDIUM |
| Gossip message deduplicator | Untrusted (Peer) | `[]byte` message IDs | Bloom filter + in-memory map | LOW |
| Veiled Wave content | Untrusted (Peer) | Encrypted ciphertext | XChaCha20-Poly1305 decryption, content display | LOW |
| Abyssal Wave | Untrusted (Peer) | Ed25519 one-time pubkey | Signature verify, wave store | HIGH |
| Content filter patterns | Trusted (Local User) | Wildcard strings | `matchWildcard` string scan over Wave content | LOW |
| Keystore file | Trusted (Local Disk) | Encrypted bytes | Argon2id derive, XChaCha20-Poly1305 decrypt, key import | LOW |
| Shamir share enrollment | Trusted (Recovery Contact) | Encrypted share bytes | ECDH, HKDF, XChaCha20-Poly1305, Shamir combine | LOW |

---

## Attack Scenarios

### HIGH

- [x] **Unauthenticated Tunnel UNREGISTER — Remote Tunnel Teardown**
  — `pkg/tunneling/relay/relay.go:155-169`
  — **Vector:** Any TCP connection to the relay port. The relay's `handleConnection`
  discriminates on the first byte: if it is not `0x7F` (FrameMagic) it falls into the
  plain-text path where the first line is inspected. If the line starts with
  `UNREGISTER `, `handleUnregister` is called immediately with no authentication check.
  — **Payload:**
  ```
  telnet <relay-host> <relay-port>
  UNREGISTER alice-abcdefghijkl
  ```
  The attacker sends `UNREGISTER <known-tunnel-id>\n`. The function finds the session in
  the `tunnels` map, closes the operator's TCP connection, and removes the entry — with
  zero proof of ownership.
  — **Impact:** Any attacker who can guess or enumerate a tunnel ID (IDs are deterministic:
  `BLAKE3(pubkey||name)[0:8]` base32-encoded, so the name component is often guessable)
  can immediately terminate any active tunnel session. A denial-of-service against all
  tunnels on a relay is achievable by iterating the 13-character base32 space for common
  name prefixes. Operators' localhost services become inaccessible.
  — **Exploitability:** Trivially exploitable with a single TCP connection. No authentication
  required. No rate-limiting on the UNREGISTER path.
  — **Remediation:** Remove the plain-text `UNREGISTER` path entirely. Use `FrameTypeTeardown`
  cells with a valid signed payload (operator must sign `TEARDOWN || tunnelID || timestamp`
  using the same keypair registered at `REGISTER` time). The relay validates the signature
  against the stored `OperatorPubkey` before removing the tunnel.
  **Verification test:** `UNREGISTER <valid-id>` from an unauthenticated TCP connection should
  receive `HTTP/1.1 401 Unauthorized\r\n\r\n` and not remove the tunnel.

- [x] **`Beacon.SecretKey()` Exposes Long-Term Shroud Private Key**
  — `pkg/anonymous/shroud/circuit.go:307-310`
  — **Vector:** Any code path that calls `beacon.SecretKey()` — the method returns the 32-byte
  Curve25519 private key used for every circuit key derivation. In
  `performKeyAgreements → deriveHopKey`, all hop shared secrets are derived from this single
  long-term key: `X25519(beacon.secretKey, relay.PublicKey)`. There is no ephemeral key
  contribution on the initiator side.
  — **Impact:** If an attacker obtains the Beacon private key (via memory dump, core file,
  bug report, log line) they can retroactively compute `X25519(secretKey, relay.PublicKey)`
  for every relay in the network and recover all past and future circuit hop keys. This
  breaks Shroud's anonymity guarantee for every circuit ever built by the compromised node.
  The `SecretKey()` method is a public API that actively invites callers to handle key
  material. Combined with the non-ephemeral key exchange, this is a forward-secrecy
  violation.
  — **Specific scenario:** An attacker compromises relay telemetry, which logs beacon public
  key material. By calling `SecretKey()` during a diagnostic mode, the private key appears
  in a support log. The attacker reconstructs all circuit keys using the logged value.
  — **Remediation:**
    1. Remove `SecretKey()` from the public API. Key material must never leave the `Beacon`
       struct.
    2. Replace the static DH with an ephemeral key exchange per circuit: generate a fresh
       Curve25519 ephemeral key per `BuildCircuit` call; discard after `performKeyAgreements`
       completes. This provides forward secrecy.
    3. Use HKDF-SHA-256 (spec-mandated) rather than BLAKE3 for hop key derivation (see
       separate finding below).
  — **Status (2026-05-08):** Completed items (1) and (2). `Beacon.SecretKey()` removed from
    `pkg/anonymous/shroud/circuit.go`; `BuildCircuit` now generates a fresh ephemeral
    Curve25519 initiator keypair per circuit and zeroes the ephemeral secret after use. Added
    `TestBuildCircuitUsesEphemeralInitiatorKey` in
    `pkg/anonymous/shroud/circuit_test.go` to verify shared keys are not reused across
    circuits built over the same relay set.

- [ ] **Abyssal Wave Specter-Key Compromise Retroactively Deanonymises All Abyssal Waves**
  — `pkg/content/waves/abyssal.go:deriveAbyssalKeypairFromNonce` (approx. line 62-72)
  — **Vector:** `deriveAbyssalKeypairFromNonce` computes:
  ```
  seed = SHA-256(specter_private_key || abyssal_nonce)
  one_time_key = Ed25519.NewKeyFromSeed(seed)
  ```
  The `AbyssalStore` stores `waveID → nonce` locally. Each published Abyssal Wave carries
  the `one_time_key.PublicKey` in the `AuthorPubkey` field.
  — **Payload (retroactive deanonymisation):**
    1. Attacker obtains `specter_private_key` (e.g., via keystore file compromise or
       passphrase brute-force).
    2. For each Abyssal Wave in the network, the attacker iterates over all possible 32-byte
       nonces from the `AbyssalStore` records (or brute-forces 32-byte nonce space).
    3. For each candidate nonce `n`, compute
       `SHA-256(specter_priv || n)` → `Ed25519.NewKeyFromSeed(...)` → check if pubkey
       matches `wave.AuthorPubkey`.
    4. All Abyssal Waves authored by this Specter are linked and their one-time author keys
       are exposed.
  — **Impact:** An adversary with access to the Specter private key (the 32-byte Curve25519
  key, but used here as an Ed25519 seed input) can link all Abyssal Waves to the Specter,
  completely defeating the "deep anonymity" guarantee. The nonce is 32 bytes; if the local
  `AbyssalStore` is also available (same device compromise), the attacker has all nonces
  directly.
  — **Remediation:**
    1. Derive the Abyssal key from a sub-key that is NOT the Specter private key directly.
       Instead, derive an Abyssal master key: `abyssal_master = HKDF-SHA-256(specter_priv,
       salt="abyssal-v1")`. Use `abyssal_master || nonce` as the SHA-256 input.
    2. This does not eliminate compromise of `abyssal_master` but it ensures Specter key
       compromise alone does not immediately yield all Abyssal keys.
    3. For maximum security, derive per-session abyssal master keys: rotate the abyssal
       sub-key on each Specter session initialisation.

- [ ] **Shroud Hop Key Uses BLAKE3 Instead of Spec-Required HKDF-SHA-256**
  — `pkg/anonymous/shroud/circuit.go:595-607`
  — **Vector:**
  ```go
  func (b *Beacon) deriveHopKey(relay *RelayInfo, hopIndex int) []byte {
      var shared [32]byte
      curve25519.ScalarMult(&shared, &b.secretKey, &relay.PublicKey)
      h := blake3.New()
      h.Write(shared[:])
      h.Write([]byte("shroud-hop-key"))
      h.Write([]byte{byte(hopIndex)})
      key := h.Sum(nil)
      ...
  }
  ```
  The `TECHNICAL_IMPLEMENTATION.md` key derivation table mandates
  `HKDF-SHA-256` for "Key derivation from DH shared secrets". `murmur-whisper` in
  `whisper.go` correctly uses `hkdf.New(sha256.New, shared, nil, []byte("murmur-whisper"))`.
  The Shroud circuit uses BLAKE3 instead.
  — **Impact:** While BLAKE3 is cryptographically sound today, this is a spec deviation with
  three consequences: (a) interoperability is broken if a future node implements the spec
  correctly and expects HKDF-SHA-256-derived keys; (b) the security proof for the protocol
  relies on HKDF's extraction-then-expand structure, which BLAKE3-based construction does
  not provide; (c) the nil HKDF salt (also absent here) is acceptable in HKDF because the
  DH output provides high entropy, but raw BLAKE3 usage without proper domain separation
  is less auditable.
  — **Remediation:** Replace `blake3.New()` in `deriveHopKey` with:
  ```go
  kdf := hkdf.New(sha256.New, shared[:], nil, []byte("murmur-shroud-hop-v1"))
  io.ReadFull(kdf, key[:])
  ```

---

### MEDIUM

- [ ] **Gossip Timestamp Validator Rejects Messages Older Than 300s — Breaks Wave TTL**
  — `pkg/networking/gossip/handlers.go:validateTimestamp`
  — **Vector:**
  ```go
  if drift > MaxTimestampDrift { // drift > 300s → reject
      return ErrInvalidTimestamp
  }
  ```
  Waves have a TTL of up to 30 days (`MaxTTL = 30 * 24 * time.Hour`). However, any Wave
  with a `created_at` timestamp more than 300 seconds in the past is rejected at the gossip
  validation layer regardless of remaining TTL.
  — **Impact (DoS / network partition):**
    1. A node that is offline for >5 minutes and reconnects cannot receive any Waves created
       during its absence — they are all stale-rejected.
    2. An attacker can exploit this to partition the network: by delaying gossip propagation
       by >300 seconds (e.g., via eclipse attack), the attacker ensures victims never see
       legitimate Waves.
    3. Nodes that have clock skew >300s (possible on embedded/IoT deployments) cannot
       participate at all.
  — **Remediation:** Separate the timestamp freshness check from the TTL-based expiry check:
    - On ingestion: allow messages within `min(MaxTimestampDrift, wave.TTLSeconds - 
      elapsed)` of now.
    - On store-and-forward relay: check `(now - created_at) < wave.TTLSeconds` instead of
      applying the 300s drift limit.
    This allows long-lived Waves to be relayed to late-joining peers while still protecting
    against messages with timestamps far in the future.

- [ ] **Health and Metrics Endpoints Bind on All Interfaces Without Authentication**
  — `pkg/networking/health/health.go:79` — `Addr: fmt.Sprintf(":%d", port)`
  — **Vector:** When `EnableHealthEndpoint: true` is set in config (intended for bootstrap
  nodes), the HTTP server listens on `0.0.0.0:<port>`. No authentication middleware is
  applied to either `/health` or `/metrics`.
  — **Payload:** `curl http://<node-ip>:8080/health`  → reveals libp2p PeerID, peer count,
  topic subscriptions, uptime. `curl http://<node-ip>:8080/metrics` → full Prometheus
  exposition with Shroud circuit build durations, gossip peer scores, message rates.
  — **Impact:**
    1. **Privacy:** The libp2p PeerID is a persistent pseudonymous identifier. Combined with
       the node's IP address, this allows network-level deanonymisation of nodes claiming to
       be privacy-conscious.
    2. **Network topology:** An attacker can enumerate bootstrap nodes by probing port 8080
       across the internet, then use `/metrics` to map peer connectivity graphs.
    3. **Operational intelligence:** Shroud circuit build timing, gossip peer scores, and
       message rates reveal operational patterns useful for traffic analysis.
  — **Remediation:**
    1. Bind to `127.0.0.1` by default; expose `0.0.0.0` only when explicitly configured with
       `HealthBindAll: true`.
    2. Add `Authorization: Bearer <token>` middleware using a randomly generated token
       printed to stdout on startup for bootstrap operators.
    3. Omit PeerID from the response body, or add an explicit privacy warning in docs.

- [ ] **Biased Relay Selection in Shroud Circuit Construction**
  — `pkg/anonymous/shroud/circuit.go:518-527`
  — **Vector:**
  ```go
  func pickRandomUnusedIndex(max int, used map[int]bool) int {
      var randomBytes [1]byte
      rand.Read(randomBytes[:])
      idx := int(randomBytes[0]) % max   // <-- modulo bias
      for used[idx] {
          idx = (idx + 1) % max          // <-- linear probe
      }
      return idx
  }
  ```
  With `max = 3` (exactly 3 eligible relays, the minimum):
  - Byte 0 → idx 0 (always), Byte 1 → idx 1, Byte 2 → idx 2, ...
  - Bytes 3-255 wrap: idx cycles 0→1→2→0... with unequal probability
  - First relay selected from byte range 0x00..0x55 (86 values = 85 + 1), second from
    0x56..0xAA (85), third from 0xAB..0xFF (85). Near-equal for 3 relays, but for
    larger relay sets the bias grows.
  - The linear probe means: when `used[0]=true` and `randomBytes[0]=0`, the probe always
    picks index 1 as the fallback — never 2. This creates correlation between selections.
  — **Impact:** An adversary operating multiple Shroud relay nodes can use the statistical
  bias to predict which relay positions they are more likely to occupy, increasing the
  probability of controlling two of the three hops. With two controlled hops, timing
  correlation attacks become feasible to deanonymise circuit users.
  — **Remediation:** Use `crypto/rand.Int` with `big.Int` modular arithmetic to eliminate
  modulo bias, or use Fisher-Yates shuffle on the eligible slice with proper uniform
  sampling:
  ```go
  import "math/rand"
  // Use rand.New(rand.NewSource(cryptoSeed)) or math/rand/v2 with crypto seeding
  ```
  Alternatively, use `rand.Shuffle` with a crypto-seeded source on the eligible slice and
  take the first `CircuitLength` elements.

- [ ] **`ZeroBytes` Loop May Be Optimised Away by the Go Compiler**
  — `pkg/identity/keys/keypair.go:216-222`
  — **Vector:**
  ```go
  func ZeroBytes(b []byte) {
      for i := range b {
          b[i] = 0
      }
  }
  ```
  The Go compiler's escape analysis and dead-store elimination can, in principle, remove
  writes to memory that is not subsequently read within the same function. The `defer
  ZeroBytes(data)` pattern in keystore operations defers the call; but once the function
  returns and the slice is no longer reachable, the compiler is not required to execute
  the stores. In Go 1.22+, this is currently safe because the compiler does not eliminate
  the writes when the slice is passed by reference — but this is an implementation detail,
  not a language guarantee.
  — **Impact:** On a hypothetical future Go version that adds stronger dead-store
  elimination, private key bytes could remain in memory after `ZeroBytes` is called, making
  them recoverable from process heap dumps or core files.
  — **Remediation:** Use `crypto/internal/subtle.InexactOverlap` (internal, not exported)
  or the pattern `runtime.KeepAlive(b)` after the loop, or use `golang.org/x/crypto`'s
  `subtle` helpers. As a practical interim, wrap the zeroing in a call that cannot be
  inlined:
  ```go
  //go:noinline
  func ZeroBytes(b []byte) {
      for i := range b { b[i] = 0 }
      runtime.KeepAlive(b)
  }
  ```

---

### LOW

- [ ] **Missing HKDF Salt in Veiled Wave, Whisper, and Recovery Key Derivation**
  — `pkg/content/waves/veiled.go:205`, `pkg/anonymous/shroud/whisper.go:130`,
    `pkg/identity/recovery/recovery.go:71`
  — **Vector:** All three HKDF invocations use `nil` as the salt parameter:
  ```go
  kdf := hkdf.New(sha256.New, dhSharedSecret, nil, []byte("murmur-veil-wrap-v1"))
  ```
  The HKDF specification (RFC 5869) recommends a random salt when the input keying
  material may be weak. When the IKM is a full 32-byte Curve25519 DH output, the nil
  salt is acceptable because the DH output already provides sufficient entropy for
  extraction. However, it creates a minor auditability concern.
  — **Impact:** Low in practice for Curve25519 DH inputs with 32 bytes of entropy.
  Theoretical: if a low-order point slip occurred and DH output was weak, HKDF with a
  proper random salt would provide additional entropy extraction. (Low-order point check
  exists in `DeriveSharedSecret` for whisper but not uniformly for veiled wave wrapping.)
  — **Remediation:** Supply a 16-byte random salt per key derivation instance, or use the
  spec-recommended HKDF salt = `"murmur-domain-v1"` as a fixed domain separator.

- [ ] **Legacy Keystore `.bak` File Retention**
  — `pkg/identity/keys/keystore.go:renameLegacyFile`
  — **Vector:** `MigrateLegacyKeystore` renames the legacy combined keystore to `<path>.bak`
  on successful migration. The `.bak` file contains the encrypted combined Surface + Specter
  keypairs. If a directory listing is shared (e.g., bug report, backup) the file is
  included. If the passphrase is reused and the attacker has the `.bak` file, they can
  decrypt it.
  — **Impact:** Residual attack surface; the `.bak` file provides a second path to key
  material beyond the separated keystore files. In combination with passphrase compromise,
  the attacker can recover both Surface and Specter keys from the legacy format.
  — **Remediation:** After a configurable hold period (e.g., 24 hours with a startup message
  to the user), securely overwrite the `.bak` file bytes before deletion. At minimum, print
  a user-visible warning: "Migration complete. Delete `surface.keystore.bak` when you have
  verified your new keystores load correctly."

- [ ] **`AbyssalStore` Nonces Stored in Unencrypted In-Memory Map**
  — `pkg/content/waves/abyssal.go:AbyssalStore`
  — **Vector:** `AbyssalStore.records` maps wave IDs (as `string(waveID)`) to their 32-byte
  derivation nonces in a plain Go map. If the process is suspended (SIGSTOP) and its memory
  is dumped, all nonces are recoverable in plaintext. The nonces, combined with the Specter
  private key, allow retroactive computation of all one-time keys.
  — **Impact:** Complements the Abyssal Key Derivation finding (HIGH). Even without Specter
  key compromise, a memory dump leaks all locally-stored Abyssal nonces, which when paired
  with any future Specter key exposure provides full deanonymisation.
  — **Remediation:** Do not persist nonces in-memory beyond the immediately needed scope.
  If authorship provability is a feature, store nonces in the encrypted bbolt database
  (`BucketWaves`) using the keystore-encrypted symmetric key rather than in a plain Go map.

- [ ] **Wave Content Filter Wildcard Pattern DoS — Pathological Input**
  — `pkg/content/filtering/filter.go:matchWildcard`
  — **Vector:** The content filter applies user-configured wildcard patterns against Wave
  content bytes cast to lowercase string. A pattern like `"a*b*c*d*e*f*g*h*i*j*k*l*m"` with
  a 2048-byte Wave body containing `"aaaaaa...aaa"` (no match) triggers
  `matchWildcardParts` which iterates all pattern parts over the text using
  `strings.Index`. The linear scan is safe for a single pattern, but with O(k) patterns
  each applied to O(n) Wave content, and k patterns potentially in the thousands, this
  could cause >500ms filter latency per Wave.
  — **Impact:** A local attacker (or a confused user with many filter patterns) could cause
  the Ebitengine rendering thread to stall while filtering a large burst of Waves. Since
  filtering is called from the display path, this degrades the UI. In the worst case with
  a network-level attacker flooding Waves, the combined filter latency could stall the
  Ebitengine loop below 60fps.
  — **Remediation:** Apply a `MaxKeywordPatterns = 100` limit in `MuteKeyword`. For patterns
  with many `*` splits, cache the compiled `strings.Contains` chain rather than re-splitting
  on every call.

---

## Exploitation Chains

### Chain 1: Tunnel Tunnel-ID Enumeration → Service Disruption
1. Attacker connects to relay TCP port.
2. Attacker sends a series of `UNREGISTER <name>-<13-char-hash>\n` with common tunnel names
   (`alice`, `bob`, `dev`, `test`, etc.) plus all possible 13-character base32 suffixes
   (feasible for short names, since the hash is only 8 bytes / 64-bit space but the name
   is typically user-chosen and guessable).
3. For each valid tunnel ID, the relay closes the operator connection.
4. All operator tunnels are disrupted in one scan.
5. **Required attacker capability:** TCP access to relay port — default is publicly exposed.

### Chain 2: Beacon Key Exfiltration → Full Shroud Deanonymisation
1. Attacker identifies a target MURMUR node (via health endpoint PeerID or DHT peer discovery).
2. Attacker causes the target node to produce a diagnostic output that includes
   `beacon.SecretKey()` (e.g., via a future admin RPC, a debug log line, or a future
   error report).
3. Armed with the 32-byte Beacon secret key and the set of relay public keys (visible via
   Shroud relay advertisements on `/murmur/shroud/1`), the attacker computes all hop shared
   secrets for every circuit ever built by the target.
4. The attacker can now decrypt all Shroud circuit traffic retroactively, deanonymising the
   target's Specter identity.

### Chain 3: Health Endpoint Fingerprinting → Selective Eclipse Attack
1. Attacker scans the internet for nodes with port 8080 open and a `/health` response
   matching MURMUR's JSON format.
2. Attacker correlates PeerIDs from `/health` responses with peer IDs seen in the DHT.
3. Attacker identifies high-connectivity bootstrap nodes (high `connections` count).
4. Attacker mounts an eclipse attack against bootstrap nodes using Sybil peers, isolating
   new nodes joining the network.
5. Isolated new nodes cannot receive valid relay advertisements, degrading Shroud anonymity
   for the entire network.

---

## False Positives Considered and Rejected

| Candidate Attack | Reason Rejected |
|-----------------|-----------------|
| SQL injection | No SQL database in use; all storage is via bbolt (key-value, no query language) |
| Template injection | No text/template or html/template usage found in any message-handling code path |
| Path traversal via Wave content | Wave content bytes never used as filesystem path input |
| JWT `alg:none` attack | No JWT in use; all signatures are Ed25519 via protobuf |
| Zip Slip | No archive extraction anywhere in the codebase |
| CORS exploitation | No browser-facing HTTP endpoints with CORS headers found |
| Argon2id brute-force | Parameters (time=3, mem=64MiB, threads=4) meet OWASP recommendations; acceptable |
| GossipSub resource exhaustion via large messages | GossipSub has configurable max message size; protobuf parsing is bounded by the 2048-byte Wave content limit |
| Replay via stale messages | BLAKE3 message ID deduplication in `Deduplicator` plus timestamp check prevents replay of recent messages |
| `crypto/rand` weakness | All security-critical randomness uses `crypto/rand.Read`; no `math/rand` found in sensitive paths |
| ECDH low-order point attack on Curve25519 | `DeriveSharedSecret` in `keypair.go` explicitly checks for all-zero shared secret and returns an error |
| Shamir share forgery | Share reconstruction requires `threshold` shares; a single compromised contact cannot reconstruct the master key |
