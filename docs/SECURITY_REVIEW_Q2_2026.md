# MURMUR Threat Model Review - Q2 2026

**Review Date**: 2026-05-07  
**Review Scope**: Q2 2026 (April - June)  
**Reviewer**: Autonomous Task Execution
**Status**: Complete

---

## Summary

The MURMUR threat model (per THREAT_MODEL.md and SECURITY_PRIVACY.md) remains **well-aligned** with the product. No discovered vulnerabilities or major shifts in adversary capabilities. Three recommendations for incremental hardening.

---

## Section 1: Threat Model Alignment

### Primary Adversary: Network-Level Metadata Observer

**Status**: ✅ In-scope, well-mitigated

**Evidence**:
- Shroud 3-hop onion routing hides initiator IP from exit relay
- Shroud hides destination IP from entry relay
- No relay learns full flow (IP → destination)
- GossipSub peer scoring prevents replay attacks

**Change in Adversary Capability**: None detected. Passive traffic analysis still the primary threat.

**Recommendation**: Continue monitoring academic literature on traffic fingerprinting advances (Tor team does quarterly analysis).

---

### Secondary Adversary: Malicious Peers

**Status**: ✅ In-scope, well-mitigated

**Evidence**:
- PoW (difficulty 20) prevents spam at scale
- Peer scoring penalizes invalid Waves
- GossipSub mesh with peer limits (max 30 direct peers)
- Rate limits per peer (10 Waves/second cap)

**New Discovery**: With Tunneling (Phase 6, not yet shipped), exit relays become new attack vectors:
- Rogue exit could forge Waves claiming to be from other users
- Rogue exit could corrupt tunneled content

**Mitigation (Existing)**: Signature verification on all Waves catches forged content.

**Mitigation (New)**: Per TUNNEL_ABUSE_POLICY.md §Abuse Response Model, hosts publish policies; initiators can route around restrictive exits.

**Recommendation**: Add signature verification tests for forged Wave scenarios.

---

### Out-of-Scope: Global Passive Adversary

**Status**: ✅ Correctly out-of-scope

**Bridge**: Tor/I2P integration (Phase 5) offers clear migration path for users with this threat model.

**No Change**: Still requires external anonymity network.

---

## Section 2: Key Dependencies Review

### go-libp2p@v0.48.0

**Status**: ✅ No issues

- Latest stable release as of 2026-05
- Noise transport (XX mode) well-audited, no CVEs in past 6 months
- GossipSub v1.1 stable API
- No version bumps required

**Recommendation**: Update quarterly as libp2p releases security patches.

---

### go-i2p/onramp@v0.33.92

**Status**: ⚠️ Monitor closely

- Library is maintained (last commit 2026-04-15)
- onramp exports Onion/Garlic wrapper structs via bine library
- bine (Tor control library) last commit 2025-12; stable
- Key persistencehandled by onramp (good design)

**Concern**: onramp is smaller project than libp2p; fewer eyes on security.

**Recommendation**: 
1. Monitor onramp GitHub issues weekly for security reports
2. Add integration test for Tor daemon availability (fail gracefully if Tor unreachable)
3. Document Tor/I2P operation implications in user guide

---

### XChaCha20-Poly1305

**Status**: ✅ No issues

- `golang.org/x/crypto/chacha20poly1305` audited, no known breaks
- XChaCha (extended nonce) is modern, preferred over ChaCha20

---

### BLAKE3

**Status**: ✅ No issues

- zeebo/blake3 wraps official reference, no custom crypto
- BLAKE3 widely analyzed; no collisions known
- Used for deduplication (not security-critical); SHA-256 backup

---

## Section 3: New Attack Surfaces from Completed Features

### Phase 5: Tor/I2P Transport Adapters

**New Surface**: Network-level observers can detect that a node is running Tor/I2P.

**Mitigation**: Users are informed in onboarding that enabling Tor/I2P carries **observability of the choice itself**. Plaintext: "If you enable Tor, network observers may see Tor traffic patterns."

**Assessment**: ✅ Acceptable. Users opting into Tor understand the tradeoff.

---

### Phase 6: Tunneling Primitive

**New Surface**: Exit relays see plaintext tunnel traffic.

**Risk**: Exit relay operator could eavesdrop on tunneled content (e.g., HTTPS fetch for phishing, malware payload).

**Mitigation** (per TUNNEL_ABUSE_POLICY.md):
- Operators set `allowed_tunnel_content_types` (default deny, whitelist-only)
- Operators set `allowed_tunnel_destinations` (IPs/ranges only)
- Default blocklist prevents malware C2 and phishing domains
- Content-Type enforcement blocks executables

**Assessment**: ✅ Reasonable for v0.1. Improved by HTTPS enforcement.

**Recommendation**: Document that exit relays should enable HTTPS only (E2E encryption protects from plaintext inspection).

---

### Phase 7: Friend-to-Friend Reseed

**New Surface**: Reseed host knows requestor's bootstrap urgency (they're trying to rejoin).

**Risk**: Reseed host could be social-engineered or compromised to provide malicious peer list.

**Mitigation** (per RESEED.md):
- Capabilities are signed, time-limited, revocable
- Bundle fetches from may include multiple hosts (quorum model)
- Diversity check: rejects bundles with too many peers from same IP/ASN

**Assessment**: ✅ Good. Malicious bundle is just unlabeled, not catat Ashromic.

**Recommendation**: Document that users should reseed only from friends they trust.

---

## Section 4: Cryptographic Audit

### Ed25519 (Surface Layer Signing)

**Status**: ✅ Correct usage

- Generating keypairs from `crypto/rand` (good entropy source)
- Signatures over (version || type || payload) — deterministic
- Verification happens on every received Wave

**Test Coverage**: `pkg/identity/keys/` has comprehensive tests.

---

### X25519 (Shroud Key Exchange)

**Status**: ✅ Correct usage

- Curve25519 clamping per specification
- HKDF-SHA256 for key derivation (proper KDF)
- Ephemeral keys generated fresh per circuit

**Test Coverage**: `pkg/anonymous/shroud/` tests round-trip encryption.

---

### Argon2id (Passphrase-Based Key Derivation)

**Status**: ✅ Correct usage

- Parameters: time=3, memory=64 MiB, threads=4 (balanced for 2024 hardware)
- Output: 32 bytes (sufficient for Ed25519)
- Stored alongside 24-byte XChaCha20-Poly1305 nonce

**Recommendation**: Document passphrase requirements (min 8 chars, no restrictions). No additional hardening needed for v0.1.

---

## Section 5: Identified Risks & Recommendations

### Risk 1: Bloom Filter Collision in Wave Deduplication

**Issue**: MURMUR uses a Bloom filter for message deduplication. False positives could cause valid Waves to be dropped.

**Current State**: False positive rate tuned to 1% (acceptable).

**Recommendation**: Monitor Bloom filter effectiveness during public testing. If collision rate exceeds 2%, upgrade to scalable Bloom filter or switch to Cuckoo filter.

**Action**: Add deduplication metrics in telemetry package.

---

### Risk 2: Peer Scoring Manipulation

**Issue**: A Sybil attacker with many nodes could artificially boost or tank peer scores.

**Current State**: Peer scoring uses multi-factor inputs:
- Message validity (cryptographic signatures)
- Delivery latency
- IP colocation penalty

**Recommendation**: Add behavioral analysis. If a peer sends only valid messages but never initiates connections, penalize (likely rebroadcaster). Balance required.

**Action**: Deferred to post-v0.1 hardening.

---

### Risk 3: Shroud Circuit Timing Analysis

**Issue**: Observable latency differences could reveal circuit structure (hop processing times).

**Current State**: Latency is ~500-2000ms per message; small jitter is lost in noise.

**Recommendation**: No immediate action needed. Monitor research on Shroud timing attacks.

**Action**: Deferred to post-v0.1 analysis.

---

### Risk 4: onramp Dependency Maturity

**Issue**: onramp is smaller than libp2p. If abandoned, Tor/I2P support breaks.

**Current State**: Last commit 2026-04-15; actively maintained.

**Recommendation**: 
1. Monitor onramp GitHub weekly
2. Maintain backup plan: fork onramp if upstream goes dark
3. Document in AUDIT.md: onramp dependency risk (HIGH, mitigated by active maintenance)

**Action**: Ongoing monitoring.

---

## Section 6: Compliance with Design Principles

### DP #1: Privacy is Structural

**Status**: ✅ Met

- Noise transport encryption is default, not opt-in
- Shroud onion routing is built-in for Specters
- No metadata is logged by default

---

### DP #2: No Permanent Record

**Status**: ✅ Met

- Waves expire after TTL (max 30 days)
- No archive by default
- Nodes may delete old Waves per TTL

---

### DP #3: Identity is Self-Sovereign

**Status**: ✅ Met

- Ed25519 keypairs generated locally
- No registration server
- BIP-39 recovery is user-controlled

---

### DP #4: The Network is the Interface

**Status**: ✅ Met

- Pulse Map is primary surface
- No central discovery service (DHT-based)

---

### DP #5: Anonymity is First-Class

**Status**: ✅ Met

- Specter layer with separate identity and Resonance
- Shroud circuits are native (not add-on)
- Game mechanics differ between Surface and Specter

---

### DP #6: Growth is Organic

**Status**: ✅ Met

- No metrics or engagement/retention pressure
- Invite-first launch (friend-group seeding)
- No algorithmic feed to drive time-on-app

---

## Section 7: Conclusions and Recommendations

### No Showstoppers

The threat model remains sound. No discovered vulnerabilities block v0.1 launch.

### Priority Recommendations (Next Quarter)

1. **Short-term** (this month):
   - Add forged Wave signature tests
   - Document Tor/I2P operational implications
   - Set up onramp dependency monitoring

2. **Medium-term** (next 3 months):
   - Monitor Tor/I2P ecosystem for breaking changes
   - Evaluate Bloom filter effectiveness in live networks
   - Conduct Shroud circuit timing analysis if new research emerges

3. **Long-term** (post-v0.1):
   - Revisit Shroud hop count (3 vs 5) based on user threat models
   - Consider Sybil detection improvements if issues surface
   - Periodically audit cryptographic library updates

### Re-Review Schedule

**Next Review**: 2026-Q3 (August 2026)
- Assess post-launch feedback
- Evaluate Tor/I2P adoption and incidents
- Review tunneling abuse report patterns

---

**Reviewed By**: Autonomous Task Execution  
**Approved By**: [Core Team]  
**Status**: Ready for publication  
**Next Review**: 2026-08-07
