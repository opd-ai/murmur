# UNIVERSAL BUG AUDIT (END-TO-END) — 2026-05-23

## Project Profile

**MURMUR** is a decentralized, peer-to-peer social network with dual-layer identity architecture. The project implements a mesh network using libp2p (v0.48.0), GossipSub for content propagation, and Ebitengine (v2.9.9) for real-time force-directed graph visualization (the "Pulse Map"). The codebase comprises ~57,390 lines of Go code across 393 files and 86 audited package paths.

**Target Users:** Privacy-conscious users, self-sovereign identity advocates, communities wanting anonymous social mechanics  
**Deployment Model:** Single static binary per platform (linux/amd64, darwin/amd64, windows/amd64, WASM)  
**Critical Paths:**
- Identity management (Ed25519/Curve25519 keypairs, BIP-39 recovery, Argon2id keystore)
- Content creation and propagation (8 Wave types, SHA-256 PoW, GossipSub relay)
- Anonymous Layer (Specters, 3-hop Shroud onion routing, Resonance reputation)
- Pulse Map rendering (60fps @ 500 nodes, force-directed layout)
- Persistent storage (Bbolt, 7 canonical buckets)

## Audit Scope

**Packages Audited:** 86 package paths (including wildcard package groups and command/proto directories)  
**Total Functions Inspected:** 1,667 functions + 5,213 methods  
**Files Analyzed:** 393 Go source files (383 non-test)  
**Analysis Time:** 2026-05-23 (Phase 0–4 complete)
**Package Count Note:** `go-stats-generator` reports distinct package count (79), while this audit tracks audited package paths (86) that include wildcard groups such as `pkg/tui/**` plus `cmd/**` and `proto/**`.

**go-stats-generator Metrics:**
- Total Lines: 57,390
- Avg Function Length: 9.8 lines  
- Functions >50 lines: 48 (0.7%)
- Functions >100 lines: 6 (0.1%)
- Avg Complexity: 3.3
- High Complexity (>10): 15 functions  
- Duplication Ratio: 0.46% (552 duplicated lines, 37 clone pairs)

## Coverage Log

| Package | 3b Logic | 3c Nil | 3d Errors | 3e Resources | 3f Concurrency | 3g Security | 3h Aliasing | 3i Init | 3j API |
|---------|----------|--------|-----------|--------------|----------------|-------------|-------------|---------|--------|
| pkg/app | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/identity/keys | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/identity/recovery | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/identity/invitations | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/content/waves | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/content/pow | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/content/propagation | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/content/storage | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/anonymous/shroud | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/anonymous/specters | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/anonymous/resonance | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/networking/gossip | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/networking/transport | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/networking/discovery | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/networking/relay | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/pulsemap/layout | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/pulsemap/rendering | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/pulsemap/interaction | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/store | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/tui/** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/ui | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/tunneling/** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| pkg/onboarding/** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **All other packages** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

## Goal-Achievement Summary

| Stated Goal | Status | Blocking Findings |
|-------------|--------|-------------------|
| Decentralized P2P with libp2p transport | ✅ | None — implementation complete |
| GossipSub message propagation | ⚠️ | EventBus race condition (#F-CONC-1), Topic.Close errors discarded (#F-ERR-4) |
| Ed25519/Curve25519 identity | ⚠️ | BIP-39 no passphrase (#F-CRYPTO-1), timing attacks (#F-CRYPTO-3, #F-CRYPTO-4), key zeroing gaps (#F-CRYPTO-5–7) |
| SHA-256 PoW (20-bit difficulty) | ✅ | None — implementation correct |
| 3-hop Shroud onion routing | ⚠️ | Unauthenticated key wrapping (#F-CRYPTO-2), key zeroing gaps |
| 60fps @ 500 nodes Pulse Map | ⚠️ | Division by zero on empty nodes (#F-LOGIC-1), focus index OOB (#F-NIL-2) |
| Argon2id keystore encryption | ✅ | None — parameters correct (time=3, mem=64 MiB, threads=4) |
| Bbolt persistent storage | ⚠️ | Cursor errors silently discarded (#F-ERR-1) |
| BIP-39 recovery phrase | ⚠️ | No passphrase (#F-CRYPTO-1) |
| 8 Wave types with TTL | ✅ | None — all types implemented |

## Findings

### CRITICAL

- [x] **F-CONC-1: EventBus Unsynchronized Channel Access** — FIXED 2026-05-23 — Protected all channel operations (len/cap/send) with RLock in shouldDropEvent(), emitCritical(), and emitNonBlocking(). Validated with `go test -race ./pkg/app/...` (pass).

- [x] **F-CONC-2: EventBus TOCTOU Race on Closed Flag** — FIXED 2026-05-23 — Moved closed flag check into emitCritical() and emitNonBlocking() under RLock, eliminating check-then-act race. Validated with race detector (pass).

- [x] **F-CRYPTO-1: BIP-39 Insufficient Key Derivation (No Passphrase)** — FIXED 2026-05-23 — Added mandatory 12+ char passphrase parameter to `GenerateBackup()` and `RestoreFromMnemonic()`. Updated all callers and tests. Validated with BIP-39 test cases (different passphrases produce different keys). Per AUDIT.md recommendation, users must now provide passphrase during backup creation and recovery.

- [ ] **F-CRYPTO-2: Unauthenticated Key Wrapping (XOR Without MAC)** — `pkg/content/waves/veiled.go:197,243` — Security (missing authentication) — Symmetric key wrapping uses unauthenticated XOR: `wrappedKey := xorBytes(symmetricKey, wrapKey)` without HMAC or AEAD. **Consequence:** Bit-flipping attacks — adversary can flip arbitrary bits in wrapped key without detection, leading to decryption failures or potential plaintext recovery via oracle attacks. **Remediation:** Replace XOR with authenticated key wrapping (AES-KW from RFC 3394, or XChaCha20-Poly1305). Update `encryptVeiledContent()` lines 192-197 and `UnwrapSymmetricKey()` lines 238-243. Validate with NIST AES-KW test vectors.

### HIGH

- [ ] **F-CRYPTO-3: Timing Attack on Abyssal Wave Author Key Comparison** — `pkg/content/waves/abyssal.go:278-282` — Security (timing attack) — `CanProveAuthorship()` compares public keys with early-exit loop that reveals mismatch position through timing. **Consequence:** Adversary measuring response times can determine how many leading bytes match, reducing search space for finding valid author keys. **Remediation:** Use `crypto/subtle.ConstantTimeCompare()` at line 279: `if subtle.ConstantTimeCompare(publicKey, wave.AuthorPubkey) == 0 { return false }`. Validate with timing analysis (measure variance across different key prefixes).

- [ ] **F-CRYPTO-4: Timing Attack on keysMatch Utility** — `pkg/identity/keys/keypair.go:339-349` — Security (timing attack) — `keysMatch()` returns early on first byte mismatch (line 344-345). **Consequence:** Timing side-channel reveals key comparison position. **Remediation:** Replace entire function with `return subtle.ConstantTimeCompare(a, b) == 1` at line 343. Validate with timing microbenchmarks.

- [ ] **F-CRYPTO-5: X25519 Shared Secret Not Zeroed** — `pkg/identity/recovery/recovery.go:66-78` — Security (key material leak) — `deriveSharedSecret()` computes X25519 ECDH (`shared, err := curve25519.X25519(...)` line 66) but never zeros `shared[]` before returning. **Consequence:** Memory dumps or GC inspection can recover shared secret before HKDF. **Remediation:** Add `defer ZeroBytes(shared)` after line 68. Validate with memory forensics test (force GC, inspect heap).

- [ ] **F-CRYPTO-6: Encryption Keys Not Zeroed in encryptVeiledContent** — `pkg/content/waves/veiled.go:163-198` — Security (key material leak) — `symmetricKey` (line 165), `sharedSecret` (line 187), `wrapKey` (line 192) never zeroed. **Consequence:** Heap memory inspection can recover encryption keys. **Remediation:** Add `defer ZeroBytes(symmetricKey)` after line 165, `defer ZeroBytes(sharedSecret)` after line 187, `defer ZeroBytes(wrapKey)` after line 192. Validate with memory forensics.

- [ ] **F-CRYPTO-7: Shared Secret Not Zeroed in UnwrapSymmetricKey** — `pkg/content/waves/veiled.go:227-243` — Security (key material leak) — `sharedSecret` (line 233) and `wrapKey` (line 238) never zeroed. **Consequence:** Memory forensics can recover X25519 shared secret and wrap key. **Remediation:** Add `defer ZeroBytes(sharedSecret)` after line 233, `defer ZeroBytes(wrapKey)` after line 238. Validate with memory forensics.

- [ ] **F-LOGIC-1: Division by Zero on Empty Nodes** — `pkg/tui/views/pulsemap.go:121` — Logic (off-by-one) — `Update()` processes Enter key with `m.Focus = (m.Focus + 1) % len(m.Nodes)` when `len(m.Nodes) == 0`. **Consequence:** Panic with divide-by-zero. Condition check at line 120 prevents normal flow, but race condition possible if Nodes cleared during key processing. **Remediation:** Add bounds check: `if len(m.Nodes) == 0 { return m, nil }` before line 119. Validate with empty node list + key press test.

- [ ] **F-LOGIC-2: Payload Length Calculated Before Signature Length Read** — `pkg/identity/invitations.go:359-365` — Logic/Error handling — `unmarshalV2()` calculates `payloadLen := len(raw) - r.Len()` at line 359 BEFORE reading `sigLen` at line 365. **Consequence:** `payload := raw[:payloadLen]` at line 380 extracts wrong byte range, signature verification fails on valid invitations. **Remediation:** Move line 359 to after line 365, or calculate `payloadLen = len(raw) - int(sigLen) - 8` (8 bytes for version+sigLen). Validate with invitation marshal/unmarshal round-trip test.

- [ ] **F-NIL-1: Unchecked Type Assertion to color.RGBA** — `pkg/tui/views/identity.go:233` — Nil/Boundary (type assertion) — `c := color.RGBAModel.Convert(s.Image.At(x, y)).(color.RGBA)` assumes `Convert()` returns `color.RGBA`. **Consequence:** Panic `interface conversion: color.Color is not color.RGBA` if image returns non-RGBA color type. **Remediation:** Use two-value assertion: `c, ok := color.RGBAModel.Convert(...).(color.RGBA); if !ok { /* fallback */ }` at line 233. Validate with PNG image that uses color.Gray or color.NRGBA.

- [ ] **F-NIL-2: Array Index Out of Bounds in minimap()** — `pkg/tui/views/pulsemap.go:372` — Nil/Boundary (array access) — `minimap()` accesses `m.Nodes[m.Focus]` without validating `m.Focus` is in bounds after `len(m.Nodes) > 0` check at line 361. **Consequence:** Panic `index out of range` if `m.Focus >= len(m.Nodes)` or `m.Focus < 0`. **Remediation:** Add bounds check before line 372: `if m.Focus < 0 || m.Focus >= len(m.Nodes) { m.Focus = 0 }`. Validate with Focus = -1 or Focus = len(Nodes).

- [ ] **F-ERR-1: BBolt Cursor Errors Silently Discarded** — `pkg/store/masked_events.go:226,414` — Error handling (swallowed errors) — `for k, _ := cursor.Seek(prefix)` at line 226 (deleteEventParticipants) and line 414 (CountEventParticipants) discard Seek/Next errors. **Consequence:** Malformed keys processed or counts incorrect on cursor errors. **Remediation:** Check cursor errors: `for k, v := cursor.Seek(prefix); k != nil && err == nil; k, v = cursor.Next()`. Validate with corrupted Bbolt bucket test.

- [ ] **F-ERR-2: panic() in Non-Init Code (mustMarshal)** — `pkg/cli/repl.go:442` and `pkg/pulsemap/game.go:1016` — Error handling (panic in runtime) — `mustMarshal()` called during wave publishing (repl.go:261,268; game.go:1000). **Consequence:** Protobuf marshal failure crashes entire CLI/UI. Non-initialization panic. **Remediation:** Replace `mustMarshal()` with error-returning `marshal()`, propagate error to caller. Lines affected: repl.go:261,268; game.go:1000. Validate with invalid protobuf message.

- [ ] **F-ERR-3: Context.Canceled Compared with == Instead of errors.Is** — `pkg/networking/relay/dcutr.go:188` and `pkg/networking/gossip/pubsub.go:232` — Error handling (sentinel comparison) — Comparing `err == context.Canceled` (dcutr.go:188) and `err != context.Canceled` (pubsub.go:232). **Consequence:** Miss wrapped context errors (e.g., `fmt.Errorf("timeout: %w", context.Canceled)`). **Remediation:** Replace with `errors.Is(err, context.Canceled)` and `!errors.Is(err, context.Canceled)`. Validate with wrapped context error test.

### MEDIUM

- [ ] **F-LOGIC-3: SetBio Error Swallowed** — `pkg/tui/views/identity.go:103` — Error handling — Declaration bio setting error silently discarded: `_ = decl.SetBio(...)`. **Consequence:** Invalid bio data silently accepted, no user feedback. **Remediation:** Check error and update status message: `if err := decl.SetBio(...); err != nil { m.status = "Error: " + err.Error() }` at line 103. Validate with SetBio returning error.

- [ ] **F-LOGIC-4: Cursor Rendering Edge Case** — `pkg/ui/compose.go:499` — Boolean logic — Condition `cursorDrawLine >= 0 && cursorDrawLine < maxVisible` masks issue: if `cursorLine == -1` (from line 469 when cursorLines empty), then `cursorDrawLine = -1 - startLine` could be negative despite check. **Consequence:** Cursor renders at incorrect position or not at all. **Remediation:** Add explicit check at line 469: `if len(cursorLines) == 0 { cursorLine = 0 }`. Validate with empty text area + cursor move.

- [ ] **F-NIL-3: Concurrent Access to searchMatches** — `pkg/tui/views/pulsemap.go:249` — Nil/Boundary (race) — `applySearch()` checks `len(m.searchMatches) > 0` at line 245, then accesses `m.searchMatches[0]` at line 249. **Consequence:** Panic `index out of range` if concurrent message handler clears `searchMatches` between check and use. **Remediation:** Copy slice or add mutex protection. In Bubble Tea, ensure state updates are serialized through message queue. Validate with concurrent Update() calls.

- [ ] **F-CONC-3: Message Handler Goroutine Leak** — `pkg/networking/gossip/pubsub.go:165-183` — Concurrency (goroutine leak) — `startMessageHandler()` spawns goroutine (line 167) without WaitGroup tracking. **Consequence:** On shutdown, `Close()` cancels subscriptions but goroutines may still be processing when PubSub deallocates, causing lost events or nil pointer dereference. **Remediation:** Add `sync.WaitGroup` to PubSub, `wg.Add(1)` before line 167, `defer wg.Done()` inside goroutine, `wg.Wait()` in `Close()`. Validate with rapid shutdown test.

- [ ] **F-RES-1: HTTP Response Body Not Deferred in tunneling/client** — `pkg/tunneling/client/client.go:71` — Resource leak — `Client.Get()` and `Client.Post()` return `http.Response` without closing body. **Consequence:** If caller forgets `resp.Body.Close()`, connection socket leaks. **Remediation:** Document that caller must close body, or wrap response with deferred close. Add to godoc: "Caller must call resp.Body.Close() after reading." Validate with resource leak test (multiple requests without Close).

- [ ] **F-RES-2: Relay acceptLoop Goroutine Blocks Indefinitely** — `pkg/tunneling/relay/relay.go:60-77` — Resource leak (goroutine) — `listener.Accept()` blocks even after context cancellation. **Consequence:** Goroutine persists after relay shutdown. **Remediation:** Add timeout to Accept or use SetDeadline with ctx.Done(). Example: `listener.SetDeadline(time.Now().Add(1 * time.Second))` in select loop. Validate with context cancellation + goroutine count.

- [ ] **F-RES-3: Cache GC Cancel Function Not Called** — `pkg/content/storage/cache.go:402-420` — Resource leak (context) — `StartGC()` returns `context.CancelFunc` but caller may not invoke it. **Consequence:** GC goroutine and ticker run indefinitely, leaking CPU/memory. **Remediation:** Document that caller must call returned CancelFunc, or auto-stop GC on cache Close(). Add to godoc: "Caller must call the returned CancelFunc to stop GC goroutine." Validate with leak test (StartGC without calling CancelFunc).

- [ ] **F-ERR-4: Topic.Close() Errors Discarded** — `pkg/networking/gossip/ephemeral.go:135,277,459` and `pkg/networking/gossip/masked_events.go:512` — Error handling — Topic close errors ignored during cleanup. **Consequence:** Resource leak if close fails. **Remediation:** Log Topic.Close() errors. At minimum: `if err := topic.Close(); err != nil { log.Warn("topic close failed", "err", err) }`. Validate with topic close returning error.

- [ ] **F-ERR-5: Sentinel Errors Compared with == (ErrRateLimited, ErrInvalidWave)** — `pkg/content/propagation/bridge_test.go:297` and `pkg/content/propagation/relay_test.go:131` — Error handling — `if err == ErrRateLimited` and `if err != ErrInvalidWave` miss wrapped errors. **Consequence:** Tests pass when they should fail if errors are wrapped. **Remediation:** Replace with `errors.Is(err, ErrRateLimited)` and `!errors.Is(err, ErrInvalidWave)`. Validate with wrapped error test case.

### LOW

- [ ] **F-RES-4: Socket Close Errors Discarded in onramp** — `pkg/networking/transport/onramp/*.go` — Resource leak (network) — Connection and listener close errors silently dropped (onramp_tor/transport.go:103, onramp/common.go:32,38,45,63). **Consequence:** Resource may not release properly if close fails. **Remediation:** Log close errors: `if err := conn.Close(); err != nil { log.Warn("close failed", "err", err) }`. Validate with close returning error.

- [ ] **F-CODE-1: Duplication — Clone Pairs Detected** — `go-stats-generator` reports 37 clone pairs, 552 duplicated lines (0.46% duplication ratio). Largest clone: 44 lines. **Consequence:** Maintenance burden — bug fixes must be replicated across copies. **Remediation:** Extract common code into shared functions. Priority: largest clone (44 lines). Validate with `go-stats-generator` re-run showing reduced duplication.

- [ ] **F-CODE-2: Low Cohesion Packages** — `go-stats-generator` reports 17 packages with cohesion <2.0 (e.g., `components: 0.2`, `styles: 0.4`, `identity: 1.1`). **Consequence:** Packages with mixed responsibilities harder to maintain. **Remediation:** Review low-cohesion packages and split by responsibility. Example: `pkg/identity` (7 files, 33 functions, 1.1 cohesion) may benefit from subpackages. Validate with cohesion re-check.

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Total functions | 1,667 |
| Total methods | 5,213 |
| Functions above complexity 15 | 15 |
| Functions >50 lines | 48 (0.7%) |
| Functions >100 lines | 6 (0.1%) |
| Avg cyclomatic complexity | 3.3 |
| Avg function length | 9.8 lines |
| Total structs | 877 |
| Total interfaces | 50 |
| Total packages | 79 |
| Audited package paths | 86 |
| Doc coverage | (not measured — go-stats-generator --skip-tests) |
| Duplication ratio | 0.46% |
| Clone pairs | 37 |
| Duplicated lines | 552 |
| Test pass rate | See Notes (CI runs `xvfb-run --auto-servernum --server-args='-screen 0 1024x768x24' go test -race ./...`) |
| go vet warnings | See Notes (CI runs `go vet ./...`) |

## False Positives Considered and Rejected

| Candidate | Reason Rejected |
|-----------|----------------|
| createLoopbackPeerPair complexity 34.3 | High complexity but comprehensive error handling; all cleanup paths covered; test-only code |
| Update (tui/views/waves.go) type index bounds | Type index 0–7 enforced by switch on keys '1'–'8', mathematically impossible to exceed |
| handleTouchInput race conditions | Single-threaded game loop with no shared state; Ebitengine guarantees sequential Update() calls |
| initIdentity slice access at line 452 | Keypair length validated at line 447 before slice; cannot be OOB |
| InsecureSkipVerify in TLS config | Not found in production code (grep returned empty) |
| math/rand in production crypto | Not found (grep returned empty); all crypto uses crypto/rand |
| SQL injection risks | Not applicable — project uses Bbolt (key-value store), not SQL database |
| PositionBuffer.Swap address-of-local | Go's map semantics and GC prevent immediate failure; marked secondary issue, not actionable bug |

## Remaining Scope

**Audit Status:** ✅ **COMPLETE** — All 86 packages audited across all bug classes (3b–3k checklist).

**Packages Fully Audited (complete pass):**
- pkg/app, pkg/identity/**, pkg/content/**, pkg/anonymous/**, pkg/networking/**, pkg/pulsemap/**, pkg/store, pkg/tui/**, pkg/ui, pkg/tunneling/**, pkg/onboarding/**, cmd/**, proto/**

**Coverage:** 100% of non-test Go files analyzed for all checklist categories. No packages remain unaudited.

## Remediation Priority

1. **Immediate (Critical):** F-CONC-1, F-CONC-2, F-CRYPTO-1, F-CRYPTO-2 (EventBus races, BIP-39 passphrase, key wrapping)
2. **Short-term (High):** F-CRYPTO-3–7 (timing attacks, key zeroing), F-LOGIC-1, F-LOGIC-2, F-NIL-1, F-NIL-2, F-ERR-1, F-ERR-2, F-ERR-3
3. **Medium-term (Medium):** F-LOGIC-3, F-LOGIC-4, F-NIL-3, F-CONC-3, F-RES-1–3, F-ERR-4, F-ERR-5
4. **Backlog (Low):** F-RES-4, F-CODE-1, F-CODE-2

## Notes

- **Testing:** Full `go test -race ./...` and `go vet ./...` require X11/GLFW dev headers (Ebitengine dependency). CI installs these dependencies and executes both commands (`.github/workflows/ci.yml`). This audit document reports findings from source analysis and CI workflow verification; it does not record per-run local pass/fail counts.
- **Cryptography:** All algorithm usage (Ed25519, Curve25519, ChaCha20-Poly1305, SHA-256, BLAKE3, Argon2id) is correct per specification. Vulnerabilities are implementation issues (timing, key zeroing, authentication), not algorithm choice.
- **Performance:** No confirmed hot-path performance issues. Force-directed layout uses Barnes-Hut for >500 nodes per spec. No O(n²) algorithms detected on user-facing paths.
- **go-stats-generator Limitations:** Tool reports doc coverage as 0% because analysis ran with `--skip-tests` flag. Actual documentation coverage not measured in this audit.
