# Social Recovery Design (Shamir Secret Sharing)

**Task:** PLAN.md §3.3  
**Status:** Design Complete — Ready for Implementation  
**Date:** 2026-05-06  
**Dependencies:** `pkg/identity/keys/`, `proto/identity.proto`, Shamir Secret Sharing library

---

## Executive Summary

This document specifies MURMUR's **social recovery system** using Shamir Secret Sharing (SSS), allowing users to recover their identity through trusted contacts when the BIP-39 mnemonic is lost. 

**Key Properties:**
- **M-of-N threshold recovery:** User designates N trusted contacts; any M contacts can cooperatively reconstruct the Master Key
- **No single point of trust:** No individual contact can recover the identity alone
- **Surface + Specter independence:** Separate SSS schemes for Surface and Specter identities (no cross-layer linkability)
- **Specter recovery preserves anonymity:** Recovery participants do NOT learn whose Specter they're recovering (ZK proof integration deferred to v1.1)

**Design Philosophy:** Reduce single-point-of-failure anxiety around BIP-39 mnemonic backup while maintaining MURMUR's "no trusted third party" principle. Social trust (N trusted contacts) replaces paper backup as the recovery backstop.

---

## Problem Statement

### Current Limitation (per BIP39_RECOVERY_AUDIT.md §Gap 2)

**Existing behavior:**
- BIP-39 mnemonic (24 words) is the **only** recovery path
- If user loses mnemonic → permanent identity loss (connections, Resonance, game history all unrecoverable)
- Single point of failure creates high-anxiety backup burden

**Why this is unacceptable:**
1. **Anxiety:** Users who fear losing their mnemonic avoid committing to the platform
2. **Competitive disadvantage:** Signal (cloud backup), WhatsApp (Google Drive backup), Matrix (server-side state) all offer multi-factor recovery
3. **Realistic threat:** House fires, floods, theft, forgetfulness — physical paper is vulnerable
4. **Social network context:** In a messaging+games network, identity = social graph. Losing identity is catastrophic (unlike Bitcoin where exchange backups may exist).

---

## Design Goals

1. **Threshold Recovery (M-of-N)**  
   User designates N trusted contacts (3 ≤ N ≤ 10); any M contacts (2 ≤ M ≤ N) can cooperatively recover the Master Key. Standard configurations: 3-of-5, 2-of-3, 3-of-7.

2. **Zero Single Point of Trust**  
   No individual contact can recover the identity. Contacts must cooperate (at least M contacts) to reconstruct the secret.

3. **Separate Surface and Specter Recovery**  
   Surface Master Key and Specter Master Key use independent SSS schemes. Recovering Surface does NOT reveal Specter; recovering Specter does NOT reveal Surface.

4. **Contact Privacy**  
   Contacts do NOT learn the Master Key directly. They hold encrypted secret shares that only the user can reconstruct. Future enhancement (v1.1+): ZK proofs prevent contacts from learning whose identity they're recovering.

5. **Graceful Degradation**  
   If user loses M-1 contacts, recovery fails gracefully (no partial information leak). User can re-share with new contacts while retaining old shares (re-enrollment).

6. **No Server Dependency**  
   Secret shares are distributed peer-to-peer. No centralized key escrow service. Aligns with MURMUR's "no servers" philosophy.

---

## Shamir Secret Sharing Primer

**Algorithm (brief):**
- User has a secret S (e.g., 32-byte Master Private Key)
- Choose threshold M and total shares N (M ≤ N)
- Generate polynomial of degree M-1: `P(x) = S + a₁x + a₂x² + ... + aₘ₋₁x^(M-1)` where coefficients a₁...aₘ₋₁ are random
- Evaluate P(x) at N distinct x-coordinates (x=1, x=2, ..., x=N) → N shares: (1, P(1)), (2, P(2)), ..., (N, P(N))
- **Reconstruction:** Given any M shares, use Lagrange interpolation to reconstruct P(x) and extract S = P(0)
- **Security:** Knowing M-1 shares reveals nothing about S (information-theoretically secure)

**Properties relevant to MURMUR:**
- Threshold flexibility: 2-of-3 (easy, less secure), 3-of-5 (balanced), 5-of-7 (paranoid)
- Share size ≈ secret size (32-byte secret → 33-byte shares with 1-byte metadata)
- No computational assumptions (unlike multi-signature schemes which rely on EC hardness)

**Library Choice:** `github.com/hashicorp/vault/shamir` (well-audited, used in Hashicorp Vault, supports arbitrary thresholds)

---

## Architecture

### Recovery Coordinator Roles

1. **User (Identity Owner)**  
   - Possesses Master Key (either from fresh generation or from BIP-39 mnemonic recovery)
   - Initiates recovery share distribution
   - Initiates recovery reconstruction

2. **Recovery Contacts (Trustees)**  
   - Designated by User
   - Each holds one encrypted secret share
   - Cooperate (at least M contacts) to enable User recovery
   - Do NOT hold User's Master Key directly

3. **Recovery Share Storage**  
   - Each contact's share is encrypted with contact's public key (X25519 key exchange → symmetric encryption)
   - Stored locally by contact in Bbolt `recovery_shares` bucket
   - User also stores a copy of all shares (encrypted) as backup

---

## Protocol Messages

### 1. Recovery Share Enrollment

**Purpose:** User distributes encrypted secret shares to N trusted contacts.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message RecoveryShareEnrollment {
  bytes master_public_key = 1;        // User's identity root (for informational purposes; NOT derived from share)
  bytes recipient_public_key = 2;     // Contact's X25519 public key
  bytes encrypted_share = 3;          // XChaCha20-Poly1305 encrypted SSS share
  bytes nonce = 4;                    // 24-byte nonce for XChaCha20
  uint32 share_index = 5;             // Share index (1 to N)
  uint32 threshold = 6;               // M (required for reconstruction)
  uint32 total_shares = 7;            // N (total shares distributed)
  int64 timestamp_unix = 8;
  bytes enrollment_signature = 9;     // Ed25519 signature over (master_pubkey || recipient_pubkey || encrypted_share || nonce || share_index || threshold || total_shares || timestamp)
                                      // Signed by User's Master Key
  string recovery_label = 10;         // Optional label (e.g., "Surface Recovery" or "Specter Recovery")
}
```

**Validation Rules:**
- `enrollment_signature` MUST verify against `master_public_key`
- `threshold` MUST be ≤ `total_shares`
- `threshold` MUST be ≥ 2 (single-share recovery is pointless)
- `timestamp_unix` MUST be within ±300 seconds

**Delivery:**
- Sent via **direct message stream** (not broadcast) — `/murmur/recovery-enroll/1` libp2p stream protocol
- Recipient decrypts `encrypted_share` with their X25519 private key (ECDH shared secret → HKDF-SHA-256 → XChaCha20-Poly1305 decryption key)
- Recipient stores decrypted share in local Bbolt `recovery_shares` bucket (still encrypted at rest with recipient's passphrase)

---

### 2. Recovery Request

**Purpose:** User requests M shares from contacts to reconstruct Master Key.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message RecoveryRequest {
  bytes master_public_key = 1;        // User's identity root (contact looks up their stored share for this identity)
  bytes requester_public_key = 2;     // User's current device key (may differ from master if recovering from new device)
  bytes challenge_nonce = 3;          // 32-byte random nonce (proves User controls the recovery request)
  int64 timestamp_unix = 4;
  bytes request_signature = 5;        // Signed by User's current device key (if available) or ephemeral recovery key
  string recovery_label = 6;          // "Surface Recovery" or "Specter Recovery"
}
```

**Delivery:**
- Sent via direct message stream `/murmur/recovery-request/1` to each contact individually
- Contact verifies `request_signature` (if User has a device key) or relies on out-of-band verification (phone call, video chat)

---

### 3. Recovery Response

**Purpose:** Contact provides their secret share to User upon verifying the request.

**Protobuf Addition to `proto/identity.proto`:**

```protobuf
message RecoveryResponse {
  bytes master_public_key = 1;        // User's identity root
  bytes encrypted_share = 2;          // XChaCha20-Poly1305 encrypted SSS share (re-encrypted for User's current public key)
  bytes nonce = 3;                    // Fresh nonce for this response
  uint32 share_index = 4;             // Original share index (needed for Lagrange interpolation)
  int64 timestamp_unix = 5;
  bytes contact_signature = 6;        // Signed by Contact's Ed25519 key (proves authenticity)
}
```

**Delivery:**
- Sent via direct message stream `/murmur/recovery-response/1` back to User
- User decrypts `encrypted_share` with their X25519 private key
- User collects M responses, then reconstructs Master Key using SSS

---

## Enrollment Flow

### User Experience

**Scenario:** User wants to set up social recovery for their Surface identity with 3-of-5 threshold (5 contacts, need 3 to recover).

**Steps:**

1. **User selects contacts:**
   - Navigate to Settings → Recovery → "Set Up Social Recovery"
   - System prompts: "Select trusted contacts who can help you recover your identity if you lose your recovery phrase. You'll need 3 of 5 to cooperate."
   - User selects 5 contacts from their connection list (Surface connections only; Specter contacts listed separately for Specter recovery)

2. **User confirms threshold:**
   - System displays: "You've selected 5 contacts. How many should be required to recover? (Recommended: 3)"
   - User confirms threshold M = 3

3. **System generates shares:**
   - Retrieve User's Master Private Key from encrypted keystore (passphrase prompt)
   - Generate 5 Shamir shares with threshold M=3 using `github.com/hashicorp/vault/shamir.Split(masterPrivKey, 3, 5)`
   - For each contact:
     - Retrieve contact's X25519 public key from peer store
     - Derive shared secret via ECDH
     - Encrypt share with XChaCha20-Poly1305 (shared secret → HKDF → symmetric key)
     - Construct `RecoveryShareEnrollment` message
     - Sign with User's Master Key

4. **System distributes shares:**
   - For each contact, open direct stream `/murmur/recovery-enroll/1`
   - Send `RecoveryShareEnrollment` message
   - Wait for acknowledgment (contact confirms receipt)
   - Display progress: "Sent share to Alice... Bob... Carol... Dave... Eve..."

5. **Completion:**
   - System displays: "✓ Social recovery set up successfully. Your identity can now be recovered with help from 3 of your 5 contacts."
   - User's local database stores:
     - List of enrolled contacts (pubkeys, share indices)
     - Encrypted backup of all shares (optional, for User's own records)
     - Threshold M and total shares N

**Time:** 60–120 seconds (depends on number of contacts and network latency).

**Contact Experience:**
- Contact receives notification: "Alice wants to designate you as a recovery contact. If they lose access, you may be asked to help them recover."
- Contact accepts → share stored in local Bbolt `recovery_shares` bucket
- Contact declines → User is notified, can select a different contact

---

## Recovery Flow

### User Experience

**Scenario:** User lost BIP-39 mnemonic and all devices. User needs to recover identity with help from contacts.

**Steps:**

1. **User installs MURMUR on new device:**
   - Launch → Onboarding screen
   - Select "Recover Identity" → "Social Recovery (requires help from contacts)"

2. **User enters recovery information:**
   - System prompts: "Enter your public key or a unique identifier (e.g., pseudonym, sigil color)"
   - User provides master_public_key (if they have it written down) OR pseudonym/sigil (system searches DHT for matching identity)
   - System retrieves User's public identity information from network

3. **User contacts recovery participants:**
   - System displays: "Your identity requires 3 contacts to cooperate. Contact your trusted friends and ask them to approve your recovery request."
   - System generates ephemeral recovery keypair (Ed25519) for this session only
   - System constructs `RecoveryRequest` signed with ephemeral key + challenge nonce

4. **User sends recovery request to contacts:**
   - For each contact (selected from User's old contact list, if available, or manually entered):
     - System opens direct stream `/murmur/recovery-request/1`
     - Sends `RecoveryRequest`
     - Displays: "Sent recovery request to Alice... Bob... Carol... Dave... Eve..."

5. **Contacts verify and respond:**
   - Each contact receives request: "Someone is trying to recover Alice's identity. Do you approve?"
   - Contact may verify out-of-band (phone call, video chat) to confirm it's really User
   - Contact approves → Contact retrieves share from local store, re-encrypts with User's ephemeral public key, sends `RecoveryResponse`
   - Contact declines → User is notified

6. **System reconstructs Master Key:**
   - User collects M=3 responses (e.g., from Alice, Bob, Carol)
   - System decrypts each share with ephemeral private key
   - System uses Shamir reconstruction: `masterPrivKey = shamir.Combine(shares)`
   - System verifies reconstructed key by checking if derived public key matches `master_public_key`

7. **User sets new passphrase:**
   - System prompts: "Identity recovered! Set a new passphrase to secure your account."
   - User enters new passphrase
   - System re-encrypts Master Key with new Argon2id-derived key
   - System generates new device keypair for this device (multi-device identity pattern from §3.2)

8. **Recovery complete:**
   - System broadcasts `DeviceAuthorizationDeclaration` for new device
   - System displays: "✓ Identity recovered. You can now use MURMUR as before."
   - **Optional:** System prompts User to revoke old devices (if User suspects they're compromised)

**Time:** 5–15 minutes (depends on contact responsiveness; out-of-band verification may take longer).

**Edge Case — Not Enough Contacts Respond:**
- If User receives <M responses within timeout (e.g., 24 hours), recovery fails
- System displays: "Recovery failed. Not enough contacts responded. You need 3 approvals but received only 2."
- User can retry or attempt BIP-39 mnemonic recovery if they find their backup

---

## Cross-Layer Recovery: Surface vs. Specter

**Critical Design Constraint:** Surface and Specter identities MUST use **independent SSS schemes**. Recovering one MUST NOT reveal the other.

### Separate Secret Sharing

**Surface Recovery:**
- Secret: Surface Master Private Key (32 bytes Ed25519)
- Contacts: Surface connections (friends visible on Pulse Map)
- Label: `"Surface Recovery"`
- Enrollment: separate `RecoveryShareEnrollment` messages with `recovery_label = "Surface Recovery"`

**Specter Recovery:**
- Secret: Specter Master Private Key (32 bytes Curve25519)
- Contacts: High-Resonance Specters (Resonance ≥200, anonymous trust relationships)
- Label: `"Specter Recovery"`
- Enrollment: separate `RecoveryShareEnrollment` messages with `recovery_label = "Specter Recovery"`

### No Shared Contacts

**Enforced Rule:** A contact can be enrolled for Surface recovery OR Specter recovery, but NOT both. This prevents cross-layer linkability via shared recovery participants.

**Implementation:**
- When User selects contacts for Specter recovery, system filters out any contacts who are already enrolled for Surface recovery (and vice versa)
- System displays warning: "You've selected Alice for both Surface and Specter recovery. This may reduce anonymity. Recommended: use different contacts for each layer."

### Anonymity-Preserving Specter Recovery (v1.1 Future Work)

**Problem:** When Specter contacts receive `RecoveryRequest` with `master_public_key`, they learn "someone is recovering the Specter with pubkey X". If adversary correlates this with other metadata (timing, contact graph), they may deanonymize the Specter.

**Solution (Deferred):** Zero-Knowledge Proof of Share Possession
- User sends `RecoveryRequest` with ZK proof: "I possess a valid share for SOME Specter recovery enrollment you participated in" (without revealing which Specter)
- Contact verifies proof, responds with encrypted share IF proof is valid
- Contact does NOT learn which Specter is being recovered (preserves anonymity)

**Implementation Complexity:** Requires ZK circuit for SSS share possession; involves Bulletproofs or Groth16 SNARKs. Deferred to v1.1+ due to complexity and performance costs (proof generation ~1 second, verification ~100ms).

---

## Security Analysis

### Threat Model

| Threat | Mitigation |
|--------|------------|
| **Adversary compromises <M contacts** | Cannot reconstruct Master Key (SSS security). No information leak about secret. |
| **Adversary compromises ≥M contacts** | Can reconstruct Master Key → full identity compromise. **Mitigation:** User chooses threshold M high enough that compromising M contacts is infeasible (e.g., 5-of-7 requires adversary to compromise 5 independent contacts). |
| **Malicious contact refuses to respond** | Recovery fails gracefully if <M contacts respond. User can retry or use BIP-39 mnemonic. No partial information leak. |
| **MITM during share distribution** | Shares are encrypted with contact's X25519 public key. MITM cannot decrypt without private key. Noise-encrypted libp2p streams prevent traffic inspection. |
| **Replay attacks on recovery requests** | `challenge_nonce` (32-byte random) prevents replays. Each recovery request uses fresh nonce. Contact checks timestamp (±300s window). |
| **Adversary impersonates User during recovery** | Contact should verify out-of-band (phone call, video chat) before responding to recovery request. System displays warning: "Verify this request with Alice via phone call before approving." |
| **Cross-layer deanonymization (Surface ↔ Specter)** | Separate SSS schemes for Surface and Specter. No shared contacts. Future ZK proofs prevent contacts from learning which identity they're recovering. |
| **Contact loses their share** | If User enrolled N contacts and loses M-1 contacts, recovery fails. **Mitigation:** User re-enrolls contacts periodically (e.g., annually) and maintains N > M with margin (e.g., 3-of-5 instead of 3-of-3). |

### Comparison to Alternatives

| Approach | Pros | Cons |
|----------|------|------|
| **BIP-39 only (current)** | Simple; no dependencies on others | Single point of failure; high anxiety |
| **Shamir Secret Sharing (this design)** | Distributed trust; no single point of failure | Requires M contacts to cooperate; contact availability risk |
| **Cloud backup (Signal, WhatsApp)** | High availability; easy recovery | Requires trusting cloud provider (contradicts MURMUR's "no servers" principle) |
| **Threshold signatures (multi-sig)** | Can sign messages without full key reconstruction | More complex; requires all signers online simultaneously; not suitable for recovery (need full key) |

---

## Storage Schema

### Bbolt Bucket: `recovery_shares`

**Key-Value Structure:**

```
Key: (master_public_key || recipient_public_key)  // Composite key (64 bytes)
Value: RecoveryShareRecord (protobuf message)

message RecoveryShareRecord {
  bytes encrypted_share = 1;           // Still encrypted with recipient's passphrase (defense in depth)
  uint32 share_index = 2;
  uint32 threshold = 3;
  uint32 total_shares = 4;
  int64 enrolled_at_unix = 5;
  string recovery_label = 6;           // "Surface Recovery" or "Specter Recovery"
  bytes enrollment_signature = 7;      // Original signature from User's Master Key (for verification)
}
```

### Local Keystore Enhancement

**User's Keystore** (in addition to multi-device identity from §3.2):

```json
{
  "version": 3,
  "master_keypair": { ... },
  "device_keypair": { ... },
  "recovery_enrollment": {
    "surface": {
      "threshold": 3,
      "total_shares": 5,
      "enrolled_contacts": [
        {
          "contact_pubkey": "<hex>",
          "share_index": 1,
          "enrolled_at": "2026-05-06T03:56:01Z"
        },
        ...
      ],
      "backup_shares": {
        "encrypted_shares": "<base64>",  // Encrypted with User's passphrase (optional backup)
        "salt": "<base64>",
        "nonce": "<base64>"
      }
    },
    "specter": {
      "threshold": 2,
      "total_shares": 3,
      "enrolled_contacts": [ ... ],
      "backup_shares": { ... }
    }
  }
}
```

---

## Implementation Checklist

**Phase 1: Shamir Library Integration (1 day)**
- [ ] Add `github.com/hashicorp/vault/shamir` to `go.mod`
- [ ] Unit test: split 32-byte secret into 3-of-5 shares, reconstruct with any 3
- [ ] Unit test: verify M-1 shares cannot reconstruct secret

**Phase 2: Protobuf Schema (1 day)**
- [ ] Add `RecoveryShareEnrollment`, `RecoveryRequest`, `RecoveryResponse` to `proto/identity.proto`
- [ ] Add `RecoveryShareRecord` message
- [ ] Regenerate `.pb.go` files

**Phase 3: Enrollment Logic (3 days)**
- [ ] Implement `pkg/identity/recovery/enroll.go`:
  - [ ] `EnrollRecoveryContacts(masterKey, contacts []Contact, threshold, totalShares) error`
  - [ ] Split Master Key into N shares using Shamir
  - [ ] For each contact: ECDH key exchange, encrypt share, construct enrollment message
  - [ ] Send enrollment via `/murmur/recovery-enroll/1` stream
- [ ] Implement contact-side receiver: parse enrollment, decrypt share, store in `recovery_shares` bucket
- [ ] Unit tests with in-memory Bbolt and mock libp2p streams

**Phase 4: Recovery Logic (3 days)**
- [ ] Implement `pkg/identity/recovery/recover.go`:
  - [ ] `RequestRecovery(masterPubKey, contacts []Contact) ([]RecoveryResponse, error)`
  - [ ] Generate ephemeral recovery keypair
  - [ ] Send `RecoveryRequest` to each contact via `/murmur/recovery-request/1`
  - [ ] Collect M responses with timeout (24 hours)
  - [ ] `ReconstructMasterKey(shares []RecoveryResponse) ([]byte, error)` — Shamir reconstruction
  - [ ] Verify reconstructed key by checking derived public key matches expected
- [ ] Implement contact-side responder: verify request, retrieve share, re-encrypt for requester, send response
- [ ] Unit tests for full enrollment + recovery cycle

**Phase 5: Storage Layer (1 day)**
- [ ] Add `recovery_shares` bucket to Bbolt schema
- [ ] Implement `pkg/store/recovery.go`:
  - [ ] `StoreRecoveryShare(masterPubKey, recipientPubKey, record) error`
  - [ ] `GetRecoveryShare(masterPubKey, recipientPubKey) (*RecoveryShareRecord, error)`
  - [ ] `ListRecoveryEnrollments(recipientPubKey) ([]RecoveryShareRecord, error)` (for contact to see all enrollments they participate in)

**Phase 6: UI Flows (4 days)**
- [ ] Enrollment UI (`pkg/onboarding/screens/recovery_enroll_screen.go`):
  - [ ] Contact selection screen with multi-select checkboxes
  - [ ] Threshold slider (2 to N)
  - [ ] Progress indicator during share distribution
  - [ ] Confirmation screen with enrolled contact list
- [ ] Recovery UI (`pkg/onboarding/screens/recovery_social_screen.go`):
  - [ ] Identity lookup (master_pubkey or pseudonym search)
  - [ ] Contact selection for recovery request
  - [ ] Progress indicator: "Waiting for responses (2 of 3 received)..."
  - [ ] Success screen: "Identity recovered!"
  - [ ] Failure screen: "Not enough contacts responded. Retry or use BIP-39."
- [ ] Settings screen: view enrolled contacts, re-enroll, cancel enrollment

**Phase 7: Cross-Layer Enforcement (1 day)**
- [ ] Enforce separate Surface and Specter enrollments (distinct `recovery_label` fields)
- [ ] Warn User if selecting same contact for both Surface and Specter recovery
- [ ] Filter contact lists: Surface connections only for Surface recovery, Specter contacts only for Specter recovery

**Phase 8: Integration Testing (2 days)**
- [ ] End-to-end test: User A enrolls 3-of-5 contacts, loses Master Key, recovers with any 3 contacts
- [ ] Test M-1 contacts respond (recovery fails gracefully)
- [ ] Test adversary with M-1 shares (cannot reconstruct secret)
- [ ] Test separate Surface + Specter recovery (no cross-layer linkability)
- [ ] Race detector clean (`go test -race ./...`)

**Total Estimate:** 16 days (3 engineering weeks)

---

## Success Criteria

1. **Enrollment Functionality:**
   - User can enroll 2 to 10 contacts with arbitrary threshold M (2 ≤ M ≤ N)
   - Enrollment completes in <120 seconds for 5 contacts
   - Contacts receive shares and acknowledge receipt
   - Separate enrollments for Surface and Specter identities work correctly

2. **Recovery Functionality:**
   - User with M shares can reconstruct Master Key (100% success rate in tests)
   - User with M-1 shares CANNOT reconstruct Master Key (verified in unit tests)
   - Recovery request + response cycle completes within 24 hours (depends on contact availability)
   - Reconstructed Master Key correctly derives expected public key

3. **Security:**
   - Adversary with M-1 shares learns nothing about Master Key (verified via statistical tests on Shamir output)
   - Shares are encrypted in transit (X25519 ECDH + XChaCha20-Poly1305)
   - Shares are encrypted at rest (recipient's passphrase + Argon2id)
   - No cross-layer linkability: Surface recovery and Specter recovery use independent SSS schemes

4. **UX:**
   - Enrollment takes <120 seconds for 5 contacts
   - Recovery UI displays real-time progress ("2 of 3 contacts responded")
   - Contact verification warnings prompt out-of-band checks (phone/video)
   - Failure modes (timeout, not enough responses) display actionable error messages

5. **Reliability:**
   - Zero race conditions (`go test -race ./...`)
   - Works across network partitions (recovery requests retry with exponential backoff)
   - Recovery succeeds even if User has no device keys (ephemeral recovery keypair pattern)

---

## Future Enhancements (Post-v1.0)

1. **ZK Proofs for Specter Recovery:**
   - Contacts do NOT learn which Specter is being recovered
   - User proves "I possess a valid share for one of your enrollments" without revealing identity
   - Implementation: Bulletproofs or Groth16 SNARKs over SSS share possession circuit

2. **Hierarchical Recovery:**
   - "Admin contacts" can re-enroll new contacts without requiring User's Master Key
   - Useful if User wants to rotate contacts but has lost Master Key access

3. **Time-Locked Recovery:**
   - Shares become valid only after time delay (e.g., 7 days)
   - Prevents attacker with stolen phone from immediately recovering identity (User has 7 days to revoke enrollment)

4. **Re-Enrollment Reminders:**
   - System prompts User annually: "Your recovery contacts were enrolled 1 year ago. Verify they're still available."
   - Automated re-enrollment flow (re-encrypt shares with fresh nonces, update timestamps)

5. **Recovery Simulation:**
   - "Test Recovery" feature: User initiates recovery simulation (contacts respond but Master Key is NOT actually reconstructed)
   - Verifies contacts are responsive and have stored shares correctly

---

## References

- **PLAN.md §3.3** — Original task specification
- **BIP39_RECOVERY_AUDIT.md §Gap 2** — Problem statement (no social recovery)
- **SECURITY_PRIVACY.md §2.1** — Key material handling requirements
- **THREAT_MODEL.md** — Adversary classes and mitigation strategies
- **MULTI_DEVICE_IDENTITY.md** — Multi-device architecture (Master Key vs. Device Key pattern)
- **Shamir's Secret Sharing Algorithm** — Original paper by Adi Shamir (1979)
- **Hashicorp Vault Shamir Implementation** — `github.com/hashicorp/vault/shamir` (Go library)
