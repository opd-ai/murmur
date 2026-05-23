# Implementation Gaps — 2026-05-23

This document identifies gaps between MURMUR's stated capabilities and actual implementation, based on comprehensive code audit of 393 Go source files across 86 audited package paths.

---

## Gap 1: BIP-39 Recovery Requires User-Supplied Passphrase

- **Stated Goal**: "BIP-39 recovery" with "Argon2id keystore encryption" (README.md line 103)
- **Current State**: BIP-39 mnemonic-to-seed derivation uses **empty passphrase**: `bip39.NewSeed(mnemonic, "")` in `pkg/identity/keys/backup.go:39,67`
- **Impact**: 
  - Adversary who obtains 24-word mnemonic can directly derive keypair without additional challenge
  - No defense-in-depth: mnemonic alone grants full access (vs. mnemonic + passphrase)
  - Contradicts "Privacy is structural" principle — relies on secrecy of mnemonic alone
- **Closing the Gap**: 
  1. Add passphrase parameter to `ExportMnemonic()` and `RestoreFromMnemonic()` in `pkg/identity/keys/backup.go`
  2. Recommend 12+ character passphrase in onboarding flow (`pkg/onboarding/flow/identity.go`)
  3. Pass user-supplied passphrase to `bip39.NewSeed(mnemonic, passphrase)` at lines 39 and 67
  4. Update godoc to document passphrase requirement (NOT optional)
  5. Validate with BIP-39 test vectors that include passphrases (TREZOR test suite)

---

## Gap 2: Symmetric Key Wrapping Lacks Authentication

- **Stated Goal**: "All connections encrypted" with "structural privacy" (README.md line 12)
- **Current State**: Veiled Wave symmetric key wrapping uses **unauthenticated XOR**: `wrappedKey := xorBytes(symmetricKey, wrapKey)` in `pkg/content/waves/veiled.go:197,243`
- **Impact**: 
  - Bit-flipping attacks: adversary can flip arbitrary bits in wrapped key without detection
  - Decryption oracle: modified ciphertext can reveal plaintext through error messages
  - Violates authenticated encryption best practice (no MAC or AEAD)
- **Closing the Gap**: 
  1. Replace XOR with AES-KW (RFC 3394) or XChaCha20-Poly1305 AEAD for key wrapping
  2. Update `encryptVeiledContent()` lines 192-197 to use authenticated wrapping
  3. Update `UnwrapSymmetricKey()` lines 238-243 to verify authentication tag
  4. Bump Veiled Wave version field to indicate authenticated wrapping (backward incompatible)
  5. Validate with NIST AES-KW test vectors or libsodium crypto_secretbox test suite

---

## Gap 3: Timing Attacks on Cryptographic Comparisons

- **Stated Goal**: "Security: Key zeroing, ... ZK Resonance proofs" (README.md line 109)
- **Current State**: 
  - **Abyssal Wave authorship**: Early-exit loop compares public keys byte-by-byte (`pkg/content/waves/abyssal.go:278-282`)
  - **keysMatch utility**: Early-exit loop on first mismatch (`pkg/identity/keys/keypair.go:344-345`)
- **Impact**: 
  - Timing side-channels reveal key comparison position (measure variance across key prefixes)
  - Reduces search space for valid author keys (Abyssal Waves)
  - Compromises "structural privacy" claim if adversary can time key verification
- **Closing the Gap**: 
  1. Replace early-exit loops with `crypto/subtle.ConstantTimeCompare()`:
     - `pkg/content/waves/abyssal.go:279` → `if subtle.ConstantTimeCompare(publicKey, wave.AuthorPubkey) == 0 { return false }`
     - `pkg/identity/keys/keypair.go:343` → `return subtle.ConstantTimeCompare(a, b) == 1`
  2. Audit all other cryptographic comparisons for timing leaks (HMAC verification, signature checks)
  3. Validate with timing analysis microbenchmarks (measure variance across 10,000 comparisons with different key prefixes)

---

## Gap 4: Key Material Not Zeroed Before Garbage Collection

- **Stated Goal**: "Security: Key zeroing" (README.md line 109)
- **Current State**: 10 files implement `ZeroBytes()` with `runtime.KeepAlive()`, but **key material not zeroed** in:
  - **X25519 shared secrets**: `pkg/identity/recovery/recovery.go:66-78` (deriveSharedSecret)
  - **Veiled Wave encryption**: `pkg/content/waves/veiled.go:165,187,192` (symmetricKey, sharedSecret, wrapKey)
  - **Veiled Wave decryption**: `pkg/content/waves/veiled.go:233,238` (sharedSecret, wrapKey)
- **Impact**: 
  - Memory dumps (process inspection, swap files, crash dumps) can recover encryption keys
  - Violates "structural privacy" — keys persist in heap memory after use
  - Undermines "Key zeroing" security claim
- **Closing the Gap**: 
  1. Add `defer ZeroBytes(shared)` after `pkg/identity/recovery/recovery.go:68`
  2. Add `defer ZeroBytes(symmetricKey)` after `pkg/content/waves/veiled.go:165`
  3. Add `defer ZeroBytes(sharedSecret)` after `pkg/content/waves/veiled.go:187`
  4. Add `defer ZeroBytes(wrapKey)` after `pkg/content/waves/veiled.go:192`
  5. Add `defer ZeroBytes(sharedSecret)` after `pkg/content/waves/veiled.go:233`
  6. Add `defer ZeroBytes(wrapKey)` after `pkg/content/waves/veiled.go:238`
  7. Validate with memory forensics test: force GC, inspect heap for unzeroed keys using `pprof` heap profile

---

## Gap 5: EventBus Race Condition Under Concurrent Load

- **Stated Goal**: "Test suite: 100% pass rate ... zero race conditions" (README.md line 113)
- **Current State**: `pkg/app/eventbus.go` has **two race conditions**:
  1. **Unsynchronized channel access** (lines 417-471): `eb.inbound` channel accessed after releasing `mu.RLock()` — concurrent length checks and sends cause data races
  2. **TOCTOU on closed flag** (lines 416-424): `Emit()` checks `eb.closed`, releases lock, then sends to channel — panic if `Stop()` called between check and send
- **Impact**: 
  - Panics under concurrent Wave publishing (primary use case)
  - Lost events due to incorrect capacity calculations
  - Contradicts "zero race conditions" claim
- **Closing the Gap**: 
  1. Protect all `eb.inbound` operations with `mu` lock (lines 441, 449-450, 471)
  2. Hold `mu.RLock()` across entire `Emit()` operation, or use atomic flag for `closed`
  3. Re-run `go test -race ./pkg/app/...` to verify fix
  4. Update CI config (`.github/workflows/ci.yml`) to enforce `-race` flag on all tests

---

## Gap 6: Non-Init panic() Crashes UI/CLI

- **Stated Goal**: "Binary builds and connects to network" (README.md line 113)
- **Current State**: `mustMarshal()` function calls `panic()` during runtime Wave publishing:
  - `pkg/cli/repl.go:442` called at lines 261, 268
  - `pkg/pulsemap/game.go:1016` called at line 1000
- **Impact**: 
  - Protobuf marshal failure (malformed Wave) crashes entire CLI or game UI
  - User loses all unsaved work (no graceful degradation)
  - Violates Go best practice: panic only in init/unrecoverable states
- **Closing the Gap**: 
  1. Replace `mustMarshal()` with error-returning `marshal()` in both files
  2. Propagate error to caller: `if err := marshal(...); err != nil { return err }`
  3. Update callers (repl.go:261,268; game.go:1000) to handle marshal errors gracefully
  4. Show user-facing error message instead of crash (e.g., "Failed to publish Wave: invalid content")
  5. Validate with invalid protobuf message test (nil pointer, oversized field)

---

## Gap 7: BBolt Cursor Errors Ignored

- **Stated Goal**: "Storage: Bbolt with 7 canonical buckets, typed accessors" (README.md line 108)
- **Current State**: `pkg/store/masked_events.go` discards BBolt cursor errors with `_, _ := cursor.Seek(...)` pattern:
  - Line 226: `deleteEventParticipants()` — malformed keys processed on error
  - Line 414: `CountEventParticipants()` — incorrect counts on error
- **Impact**: 
  - Data corruption: deletion continues on cursor error
  - Incorrect UI counts: participant count wrong if cursor fails
  - Silent failure: no error propagated to caller
- **Closing the Gap**: 
  1. Check cursor errors: `for k, v := cursor.Seek(prefix); k != nil && err == nil; k, v = cursor.Next()`
  2. Return error if cursor operation fails: `if err != nil { return 0, fmt.Errorf("cursor error: %w", err) }`
  3. Update all cursor iteration patterns in `pkg/store/` (audit found 2 instances, likely more)
  4. Validate with corrupted Bbolt bucket test (force cursor error by invalidating bucket)

---

## Gap 8: Goroutine Lifecycle Not Tracked

- **Stated Goal**: "~8 persistent goroutines" with "event bus goroutine (fan-out)" (docs/TECHNICAL_IMPLEMENTATION.md lines 277,291)
- **Current State**: 
  - **GossipSub message handlers** (`pkg/networking/gossip/pubsub.go:165-183`): goroutines spawned without `sync.WaitGroup` tracking
  - **Relay acceptLoop** (`pkg/tunneling/relay/relay.go:60-77`): `listener.Accept()` blocks indefinitely even after context cancellation
  - **Cache GC** (`pkg/content/storage/cache.go:402-420`): returned `CancelFunc` may not be called, goroutine persists
- **Impact**: 
  - Goroutine leaks on shutdown: handlers/acceptLoop may still execute after Close()
  - Lost events: handlers access deallocated subsystems
  - Resource leak: GC goroutine runs indefinitely if CancelFunc not called
- **Closing the Gap**: 
  1. Add `sync.WaitGroup` to `PubSub` struct, track all message handler goroutines
  2. Add `wg.Wait()` in `PubSub.Close()` to ensure all handlers exit before returning
  3. Replace `listener.Accept()` with timeout-aware select or `SetDeadline()` in acceptLoop
  4. Document `StartGC()` godoc: "Caller must call returned CancelFunc to stop GC goroutine"
  5. Validate with goroutine count test: `runtime.NumGoroutine()` before/after shutdown

---

## Gap 9: Sentinel Errors Compared with == Instead of errors.Is

- **Stated Goal**: Go 1.25.7 with modern error handling (`go.mod` line 3)
- **Current State**: Multiple sentinel error comparisons use `==` or `!=` instead of `errors.Is()`:
  - `pkg/content/propagation/bridge_test.go:297`: `if err == ErrRateLimited`
  - `pkg/content/propagation/relay_test.go:131`: `if err != ErrInvalidWave`
  - `pkg/networking/relay/dcutr.go:188`: `if err == context.Canceled`
  - `pkg/networking/gossip/pubsub.go:232`: `if err != context.Canceled`
- **Impact**: 
  - Wrapped errors missed: `fmt.Errorf("timeout: %w", ErrRateLimited)` not detected by `==` check
  - Test false negatives: tests pass when they should fail (wrapped errors)
  - Production false negatives: context cancellation not detected if error wrapped
- **Closing the Gap**: 
  1. Replace all `err == sentinel` with `errors.Is(err, sentinel)`
  2. Replace all `err != sentinel` with `!errors.Is(err, sentinel)`
  3. Audit entire codebase for sentinel comparisons: `grep -r "== Err" pkg/`
  4. Update tests to verify wrapped error handling: wrap sentinel, ensure `errors.Is()` still detects
  5. Add `errorlint` (via golangci-lint) to flag direct sentinel comparisons and enforce `errors.Is()`/`errors.As()` usage

---

## Gap 10: HTTP Response Bodies Not Automatically Closed

- **Stated Goal**: "Resource safety" with "missing defers" audit checklist (implied by project structure)
- **Current State**: `pkg/tunneling/client/client.go:71` returns `http.Response` without deferred Close()
- **Impact**: 
  - Connection leak if caller forgets `resp.Body.Close()`
  - File descriptor exhaustion under high request volume
  - HTTP client connection pool exhaustion
- **Closing the Gap**: 
  1. **Option A (document)**: Add godoc comment: "Caller must call resp.Body.Close() after reading response"
  2. **Option B (wrap)**: Return custom `Response` type with deferred Close in finalizer (risky, GC-dependent)
  3. **Option C (helper)**: Provide `MustReadAndClose(resp) ([]byte, error)` helper that guarantees cleanup
  4. **Recommendation**: Option A + Option C — document requirement AND provide safe helper
  5. Validate with resource leak test: 1000 requests without Close(), monitor FD count

---

## Gap 11: Topic.Close() Errors Silently Discarded

- **Stated Goal**: "GossipSub configuration, topics" (README.md line 47)
- **Current State**: Topic close errors ignored in 4 locations:
  - `pkg/networking/gossip/ephemeral.go:135,277,459`: `_ = topic.Close()`
  - `pkg/networking/gossip/masked_events.go:512`: `topic.Close()` (no error check at all)
- **Impact**: 
  - Resource leak if GossipSub topic close fails (pubsub connection persists)
  - Silent failure: no visibility into cleanup errors
  - Debugging difficulty: unclear if topic closed successfully
- **Closing the Gap**: 
  1. Log all Topic.Close() errors: `if err := topic.Close(); err != nil { log.Warn("topic close failed", "topic", topicName, "err", err) }`
  2. Update cleanup functions (ephemeral.go:135,277,459; masked_events.go:512)
  3. Consider adding metrics for close failures (Prometheus counter)
  4. Validate with topic close returning error: mock `topic.Close()` to return error, verify logged

---

## Summary Table

| Gap | Severity | Component | Status | Blocking Version |
|-----|----------|-----------|--------|------------------|
| BIP-39 no passphrase | **CRITICAL** | Identity | ❌ Missing | v0.1 (security) |
| Unauthenticated key wrapping | **CRITICAL** | Anonymous Layer | ❌ Missing | v0.1 (security) |
| Timing attacks on crypto | **HIGH** | Security | ❌ Missing | v0.1 (security) |
| Key material not zeroed | **HIGH** | Security | ⚠️ Partial (10 files have ZeroBytes, 6 missing) | v0.1 |
| EventBus race condition | **CRITICAL** | Core | ❌ Missing | v0.1 (stability) |
| Non-init panic() | **HIGH** | CLI/UI | ❌ Missing | v0.1 (stability) |
| BBolt cursor errors ignored | **MEDIUM** | Storage | ❌ Missing | v0.2 |
| Goroutine lifecycle tracking | **MEDIUM** | Core | ⚠️ Partial (event bus tracked, handlers not) | v0.2 |
| Sentinel error == comparisons | **MEDIUM** | Error handling | ❌ Missing (8+ instances) | v0.2 |
| HTTP response body leaks | **MEDIUM** | Networking | ⚠️ Documented risk | v0.2 |
| Topic.Close() errors discarded | **LOW** | Networking | ❌ Missing | v0.2 |

---

## Notes

- **v0.1 Blocking Gaps**: 5 gaps block v0.1 release (BIP-39, key wrapping, timing attacks, key zeroing, EventBus races, panic)
- **Security Gaps**: 4 gaps compromise "Privacy is structural" and "Security: Key zeroing" claims
- **Test Suite Gap**: "zero race conditions" claim contradicted by EventBus races (F-CONC-1, F-CONC-2)
- **Partial Implementation**: Key zeroing implemented in 10 files but missing in 6 critical locations
