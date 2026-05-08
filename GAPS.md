# Security Defense Gaps — 2026-05-08

## Unauthenticated Tunnel UNREGISTER

- **Attack Scenario**: An attacker opens a TCP connection to the tunnel relay's public port
  and sends `UNREGISTER <tunnel-id>\n`. The relay's `handleUnregister` function performs no
  signature check, no session binding, and no rate limit. Any party that knows or can guess
  a tunnel ID can close any active operator session. Since tunnel IDs are deterministic
  (`BLAKE3(pubkey || name)[0:8]` base32), common names such as `dev`, `alice`, or `test`
  can be brute-forced at rates limited only by TCP connection setup time.
- **Current Defenses**: None. The code path is reachable from the first byte of any
  unauthenticated TCP connection to the relay port.
- **Missing Defense**: Operator-owned teardown authentication. The relay stores the
  operator's Ed25519 public key at registration time (`TunnelRegisterCell.OperatorPubkey`).
  The `UNREGISTER` path must require a signed teardown cell where the operator proves
  possession of the corresponding private key.
- **Remediation**:
  1. Remove the plain-text `UNREGISTER` path from `relay.handleConnection`.
  2. Add `FrameTypeTeardown` processing in `handleFramedOperator`: decode a
     `TunnelTeardownCell`, verify its `Signature` field (Ed25519 over
     `tunnelID || timestamp`) against the stored `OperatorPubkey`, check timestamp within
     ±60 seconds, then remove the tunnel.
  3. In `initiator.Stop()`, replace `UNREGISTER <id>\n` with a framed
     `FrameTypeTeardown` cell signed by the operator private key.
  4. Add per-source-IP rate limiting on registration attempts (max 5 REGISTER/UNREGISTER
     per minute per IP) using `golang.org/x/time/rate`.
- **Priority**: HIGH — Publicly exploitable with a single TCP connection; no credentials
  required; directly disrupts active tunnels.

---

## Beacon Long-Term Private Key Exposure and Missing Forward Secrecy

- **Attack Scenario**: `pkg/anonymous/shroud.Beacon.SecretKey()` is a public method that
  returns the raw 32-byte Curve25519 private key. All Shroud circuit hop keys are derived
  from `X25519(beacon.secretKey, relay.PublicKey)` — a static DH with no ephemeral
  component. If the key is logged, serialised in diagnostics, or passed to an untrusted
  component via this method, an attacker can retroactively decrypt all past circuit traffic
  by recomputing every hop shared secret.
- **Current Defenses**: The key is stored in an unexported struct field. Key is only
  accessed via `SecretKey()` or indirectly in `deriveHopKey`. No logging of the key is
  currently present.
- **Missing Defense**:
  1. Ephemeral key exchange per circuit (forward secrecy).
  2. Removal of the `SecretKey()` public accessor.
  3. HKDF-SHA-256 derivation per spec instead of raw BLAKE3.
- **Remediation**:
  1. Delete `func (b *Beacon) SecretKey() [32]byte`. Internals that need it already have
     access via the unexported `b.secretKey` field.
  2. In `BuildCircuit`, generate a fresh ephemeral Curve25519 key pair per circuit. Use the
     ephemeral private key in `deriveHopKey` and discard it (zero and release) once all hop
     keys are established. Send the ephemeral public key to each relay hop during circuit
     setup so relays can compute the same shared secret from their side.
  3. Replace the BLAKE3 KDF in `deriveHopKey` with:
     ```go
     kdf := hkdf.New(sha256.New, shared[:], nil, []byte("murmur-shroud-hop-v1"))
     io.ReadFull(kdf, key[:32])
     ```
- **Priority**: HIGH — Structural anonymity guarantee depends on forward secrecy; public
  key exposure is a latent but permanently exploitable weakness.

---

## Abyssal Wave One-Time Key Derivation Tied to Specter Private Key

- **Attack Scenario**: `DeriveAbyssalKeyPair` computes
  `seed = SHA-256(specter_private_key || nonce)`. The `AbyssalStore` holds `waveID →
  nonce` in a plain Go map. An attacker who obtains the Specter Curve25519 private key
  (via keystore compromise, memory dump, or passphrase attack) can iterate all nonces from
  the `AbyssalStore` (or brute-force the 32-byte nonce space for each published wave) to
  reconstruct every one-time key pair. This retroactively attributes all Abyssal Waves to
  the same Specter identity.
- **Current Defenses**: The Specter private key is protected by Argon2id + XChaCha20-Poly1305
  in the keystore. The `AbyssalStore` is in-memory only.
- **Missing Defense**: Abyssal keys must be derived from a secret that is independent of
  and not derivable from the Specter private key alone. Additionally, nonces must be
  protected from in-memory exfiltration.
- **Remediation**:
  1. Introduce an `AbyssalMasterKey` derived via `HKDF-SHA-256(specter_priv, salt,
     "murmur-abyssal-master-v1")` and store it separately. Derive one-time keys from
     `HKDF(abyssal_master || session_epoch || nonce)`.
  2. Session-epoch rotate the abyssal master key on each application launch so that
     compromise of one epoch does not expose other epochs.
  3. Persist nonces in the encrypted bbolt `BucketWaves` bucket (encrypted with the
     keystore symmetric key) rather than a plain in-memory map.
  4. Consider not storing nonces at all if the "prove authorship" feature is not required
     by users; Abyssal Waves already carry a one-time public key that is unlinkable without
     the nonce.
- **Priority**: HIGH — Specter private key compromise (the primary threat against the
  Anonymous Layer) immediately yields full Abyssal attribution.

---

## Gossip Timestamp Validation Breaks Long-TTL Wave Delivery

- **Attack Scenario**: `validateTimestamp` in `pkg/networking/gossip/handlers.go` rejects
  any message with `created_at` older than 300 seconds. Waves are designed to live for up
  to 30 days (`MaxTTL`). A node that reconnects after a 6-minute outage cannot receive any
  Wave created during its absence — all are rejected as "timestamp out of acceptable range".
  An adversary mounting an eclipse or delay attack against a node can ensure the victim
  misses all Waves published during the blackout. Waves cannot be resynchronised across
  time gaps, permanently partitioning old content from late-joining nodes.
- **Current Defenses**: GossipSub's peer scoring detects message withheld
  (`FirstMessageDeliveriesWeight` and `InvalidMessageDeliveriesWeight` are both configured).
  However, this penalises the peer delivering the old message — not the attacker who
  withheld it.
- **Missing Defense**: TTL-aware timestamp validation that distinguishes between "message
  timestamp is in the future" (reject) and "message timestamp is in the past but within
  TTL" (relay).
- **Remediation**:
  1. Tighten the future-timestamp check: reject messages with `created_at > now + 30s`.
  2. Loosen the past-timestamp check: accept messages where
     `now - created_at < wave.TTLSeconds`. Remove the blanket 300-second cap for
     past timestamps.
  3. For the Wave sync protocol (`/murmur/wave-sync/1`), explicitly support requesting Waves
     by hash or time range to allow late-joining nodes to catch up within TTL.
- **Priority**: MEDIUM — Affects availability and data integrity for legitimate nodes;
  exploitable for targeted content suppression.

---

## Unauthenticated Health and Metrics Endpoint Exposing Node Identity

- **Attack Scenario**: When `EnableHealthEndpoint: true`, the HTTP server binds on
  `0.0.0.0:<port>`. `GET /health` returns the node's libp2p PeerID (a persistent
  pseudonymous identifier), peer count, subscribed topics, and uptime in JSON.
  `GET /metrics` returns full Prometheus metrics including Shroud circuit build durations,
  gossip peer scores per topic, and message delivery rates. An adversary scanning the
  internet for MURMUR nodes (port 8080) can build a persistent map of node PeerIDs to IP
  addresses, cross-referencing with DHT peer discovery to construct network topology.
- **Current Defenses**: `EnableHealthEndpoint` defaults to `false`. Documentation notes the
  endpoint is for "bootstrap node operators".
- **Missing Defense**:
  1. Localhost-only bind by default.
  2. Authentication token for external access.
  3. PeerID omission from unauthenticated responses.
- **Remediation**:
  1. Change `Addr: fmt.Sprintf(":%d", port)` to `Addr: fmt.Sprintf("127.0.0.1:%d", port)`.
     Add a separate `HealthBindPublic bool` config flag for operators who explicitly need
     external access.
  2. Generate a random bearer token at startup when `HealthBindPublic: true`. Print it to
     stdout: `health endpoint token: <token>`. Apply `net/http` middleware to reject
     requests without `Authorization: Bearer <token>`.
  3. In the `/health` response, replace `peer_id` with `peer_id_hash` (first 8 bytes of
     BLAKE3 over PeerID) to prevent direct PeerID correlation while still allowing health
     monitoring.
- **Priority**: MEDIUM — Privacy violation for privacy-focused users; enables network-level
  deanonymisation and topology mapping at scale.

---

## Biased Relay Selection Weakens Shroud Anonymity Set

- **Attack Scenario**: `pickRandomUnusedIndex` in
  `pkg/anonymous/shroud/circuit.go` reads a single byte from `crypto/rand` and applies
  modulo over the relay count. With 3 relays, byte values `0x00-0x54` (85 values) map to
  index 0, `0x55-0xA9` (85 values) to index 1, and `0xAA-0xFF` (86 values) to index 2 —
  effectively uniform for exactly 3 relays. However, the linear-probe collision resolution
  (`idx = (idx+1) % max`) creates non-uniform fallback: when the first pick collides,
  the fallback always increments rather than re-sampling uniformly. An adversary operating
  three malicious relays and observing the distribution of circuits can statistically
  confirm nodes that exhibit the biased pattern.
- **Current Defenses**: `crypto/rand` is used for the initial byte; the selection is
  slightly biased but not trivially predictable.
- **Missing Defense**: Uniform sampling with rejection or Fisher-Yates shuffle.
- **Remediation**:
  Shuffle the eligible relay slice using a crypto-seeded Fisher-Yates algorithm and take
  the first `CircuitLength` elements:
  ```go
  for i := len(eligible) - 1; i > 0; i-- {
      jBytes := make([]byte, 8)
      rand.Read(jBytes)
      j := int(binary.BigEndian.Uint64(jBytes) % uint64(i+1))
      eligible[i], eligible[j] = eligible[j], eligible[i]
  }
  selected := eligible[:CircuitLength]
  ```
  This eliminates modulo bias and linear-probe correlation simultaneously.
- **Priority**: MEDIUM — Weakens statistical anonymity; exploitable by adversaries
  controlling a significant fraction of the relay pool.

---

## `ZeroBytes` Not Guaranteed to Execute Due to Compiler Optimisation

- **Attack Scenario**: `ZeroBytes` in `pkg/identity/keys/keypair.go` is a plain loop over
  a byte slice. The Go specification does not guarantee that dead stores (writes to memory
  that is not subsequently read) are preserved. If the compiler determines the slice is not
  read after the zeroing call (e.g., after `defer ZeroBytes(data)` and the data is
  returned or the local variable goes out of scope), future compiler versions may elide the
  writes as an optimisation. A process memory dump taken after a key-using operation could
  still contain plaintext key material.
- **Current Defenses**: Go 1.22 does not currently elide such writes in practice. The
  `defer ZeroBytes(data)` pattern is applied consistently in keystore operations.
- **Missing Defense**: Compiler-barrier or `runtime.KeepAlive` to prevent dead-store
  elimination; alternatively, platform-specific secure memory zeroing (`memset_s` on Unix).
- **Remediation**:
  1. Add `runtime.KeepAlive(b)` at the end of `ZeroBytes` to prevent the compiler from
     eliding the preceding writes.
  2. Add `//go:noinline` to prevent the function from being inlined (inlining can sometimes
     re-enable dead-store elimination across the call boundary).
  3. Long-term, use `golang.org/x/sys/unix.Mlock` on key material allocations to prevent
     them from being swapped to disk.
- **Priority**: MEDIUM — Speculative risk in current Go versions; real risk if the Go
  toolchain improves dead-store elimination in future releases.

---

## Missing HKDF Salt in Three Key Derivation Sites

- **Attack Scenario**: `hkdf.New(sha256.New, ikm, nil, info)` is called with `nil` salt at
  `pkg/content/waves/veiled.go:205`, `pkg/anonymous/shroud/whisper.go:130`, and
  `pkg/identity/recovery/recovery.go:71`. Per RFC 5869, a `nil` salt is replaced by a
  zero-valued byte string of `HashLen` (32 bytes for SHA-256). If the input keying material
  `ikm` has less entropy than assumed (e.g., due to a Curve25519 implementation bug or
  small-subgroup attack that was not caught by the low-order check), the HKDF extraction
  step provides less entropy expansion than a properly salted invocation would.
- **Current Defenses**: Curve25519 low-order point check in `DeriveSharedSecret` mitigates
  the most obvious entropy-collapse attack. The DH output is 32 bytes of high entropy in
  the normal case.
- **Missing Defense**: Random or fixed-domain salt to improve extraction robustness.
- **Remediation**:
  Add a fixed domain-separator salt to all three HKDF call sites:
  ```go
  salt := []byte("murmur-hkdf-salt-v1")
  kdf := hkdf.New(sha256.New, dhSharedSecret, salt, []byte("murmur-veil-wrap-v1"))
  ```
  This is a best-practice improvement that strengthens the security proof without
  behavioral change for well-formed inputs.
- **Priority**: LOW — No current exploitation path; hardening measure for defence-in-depth.

---

## Legacy Keystore Backup File Retains Sensitive Key Material

- **Attack Scenario**: After `MigrateLegacyKeystore` runs in
  `pkg/identity/keys/keystore.go`, the original combined keystore is renamed to
  `<path>.bak` rather than securely deleted. An attacker who gains read access to the data
  directory (e.g., via a directory traversal in a future component, a misconfigured backup
  tool, or a cloud-sync client that automatically backs up `~/.murmur/`) obtains a copy of
  the encrypted combined keystore. If the user's passphrase is later compromised, this
  backup provides a second path to recover both Surface and Specter private keys from the
  legacy format.
- **Current Defenses**: The `.bak` file is Argon2id + XChaCha20-Poly1305 encrypted;
  decryption requires the passphrase. The migration only runs once.
- **Missing Defense**: Secure deletion with data overwrite before removing the `.bak` file;
  user notification about the `.bak` file's existence.
- **Remediation**:
  1. After successful migration and on the next application startup, overwrite the `.bak`
     file with random bytes before deleting it:
     ```go
     rand.Read(overwriteBuffer)
     os.WriteFile(bakPath, overwriteBuffer, 0600)
     os.Remove(bakPath)
     ```
  2. On first startup post-migration, display a one-time warning:
     `"Migration complete. Backup file at <path>.bak will be securely deleted in 24h."`
  3. Set a cleanup flag in the bbolt config bucket to trigger deletion on next launch.
- **Priority**: LOW — Requires passphrase compromise in addition to file access; defence-in-depth.
