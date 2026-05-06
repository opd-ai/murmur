# BIP-39 Recovery UX Audit

**Date:** 2026-05-06  
**Purpose:** Per PLAN.md §3.1 — Audit current BIP-39 recovery UX  
**Implementation:** `pkg/identity/keys/backup.go`, `pkg/onboarding/screens/recovery_screen.go`

---

## Time-to-Recover on New Device

### Current Flow

1. **User launches MURMUR on new device** → Onboarding welcome screen
2. **Select "Recover Identity"** → Recovery method selection screen
3. **Select "Recovery Phrase (24 words)"** → Mnemonic entry screen
4. **Type/paste 24-word phrase** → Real-time input, no word suggestions, no autocomplete
5. **Press Enter** → BIP-39 validation + keypair restoration
6. **Success** → Identity recovered, proceed to mode selection

### Time Estimate

| Step | Estimated Time | Notes |
|------|----------------|-------|
| Method selection | 5–10 seconds | Clear two-button UI |
| Mnemonic entry | 60–180 seconds | Typing 24 words manually; 90 seconds if pasting from secure storage |
| Validation | <1 second | BIP-39 validation is instant |
| **Total** | **~90–200 seconds** | Faster if user pastes; slower if typing from paper backup |

### Observations

✅ **Fast for paste workflow** — Users with digital backups (password manager, encrypted note) can paste and recover in <2 minutes  
⚠️ **Slow for manual entry** — Typing 24 words from paper takes 2–3 minutes with potential typos  
❌ **No autocomplete** — BIP-39 wordlist has 2048 words; autocomplete would significantly speed manual entry  
❌ **No word validation feedback** — User doesn't know if individual words are valid until they press Enter  

---

## Failure Modes When Seed Is Partially Remembered

### Scenario: User Lost Physical Backup, Remembers ~20 of 24 Words

**Current Behavior:**  
- User enters the 20 words they remember plus 4 guesses
- System returns: **"Invalid recovery phrase. Please check and try again."**
- No indication of which words are incorrect
- No partial recovery option

**Workarounds:**  
❌ **Brute force remaining words** — Infeasible: 2048^4 = ~17.6 trillion combinations  
❌ **Manual word-by-word validation** — Not exposed in UI  
❌ **Partial seed recovery tools** — Not integrated; user must use external tools (e.g., BTCRecover)  

### Assessment

🔴 **CRITICAL LIMITATION** — Users with partial memory of their seed phrase have effectively lost their identity. The 24-word mnemonic is an all-or-nothing recovery mechanism.

**Impact:** In a messaging+games network where identity = social connections, losing identity is catastrophic. Unlike Bitcoin wallets (where loss = financial damage but recovery may be possible via exchanges), MURMUR identity loss means:
- Loss of all connections (Surface + Specter)
- Loss of Resonance history (reputation reset)
- Loss of game participation history
- Loss of Phantom Council memberships

This makes the current BIP-39-only recovery strategy **unacceptable for product-market fit**.

---

## What Is Preserved vs. Lost on Recovery?

### Preserved (Derived from Keypair)

✅ **Surface Layer identity** — Ed25519 public key (SHA-256 → sigil, deterministic pseudonym generation)  
✅ **Specter identity** — Curve25519 keypair derived from same seed (independent of Surface key per spec)  
✅ **Sigils** — Deterministic from public key hash; identical after recovery  
✅ **Cryptographic identity** — Signatures verify correctly; connections recognize recovered identity  

### Lost (Local State)

❌ **Connections** — No automatic reconnection; user must re-establish connections manually (peers may have purged the offline identity from their contact list)  
❌ **Wave history** — Waves are ephemeral (7–30 day TTL); old Waves likely expired  
❌ **Resonance scores** — Stored locally; not recoverable from network (Resonance is client-computed from observed activity)  
❌ **Game history** — Participation records, trophies, territory control are local state  
❌ **Phantom Council memberships** — Council rosters are network-observable but user must re-join (councils may have expelled inactive members)  
❌ **Shroud circuits** — Ephemeral; must be reconstructed  
❌ **Pulse Map topology state** — Local cache; must be rebuilt from DHT  

### Assessment

⚠️ **Identity recovery ≠ full account recovery** — User regains cryptographic identity but loses most social context. This is **acceptable for cryptographic correctness** but **poor UX for social applications**.

**Comparison to competitors:**
- **Signal:** Cloud backup (encrypted); preserves message history
- **Matrix/Element:** Server-side state; preserves rooms and history
- **MURMUR:** No server; loses all non-derivable state

---

## Gaps & Recommendations

### Gap 1: No Multi-Device Support (PLAN.md §3.2)

**Current:** One identity = one mnemonic = one device at a time. User recovering on a new device cannot seamlessly continue using the old device.

**Impact:** Users who upgrade phones, use multiple devices (phone + laptop), or temporarily lose a device face:
- Full identity recovery flow every time they switch devices
- Loss of local state (see "Lost" section above)
- Confusion about which device is "active"

**Recommendation:** Implement multi-device identity per PLAN.md §3.2:
- One logical identity → multiple device keys
- Device-specific Ed25519 keys linked to master identity via signed declarations
- Old device can be revoked without requiring access to that device
- Connections see device changes as continuity (via device key rotation declarations)

---

### Gap 2: No Social Recovery (PLAN.md §3.3)

**Current:** BIP-39 mnemonic is the only recovery path. Losing it = permanent identity loss.

**Impact:** High-anxiety backup burden on users; single point of failure.

**Recommendation:** Implement social recovery per PLAN.md §3.3:
- **Shamir Secret Sharing:** User designates N trusted contacts; M-of-N can co-sign recovery
- **Separate for Surface + Specter:** Surface recovery and Specter recovery use independent secret shares (no cross-layer deanonymization)
- **Specter recovery must not reveal Specter identity to recovery participants:**
  - Use ZK proofs or blinded secret sharing
  - Recovery participants prove they hold valid shares without learning whose Specter they're recovering
  - Implementation challenge: deferred to post-v1.0

**Example Flow:**  
1. User designates 5 trusted contacts (3-of-5 threshold)
2. Each contact receives encrypted secret share (only decryptable by their identity key)
3. User loses mnemonic
4. User requests recovery from any 3 of the 5 contacts
5. Contacts provide their shares; user's identity key is reconstructed
6. User sets new mnemonic, continues with recovered identity

---

### Gap 3: No Partial Seed Recovery Assistance

**Current:** All-or-nothing validation; no guided recovery for partially remembered seeds.

**Recommendation:**  
- **Word-by-word validation:** Highlight invalid BIP-39 words as user types
- **Autocomplete from BIP-39 wordlist:** Reduce typos and speed entry
- **Checksum hint:** BIP-39 last word is partially a checksum — if user has 23 correct words, the system can suggest the 24th (only 8 valid possibilities)
- **Brute-force assistant (offline only):** If user has 22–23 words, offer to search remaining combinations (2048^2 = 4M combinations is feasible on modern hardware in minutes)

---

### Gap 4: No Key Rotation (PLAN.md §3.4)

**Current:** Keypairs never rotate. If a key is compromised, user must generate entirely new identity (lose all connections).

**Impact:** No defense against key compromise; no proactive security hygiene.

**Recommendation:** Implement identity continuity across key rotation per PLAN.md §3.4:
- User generates new Ed25519 keypair
- Old keypair signs a **Continuity Declaration**: `old_pubkey → new_pubkey` with timestamp and signature
- Continuity Declaration is broadcast as special Identity Wave on Surface Layer
- Contacts observe declaration, automatically update their local peer store: `old_pubkey` now resolves to `new_pubkey`
- User can continue conversations without "is this really you?" friction
- Old key remains valid for short grace period (7 days) to allow declaration propagation

**Threat model:** Prevents impersonation after key rotation; adversary who compromises old key cannot forge continuity declaration (requires old key's signature, which the user generates before compromise is detected).

---

### Gap 5: Key File Recovery Is Incomplete

**Current Implementation:**  
- `backup.go` has `ImportKeyPairFromFile()` function
- `recovery_screen.go` has Key File method UI
- **But:** File picker integration is pending (line 399: "File picker integration pending")

**Impact:** Key file recovery is not actually usable. Users cannot select a key file from disk.

**Recommendation:**  
- Integrate platform-specific file picker (desktop: OS dialog, mobile: document picker)
- Display key file fingerprint (first 8 hex chars of SHA-256 hash) before passphrase entry
- Provide example key file format in documentation

---

### Gap 6: No Recovery Testing

**Current:** No way for user to test their recovery phrase works **before** they need it.

**Impact:** Users may discover their backup is invalid/incomplete only during actual recovery (device loss, theft, etc.) — too late.

**Recommendation:**  
- Add "Test Recovery Phrase" option in Settings
- User enters their mnemonic; system verifies it matches current identity
- Displays: "✓ Recovery phrase valid. This phrase will recover your current identity."
- **Critical UX:** Frame as "backup verification", not "enter your seed to prove you remember it" (latter triggers security anxiety)

---

## Summary: Current State Assessment

| Criterion | Status | Rating |
|-----------|--------|--------|
| **Time-to-recover** | 90–200 seconds (acceptable) | ⚠️ OK |
| **Failure handling (partial seed)** | No assistance; all-or-nothing | 🔴 CRITICAL GAP |
| **Preserved state** | Only cryptographic identity | ⚠️ UX LIMITATION |
| **Lost state** | Connections, Resonance, game history | ⚠️ UX LIMITATION |
| **Multi-device** | Not supported | 🔴 CRITICAL GAP |
| **Social recovery** | Not implemented | 🔴 CRITICAL GAP |
| **Key rotation** | Not supported | ⚠️ SECURITY GAP |
| **Recovery testing** | Not available | ⚠️ UX GAP |

---

## Recommendations Priority

**Must-do before v1.0 (per PLAN.md §3):**
1. **Multi-device identity (§3.2)** — Essential for realistic device usage patterns
2. **Social recovery (§3.3)** — Reduces single-point-of-failure anxiety; competitive with Signal/WhatsApp
3. **Key rotation (§3.4)** — Enables proactive security; prevents forced identity loss on compromise

**Strongly recommended before v1.0:**
4. **Recovery testing** — Builds user confidence in backup process
5. **Autocomplete + word validation** — Reduces recovery friction and typo risk

**Can defer to v1.1:**
6. **Partial seed brute-force assistant** — Niche use case; complex implementation
7. **Key file picker integration** — Alternative to mnemonic; lower priority

---

## Conclusion

The current BIP-39 recovery implementation is **cryptographically sound but UX-incomplete**. For a messaging+games network where identity = social graph, the following gaps are **blockers to product-market fit**:

1. No multi-device support → Forces users into single-device usage pattern (unrealistic)
2. No social recovery → High backup anxiety; competitors (Signal, Element) offer this
3. No partial seed assistance → Users with 95% correct mnemonic lose identity permanently

**Action:** Proceed to PLAN.md Phase 3 tasks (§3.2, §3.3, §3.4) immediately after completing current phase. These are prerequisite to any public release.
