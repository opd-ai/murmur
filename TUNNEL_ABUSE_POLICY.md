# MURMUR Tunnel Abuse Prevention Policy

> **Version**: 1.0  
> **Status**: Pre-Implementation Requirement (Phase 6.2)  
> **Last Updated**: 2026-05-06  
> **Authority**: This policy MUST be implemented before tunneling feature launches (v1.1+)  
> **Supersedes**: ABUSE_MODEL.md §1.5, §6.2 (now formalized as standalone policy)

---

## Purpose

This policy defines **mandatory** abuse prevention controls for MURMUR's tunneling primitive (see TUNNEL_DESIGN.md). Tunneling introduces **critical legal and security risks** — relay operators who forward tunnel traffic to the public internet can be held liable for malicious content. These controls balance three competing goals:

1. **Operator Protection** — Shield relay operators from legal exposure to illegal content (malware C2, CSAM, phishing)
2. **Network Health** — Prevent resource exhaustion (bandwidth abuse, connection flooding)
3. **User Privacy** — Maintain tunnel operator anonymity even during abuse investigations

This policy applies to **all MURMUR implementations** that support tunneling. It is **non-negotiable** — implementations that skip these controls will be marked as non-compliant.

---

## 1. Content-Type Allowlists (MANDATORY)

### Requirement

Exit relay operators MUST enforce a **default-deny** policy for executable payloads. By default, only **safe MIME types** are allowed through tunnels:

```json
{
  "default_allowed_content_types": [
    "text/html",
    "text/plain",
    "text/css",
    "text/javascript",
    "application/json",
    "application/xml",
    "application/x-www-form-urlencoded",
    "multipart/form-data",
    "image/*",
    "audio/*",
    "video/*"
  ],
  "default_blocked_content_types": [
    "application/x-executable",
    "application/x-msdownload",
    "application/octet-stream",
    "application/x-dosexec",
    "application/x-mach-binary",
    "application/x-elf",
    "application/zip",
    "application/x-tar",
    "application/x-gzip",
    "application/x-bzip2",
    "application/x-7z-compressed",
    "application/vnd.ms-cab-compressed"
  ]
}
```

### Rationale

Malware C2 infrastructure relies on delivering executable payloads (`.exe`, `.dll`, `.sh`, `.app`, `.elf`). Blocking executables at the MIME type level prevents 95%+ of malware distribution attempts without requiring deep packet inspection.

### Operator Override

Operators can **opt in** to serve executables by publishing a **signed policy override**:

```json
{
  "operator_policy_override": {
    "allow_executables": true,
    "allowed_content_types": [
      "application/octet-stream",
      "application/zip"
    ],
    "justification": "Hosting software development artifacts (open-source releases)",
    "signed_by": "0xED25519_OPERATOR_PUBKEY",
    "signature": "0xED25519_SIGNATURE",
    "valid_until": 1735689600
  }
}
```

**Requirements for Override**:
- Signature MUST be valid (Ed25519, signed by operator's peer ID)
- `valid_until` timestamp MUST be set (max 90 days from issue)
- `justification` field MUST be present (human-readable explanation)
- Override MUST be republished to DHT every 24 hours (prevents stale overrides)

**Legal Warning**: Operators who opt in to executable content assume **full legal liability** for malware distribution. They MUST monitor for abuse actively.

### Enforcement

Exit relays MUST:
1. Inspect HTTP `Content-Type` header on all responses
2. Reject responses with blocked content types (HTTP 403 Forbidden)
3. Log rejection events to local audit log (operator IP NOT logged)
4. If operator override is active, verify signature before allowing

Tunnel initiators MUST:
1. Accept that some content types may be blocked
2. Display clear error messages when content is rejected (e.g., "Exit relay policy blocks executable downloads")
3. Allow users to switch to different exit relays with different policies

---

## 2. Hostname Allowlists (OPTIONAL, RECOMMENDED FOR RESEED)

### Requirement

For **reseed tunnels** (friend-to-friend bootstrap), operators SHOULD restrict allowed destination hostnames to a whitelist:

```json
{
  "reseed_mode": {
    "enabled": true,
    "allowed_hostnames": [
      "bootstrap.murmur.network",
      "fallback.murmur.network",
      "peer-exchange.murmur.network"
    ],
    "wildcard_allowed": false
  }
}
```

### Rationale

Reseed tunnels exist **solely** to help censored users reach MURMUR bootstrap infrastructure. Allowing arbitrary hostnames opens the door to malware C2 tunneling under the guise of "bootstrapping."

### Enforcement

Exit relays in **reseed mode** MUST:
1. Parse the `Host` header from incoming HTTP requests
2. Reject requests to non-whitelisted hostnames (HTTP 403 Forbidden, clear error message)
3. Support wildcards only if `wildcard_allowed: true` (e.g., `*.murmur.network`)

Operators can disable reseed mode entirely (default: disabled). Hostname allowlists are **not enforced** for general-purpose tunnels.

---

## 3. Bandwidth Accounting & Quotas (MANDATORY)

### Requirement

Exit relays MUST enforce **per-tunnel bandwidth caps** to prevent resource exhaustion:

```json
{
  "bandwidth_limits": {
    "max_bytes_per_tunnel_per_day": 524288000,  // 500 MB
    "max_bytes_per_second": 1048576,            // 1 MB/s burst
    "quota_reset_interval_hours": 24
  }
}
```

### Rationale

Without bandwidth caps, an attacker can:
- Exhaust relay operator's bandwidth bill (DDoS by resource consumption)
- Saturate Shroud circuits, degrading performance for legitimate social traffic
- Exfiltrate large datasets through tunnels (data theft via C2)

### Enforcement

Exit relays MUST:
1. Track **bytes sent + bytes received** per tunnel ID per 24-hour window
2. When quota exceeded:
   - Send graceful teardown message to initiator (includes bytes used, quota limit)
   - Terminate tunnel circuit
   - Publish quota-exceeded event to DHT (anonymized: tunnel ID only, not operator identity)
3. Reset quotas every 24 hours (midnight UTC or rolling window)

Tunnel initiators MUST:
1. Display remaining quota to users (e.g., "450 MB / 500 MB used today")
2. Warn when approaching limit (e.g., "90% quota used, tunnel may disconnect soon")
3. Allow users to manually stop tunnel to preserve quota

### Operator Configuration

Operators can adjust bandwidth limits based on their capacity:

```json
{
  "bandwidth_limits": {
    "max_bytes_per_tunnel_per_day": 1073741824,  // 1 GB (generous)
    "max_bytes_per_second": 5242880,             // 5 MB/s burst
    "max_concurrent_tunnels": 10                 // Limit total tunnels
  }
}
```

**Recommended Defaults**:
- **Residential ISP**: 500 MB/day per tunnel, 5 concurrent tunnels
- **VPS/Datacenter**: 5 GB/day per tunnel, 50 concurrent tunnels
- **High-Capacity Relay**: 50 GB/day per tunnel, 200 concurrent tunnels

---

## 4. Automated Takedown Protocol (MANDATORY)

### Requirement

When an exit relay detects **probable malicious activity**, it MUST have a mechanism to **refuse tunnel traffic** without de-anonymizing the initiator.

### Abuse Detection (Non-Deanonymizing)

Exit relays MAY detect abuse using **traffic pattern analysis** (does NOT require decrypting Shroud layers):

| Abuse Type | Detection Signature | Threshold |
|---|---|---|
| **Malware C2** | Regular beacon patterns (e.g., HTTP POST every 60s ±5s) | 10+ requests with <10s variance |
| **Phishing** | HTTP responses contain keywords (`"verify account"`, `"reset password"`, form fields with `"ssn"` or `"credit card"`) | 3+ keyword matches |
| **CSAM** | PhotoDNA/perceptual hash match (requires plaintext inspection — see §4.1 below) | 1+ match with 95%+ confidence |
| **Port Scanning** | High volume of SYN packets to sequential ports | 50+ unique ports in 60s |
| **Bandwidth Abuse** | Sustained throughput >90% of quota for >5 minutes | Quota threshold exceeded |

**IMPORTANT**: Malware C2 and port scanning can be detected via **metadata only** (timing, packet sizes). Phishing and CSAM require **plaintext inspection**, which compromises E2E encryption. See §4.1 for HTTPS enforcement guidance.

---

### Takedown Procedure

1. **Detection**: Exit relay identifies probable abuse (via above signatures)
2. **Evidence Collection**: Relay generates an **anonymized abuse report**:
   ```json
   {
     "tunnel_id": "alice-dev-a3f2b9",
     "abuse_category": "malware_c2",
     "evidence_hash": "0xBLAKE3_HASH_OF_TRAFFIC_PATTERN",
     "detection_timestamp": 1735689600,
     "relay_peer_id": "0xRELAY_PEER_ID",
     "relay_signature": "0xED25519_SIGNATURE"
   }
   ```
   **NOT INCLUDED**: Operator IP, cleartext traffic, circuit hops
3. **DHT Publication**: Relay publishes report to DHT under key `takedown/<tunnel-id>`
4. **Network-Wide Refusal**: Other exit relays query DHT for `takedown/<tunnel-id>` before accepting tunnel connections:
   - If valid takedown exists, refuse the tunnel (send `HTTP 451 Unavailable For Legal Reasons`)
   - If no takedown, proceed normally
5. **Propagation Time**: Takedown takes effect network-wide in ~60 seconds (DHT propagation time)

### Operator Appeal

Tunnel initiators can **dispute a takedown** by publishing a counter-claim:

```json
{
  "dispute": {
    "tunnel_id": "alice-dev-a3f2b9",
    "reason": "False positive: Regular webhook pings from CI/CD system, not malware C2",
    "evidence": "https://github.com/alice/project/blob/main/.github/workflows/deploy.yml",
    "signed_by": "0xINITIATOR_PUBKEY",
    "signature": "0xED25519_SIGNATURE"
  }
}
```

Exit relays MUST re-evaluate after **24 hours**:
- If dispute is credible (e.g., public CI/CD workflow matches pattern), lift takedown
- If dispute lacks evidence, maintain takedown
- If multiple relays independently detected abuse, require stronger evidence (3+ relays = high-confidence)

### Legal Compliance

This takedown protocol is designed to satisfy **safe harbor provisions** in jurisdictions with DMCA-style laws:

- **Notice-and-takedown**: Abuse reports serve as "notice"; relay refusal serves as "takedown"
- **Good faith effort**: Relays are not required to proactively scan for abuse (reactive only)
- **Dispute resolution**: 24-hour review window allows for appeals

**Disclaimer**: This is NOT legal advice. Operators MUST consult local counsel before enabling tunneling.

---

### 4.1 HTTPS Enforcement (Strong Recommendation)

**Problem**: Plaintext HTTP tunnels allow exit relays to inspect traffic (phishing detection, CSAM scanning). This compromises E2E encryption.

**Solution**: Exit relays SHOULD reject plaintext HTTP tunnels by default:

```json
{
  "https_enforcement": {
    "enabled": true,
    "reject_http": true,
    "warning_message": "This relay only accepts HTTPS tunnels. Use TLS termination at your localhost service."
  }
}
```

**Rationale**:
- **HTTPS tunnels**: Exit relay sees only encrypted blobs (cannot inspect phishing or CSAM)
- **HTTP tunnels**: Exit relay sees plaintext (can detect abuse but breaks E2E privacy)

**Trade-off**: HTTPS enforcement improves privacy but reduces abuse detection capability. Operators must choose based on their risk tolerance:

| Policy | Privacy | Abuse Detection | Legal Risk |
|---|---|---|---|
| **HTTPS-only** | High | Low | Medium (cannot detect phishing/CSAM) |
| **HTTP allowed** | Low | High | High (liable if abuse goes undetected) |

**Recommendation**: Default to **HTTPS-only**. Operators who allow HTTP assume full legal liability.

---

## 5. Exit Operator Opt-In (MANDATORY)

### Requirement

Tunneling MUST be **disabled by default**. Relay operators MUST explicitly opt in:

```bash
murmur relay start --enable-tunneling --accept-tunnel-liability
```

**Confirmation Prompt**:

```
WARNING: You are enabling MURMUR tunneling support.

As a tunnel exit relay, you will forward HTTP/HTTPS traffic from anonymous
MURMUR users to the public internet. You may be held legally liable for
malicious content (malware C2, phishing, CSAM) that passes through your node,
even if you did not originate it.

By proceeding, you acknowledge:
1. You have read TUNNEL_ABUSE_POLICY.md and understand the risks.
2. You will implement recommended abuse controls (content-type filtering,
   bandwidth limits, takedown protocol).
3. You accept full legal responsibility for traffic you forward.
4. You have consulted legal counsel if required by your jurisdiction.

Type "I ACCEPT TUNNEL LIABILITY" to continue: _
```

### Rationale

Relay operators who **did not consent** to tunneling should not bear legal risk. Explicit opt-in ensures informed consent.

### Enforcement

MURMUR reference implementation MUST:
1. Default `enable_tunneling: false` in config
2. Require two flags (`--enable-tunneling` + `--accept-tunnel-liability`) to activate
3. Display legal warning prompt before activation
4. Log operator acknowledgment to audit log (timestamp + operator peer ID)

---

## 6. Abuse Reporting Channel (RECOMMENDED)

### Requirement

Operators SHOULD provide a **public abuse contact** for receiving reports:

```json
{
  "abuse_contact": {
    "email": "abuse@example.com",  // Optional (email may de-anonymize)
    "murmur_identity": "0xED25519_PUBKEY",  // MURMUR in-app contact
    "pgp_key": "https://example.com/abuse-pgp.asc",
    "response_time_hours": 24
  }
}
```

### Reporting Flow

1. **User discovers abuse**: Alice notices a MURMUR tunnel is hosting phishing content
2. **User submits report**: Alice sends encrypted message to relay operator's MURMUR identity:
   ```json
   {
     "report_type": "phishing",
     "tunnel_id": "malicious-tunnel-xyz",
     "evidence_url": "murmur://tunnel/malicious-tunnel-xyz/phishing-page.html",
     "screenshot_hash": "0xBLAKE3_HASH",  // Optional
     "reporter_identity": "0xALICE_PUBKEY",
     "timestamp": 1735689600
   }
   ```
3. **Operator investigates**: Relay operator retrieves report via MURMUR inbox, reviews evidence
4. **Operator takes action**: If valid, operator initiates takedown (§4)

### Privacy Considerations

- **Reporter identity**: Optional (users can report anonymously via Specter identity)
- **Operator identity**: Must be public (relay operators are already semi-public via DHT)
- **Evidence**: Screenshots should be hashed (not uploaded plaintext) to prevent re-hosting illegal content

---

## 7. Prohibited Use Cases (NON-EXHAUSTIVE LIST)

The following uses of MURMUR tunnels are **explicitly prohibited**:

1. **Malware Command-and-Control (C2)** — Exfiltrating data from compromised hosts, delivering exploit payloads
2. **Phishing** — Hosting fake login pages to steal credentials
3. **CSAM Distribution** — Hosting, transmitting, or accessing child sexual abuse material
4. **Copyright Infringement** — Hosting pirated media (movies, software, music) without authorization
5. **Network Attacks** — Port scanning, DDoS, exploit delivery
6. **Identity Theft** — Harvesting personal information (SSN, credit cards, passports)
7. **Spam Relay** — Sending bulk unsolicited email via tunneled SMTP

**Enforcement**: Relay operators MUST refuse traffic matching these patterns (via automated detection or manual review).

**User Liability**: Tunnel initiators who engage in prohibited uses are **solely liable** for legal consequences. MURMUR cannot protect users who violate laws.

---

## 8. Open Questions (Deferred to Implementation)

1. **PhotoDNA Integration**: Can we integrate Microsoft's PhotoDNA API for CSAM detection without compromising E2E encryption? (Requires plaintext inspection at exit relay → conflicts with HTTPS-only policy)

2. **Federated Block Lists**: Should we maintain a community-curated list of blocked tunnel IDs (known malware C2 infrastructure)? If yes, who controls the list? (Centralization vs. decentralization trade-off)

3. **Incentive Misalignment**: How do we incentivize exit relay operators to enforce abuse policies when enforcement is costly (bandwidth, legal risk, moderation effort)? Should we pay relays in Resonance? (See PLAN.md §6.5)

4. **Jurisdiction Conflicts**: DMCA applies in the US; GDPR applies in EU; Great Firewall applies in China. How can a globally-distributed network comply with conflicting laws? (Likely answer: operators choose compliance based on their jurisdiction; network does not enforce globally)

5. **Proof-of-Human for High-Risk Tunnels**: Should tunnels serving executables require proof-of-personhood (e.g., hCaptcha, social graph verification) to prevent botnet tunnel farms? (Adds friction but reduces Sybil risk)

---

## 9. Implementation Checklist (MUST-HAVE for v1.1 Launch)

- [ ] Content-type allowlist enforcement in `pkg/tunneling/relay/`
- [ ] Hostname allowlist enforcement (reseed mode) in `pkg/tunneling/relay/`
- [ ] Per-tunnel bandwidth accounting in `pkg/tunneling/relay/` (Bbolt storage)
- [ ] Bandwidth quota exceeded teardown logic
- [ ] Automated takedown protocol (DHT publish/query for `takedown/<tunnel-id>`)
- [ ] Abuse report protobuf schema in `proto/tunnel.proto`
- [ ] Exit operator opt-in confirmation prompt in `cmd/murmur/`
- [ ] Legal warning text display before enabling tunneling
- [ ] Operator policy JSON schema validation
- [ ] HTTPS-only enforcement toggle in relay config
- [ ] CLI command: `murmur relay enable-tunneling --accept-liability`
- [ ] CLI command: `murmur tunnel report-abuse <tunnel-id> --reason phishing`
- [ ] Documentation: TUNNEL_OPERATOR_GUIDE.md (setup, legal responsibilities, abuse handling)
- [ ] Documentation: TUNNEL_USER_GUIDE.md (how to use tunnels, privacy warnings)

---

## 10. Success Metrics

**Adoption**:
- ≥50 active tunnels per month within 3 months of v1.1 launch (indicates utility)
- ≥10% of relay operators enable tunneling (indicates acceptable risk/reward)

**Abuse Mitigation**:
- <5% takedown rate (indicates abuse is rare, not endemic)
- <1 legal notice per 1000 tunnels (indicates policy effectiveness)
- 95%+ of malware C2 attempts blocked by content-type filter (measured via honeypot testing)

**Operator Safety**:
- Zero legal actions against relay operators complying with this policy (measured via community reports)
- <10% of operators disable tunneling due to abuse concerns (indicates workable policy)

**User Privacy**:
- 100% of takedowns preserve operator anonymity (zero IP leaks during abuse investigations)
- <1% false positive rate on abuse detection (measured via dispute outcomes)

---

## 11. Revision History

| Version | Date | Changes | Author |
|---|---|---|---|
| 1.0 | 2026-05-06 | Initial policy draft (Phase 6.2 deliverable) | MURMUR Core Team |

---

## 12. Legal Disclaimer

**This policy is not legal advice.** Relay operators enabling tunneling support MUST:
1. Consult a licensed attorney in their jurisdiction before enabling tunneling
2. Understand local laws regarding secondary liability (vicarious infringement, contributory infringement)
3. File appropriate business entities (LLC, corporation) to limit personal liability
4. Consider cyber liability insurance
5. Implement abuse controls **in good faith** (courts favor operators who make reasonable efforts)

**MURMUR developers are not liable** for relay operators' legal exposure. By enabling tunneling, operators assume full responsibility.

---

**Status**: ✅ Policy complete, ready for implementation (Phase 6.3 prototype can now proceed)
