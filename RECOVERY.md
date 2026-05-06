# MURMUR Identity Recovery Guide

**Status:** User-Facing Documentation  
**Audience:** End users, support staff  
**Date:** 2026-05-06  
**Related:** Multi-Device Identity (§3.2), Social Recovery (§3.3), Key Rotation (§3.4)

---

## Overview

MURMUR provides **four ways to recover or protect your identity** when you lose access to your device, forget your passphrase, or experience a security incident:

1. **BIP-39 Recovery Phrase** (24 words) — Traditional seed phrase backup
2. **Multi-Device Identity** — Use multiple devices; losing one doesn't lose your identity
3. **Social Recovery** — Trusted contacts help you recover if you lose everything
4. **Key Rotation** — Replace compromised keys without starting over

This guide explains how each method works, when to use it, and what happens during recovery.

---

## Method 1: BIP-39 Recovery Phrase

### What It Is

When you create your MURMUR identity, you receive a **24-word recovery phrase** (also called a seed phrase or mnemonic). This phrase is the master key to your identity. With it, you can restore your identity on any device.

### When to Use

- **Device lost or stolen** — You need to recover your identity on a new device
- **Device upgrade** — Moving from old phone to new phone
- **Forgot passphrase** — You can still unlock your identity if you have the recovery phrase

### How to Recover

**Steps:**

1. **Install MURMUR** on your new device
2. **Launch the app** → Select "Recover Identity"
3. **Choose "Recovery Phrase (24 words)"**
4. **Enter your 24-word phrase** (type or paste from secure storage)
5. **Press "Recover"**
6. **Set a new passphrase** to protect your recovered identity
7. **Done!** Your identity is restored

**Time:** 90–200 seconds (faster if you paste from a password manager)

---

### What Gets Recovered

✅ **Your cryptographic identity** — Your public key, sigil (visual icon), pseudonym  
✅ **Ability to reconnect** — You can send messages and your contacts will recognize you  

❌ **Connections** — Your contact list is stored locally; you'll need to reconnect with contacts  
❌ **Message history** — Messages are ephemeral (expire after 7–30 days); old messages are gone  
❌ **Resonance score** — Reputation is computed locally; starts fresh on new device  
❌ **Game history** — Participation records, trophies, territory control are local state  
❌ **Council memberships** — You may need to rejoin Phantom Councils  

**Why?** MURMUR has no servers. Your identity (cryptographic keys) is portable, but local state (messages, connections, games) is not backed up automatically.

---

### How to Store Your Recovery Phrase Safely

**✅ DO:**
- Write it on paper and store in a safe place (fireproof safe, safety deposit box)
- Use a password manager (1Password, Bitwarden, KeePass) with strong master password
- Split it using "Social Recovery" (see Method 3) if you're worried about single point of failure

**❌ DON'T:**
- Store it in plaintext on your computer or phone (malware can steal it)
- Take a photo of it (photos sync to cloud and may be compromised)
- Email it to yourself (email is not secure)
- Share it with anyone (anyone with your phrase can impersonate you)

---

### Testing Your Backup

**Recommended:** Test your recovery phrase **before** you need it.

**Steps:**

1. Navigate to **Settings → Security → Test Recovery Phrase**
2. Enter your 24-word phrase
3. System verifies: "✓ Recovery phrase valid. This phrase will recover your current identity."

**Important:** This test only checks that your phrase is valid. It does NOT test whether your paper backup is legible or your password manager is accessible. Periodically verify your physical backup is still readable.

---

## Method 2: Multi-Device Identity

### What It Is

Instead of using the same private key on all your devices (dangerous), MURMUR lets you authorize **multiple devices** for one identity. Each device has its own keypair, but all are linked to your master identity.

### When to Use

- **You use multiple devices** — Phone, laptop, tablet
- **Device upgrade** — Add your new device without removing the old one
- **Proactive security** — If one device is compromised, revoke it without losing your identity

### How to Add a New Device

**Scenario:** You have MURMUR on your phone. You want to add your laptop.

**Steps:**

1. **On your phone (primary device):**
   - Navigate to **Settings → Devices → Add New Device**
   - Phone generates a QR code with pairing information
   - QR code expires in 5 minutes (security)

2. **On your laptop (new device):**
   - Install MURMUR → Launch → Select **"Add to Existing Identity"**
   - Scan the QR code from your phone (use laptop's webcam)

3. **On your phone (authorization):**
   - Phone prompts: "Authorize 'Laptop' to act as you?"
   - Confirm → Enter your passphrase (to unlock master key)
   - Phone signs authorization and broadcasts to network

4. **Done!** Your laptop is now authorized. You can send messages from either device.

**Time:** 30–60 seconds (assuming same local network)

---

### How to Revoke a Device

**Scenario:** Your phone was stolen. You want to prevent the thief from sending messages as you.

**Steps:**

1. **On your laptop (surviving device):**
   - Navigate to **Settings → Devices**
   - View list of authorized devices (Phone, Laptop, Tablet)
   - Select stolen phone → **"Revoke Device"**

2. **Confirm revocation:**
   - System prompts: "Revoke 'Phone'? This device will no longer be able to send messages as you."
   - Confirm → Enter passphrase (to unlock master key)

3. **Revocation takes effect:**
   - Your network immediately stops accepting messages from the stolen phone
   - Grace period (7 days) allows any in-flight messages to be accepted
   - After 7 days, stolen phone is completely blocked

**Time:** <60 seconds to revoke; 7-day grace period for network propagation

---

### Security Benefits

✅ **Device-specific keys** — Compromising one device doesn't compromise your master key  
✅ **Revocation without device** — You can revoke a lost device from any surviving device  
✅ **Master key offline** — Your master key stays encrypted, only used for device management  

❌ **Still need BIP-39 backup** — If you lose ALL devices, you still need your recovery phrase  

---

## Method 3: Social Recovery

### What It Is

You designate 3–7 **trusted contacts** who can help you recover your identity if you lose your recovery phrase. No single contact can recover your identity alone; you need a threshold (e.g., 3 out of 5) to cooperate.

### When to Use

- **You worry about losing your recovery phrase** — Paper burns, password managers get hacked
- **You want redundancy** — Multiple trusted people instead of single point of failure
- **Competitors offer this** — Signal, WhatsApp have social/cloud recovery; you want equivalent security

### How to Set Up Social Recovery

**Steps:**

1. Navigate to **Settings → Recovery → Set Up Social Recovery**
2. **Select trusted contacts** — Choose 5 friends from your connection list
3. **Set threshold** — "How many contacts required to recover? (Recommended: 3)"
4. **Confirm** — System encrypts and distributes secret shares to each contact
5. **Done!** Your identity can now be recovered with help from 3 of your 5 contacts

**Time:** 60–120 seconds (depends on number of contacts and network speed)

---

### How Your Contacts Experience This

When you designate someone as a recovery contact, they receive a notification:

> "Alice wants to designate you as a recovery contact. If they lose access, you may be asked to help them recover."

**If they accept:**
- They store an encrypted secret share (they cannot read it or use it alone)
- They may be asked to help you recover in the future

**If they decline:**
- You're notified and can select a different contact

---

### How to Recover with Social Recovery

**Scenario:** You lost your phone, forgot your recovery phrase, and need to recover your identity.

**Steps:**

1. **Install MURMUR on a new device** → Select "Recover Identity" → "Social Recovery"
2. **Enter your identity information** (your public key or pseudonym — ask a friend if you don't remember)
3. **Contact your recovery participants:**
   - System displays: "Your identity requires 3 contacts to cooperate."
   - Call/message your trusted contacts: "I need your help to recover my MURMUR identity. Please approve my recovery request."

4. **System sends recovery requests** to your contacts
5. **Contacts approve** (they verify it's really you — via phone call, video chat)
6. **System collects 3 approvals** and reconstructs your master key
7. **Set a new passphrase** to protect your recovered identity
8. **Done!** Your identity is restored

**Time:** 5–15 minutes (depends on how quickly contacts respond)

---

### Security Notes

✅ **No single point of trust** — No individual contact can recover your identity  
✅ **Threshold flexibility** — 2-of-3 (easy), 3-of-5 (balanced), 5-of-7 (paranoid)  
✅ **Separate for Surface and Specter** — Surface contacts can't recover your anonymous identity  

⚠️ **Contact availability** — If too many contacts are unavailable, recovery fails (choose reliable contacts)  
⚠️ **Out-of-band verification** — Contacts should verify it's really you before approving (phone call recommended)  

---

### Who Should Use Social Recovery?

**Use it if:**
- You have 3–7 trusted contacts (family, close friends)
- You're worried about losing your recovery phrase
- You want redundancy (multiple recovery paths)

**Skip it if:**
- You don't have trusted contacts yet
- You prefer solo control (BIP-39 phrase only)
- You use multi-device identity (reduces single-device risk already)

**Recommendation:** Set up social recovery AFTER you have an established social graph on MURMUR. New users should start with BIP-39 + multi-device identity.

---

## Method 4: Key Rotation

### What It Is

If your key is compromised (device stolen, malware suspected), you can generate a **new key** and broadcast a signed statement: "My old key is X, my new key is Y. Treat messages from Y as from me." Your contacts automatically recognize the new key.

### When to Use

- **Security incident** — You suspect your key was compromised
- **Proactive security** — Rotate keys annually (security best practice)
- **Device recovered after theft** — You got your phone back but don't trust it anymore

### How to Rotate Your Key

**Steps:**

1. Navigate to **Settings → Security → Rotate Identity Key**
2. **Confirm rotation:**
   - System displays: "Rotating your key will generate a new cryptographic identity. Your contacts will automatically recognize the new key. Old key remains valid for 7 days."
   - Select reason (optional): "Security incident", "Proactive rotation", "Device upgrade"
   - Confirm → Enter passphrase

3. **System generates new key** and broadcasts declaration
4. **Grace period (7 days):**
   - Your old key still works (allows time for network propagation)
   - Your new key also works (you can start using it immediately)

5. **After 7 days:**
   - Old key stops working
   - Only new key is valid

**Time:** 30–60 seconds

---

### What Your Contacts See

When you rotate your key, your contacts receive a notification:

> "Alice rotated their identity key (reason: proactive security measure)"

**No action required** — Their app automatically recognizes your new key. You remain "Alice" in their contact list. Your sigil (visual icon) stays the same.

---

### Security Benefits

✅ **No identity loss** — Keep your connections, social graph, Resonance  
✅ **Limits attacker window** — Attacker with old key has 7 days max before key expires  
✅ **Proactive hygiene** — Rotate keys annually to reduce long-term exposure  

⚠️ **Grace period** — Attacker with old key can still send messages for 7 days (or configure 1-day grace for urgent rotations)  

---

### When NOT to Rotate

**Don't rotate if:**
- You're just adding a new device (use multi-device identity instead — no need to rotate master key)
- You're switching phones (use BIP-39 recovery or multi-device pairing)

**Do rotate if:**
- You suspect key compromise (malware, data breach)
- Annual security hygiene (proactive rotation)
- Device was stolen then recovered (you don't trust it anymore)

---

## Comparison Table

| Method | Speed | What's Recovered | When to Use | Requirements |
|--------|-------|------------------|-------------|--------------|
| **BIP-39 Recovery** | 90–200 sec | Identity keys only (lose connections, messages) | Device lost/stolen, forgot passphrase | 24-word phrase |
| **Multi-Device Identity** | 30–60 sec | Full continuity (keep connections, messages) | Proactive (add devices before loss) | One surviving device |
| **Social Recovery** | 5–15 min | Identity keys (lose connections, messages) | Lost all devices + recovery phrase | 3–7 trusted contacts |
| **Key Rotation** | 30–60 sec | Full continuity (keep everything) | Key compromised, proactive security | Any authorized device |

---

## Failure Modes & Troubleshooting

### "I lost my 24-word recovery phrase AND all my devices"

**If you set up social recovery:**
→ Use **Method 3: Social Recovery** (contact your trusted contacts)

**If you did NOT set up social recovery:**
→ ❌ Your identity is unrecoverable. You must create a new identity. This is why we recommend:
1. Writing down your recovery phrase and storing it safely
2. Setting up social recovery with 3–5 trusted contacts
3. Using multi-device identity (reduces risk of losing all devices at once)

---

### "Social recovery failed — not enough contacts responded"

**Why it happens:**
- Contacts are offline or unavailable
- Contacts declined (they weren't sure it was really you)
- You selected unreliable contacts

**What to do:**
- Retry with different contacts (if available)
- Use BIP-39 recovery phrase if you have it
- If both fail, your identity is unrecoverable (create new identity)

**Prevention:**
- Choose reliable contacts (family, close friends who respond quickly)
- Set threshold with margin (3-of-5 instead of 3-of-3, so losing 2 contacts doesn't break recovery)
- Periodically test: "Hey, I'm going to send you a test recovery request. Can you approve it?"

---

### "I rotated my key but contacts still see messages from my old key"

**Why it happens:**
- Grace period (default 7 days) allows old key to remain valid
- This is intentional (allows time for network propagation)

**What to do:**
- Wait for grace period to expire (old key will stop working automatically)
- If urgent (confirmed compromise), you can revoke the old key immediately:
  - Settings → Security → Revoke Old Key → Confirm
  - Contacts will immediately reject messages from old key (no grace period)

---

### "I added a new device but now my old device stopped working"

**Why it happens:**
- You may have accidentally revoked the old device
- Or you rotated your master key (which invalidates old device keys)

**What to do:**
- Check Settings → Devices → View authorized devices
- If old device is not listed, re-authorize it (use "Add Device" from the device that still works)
- If you rotated master key, generate new device keys for both devices (can't reuse old device keys after master key rotation)

---

### "My contact says I rotated my key but I didn't"

**⚠️ Security Alert:** Someone may have compromised your key and rotated it without your consent.

**What to do:**

1. **Immediately revoke the unauthorized key:**
   - Settings → Security → Report Unauthorized Rotation
   - System will broadcast a revocation and rotate to a new legitimate key

2. **Change your passphrase** (in case attacker has access)

3. **Check your devices:**
   - Settings → Devices → Review authorized devices
   - Revoke any devices you don't recognize

4. **Investigate:**
   - Was your device stolen? Lost? Infected with malware?
   - Consider factory resetting compromised device

---

## Best Practices

### For Maximum Security

1. **Write down your 24-word recovery phrase** and store it in a safe place (not on your device)
2. **Set up social recovery** with 3–5 trusted contacts (redundancy)
3. **Use multi-device identity** (so losing one device doesn't lose your identity)
4. **Rotate your key annually** (proactive security hygiene)
5. **Test your recovery phrase yearly** (Settings → Security → Test Recovery Phrase)

### For Convenience

1. **Store recovery phrase in a password manager** (1Password, Bitwarden) with strong master password
2. **Set up multi-device identity** immediately (phone + laptop at minimum)
3. **Skip social recovery** if you have reliable BIP-39 backup

### For Paranoid Users (Maximum Privacy)

1. **Use social recovery with 5-of-7 threshold** (higher redundancy, higher security)
2. **Rotate keys quarterly** (reduces long-term exposure)
3. **Never store recovery phrase digitally** (paper only, in fireproof safe)
4. **Use separate contacts for Surface and Specter recovery** (prevents cross-layer linkability)

---

## FAQ

### Q: Can I recover my anonymous identity (Specter) the same way as my Surface identity?

**A:** Yes, but separately. Surface recovery and Specter recovery use independent processes. Your Surface contacts cannot recover your Specter identity (and vice versa). This prevents cross-layer deanonymization.

---

### Q: If I use social recovery, can my contacts read my messages?

**A:** No. Contacts hold encrypted secret shares. They cannot decrypt your messages or impersonate you individually. They can only cooperatively help you recover if you ask them to (and you need at least 3 of 5 to cooperate).

---

### Q: What if I lose contact with one of my recovery contacts?

**A:** As long as you still have M contacts available (e.g., 3 out of 5), recovery works. If you lose too many contacts, re-enroll new contacts:
- Settings → Recovery → Re-Enroll Contacts
- System distributes new shares to new contacts
- Old shares remain valid (you can use old OR new shares)

---

### Q: Can I use the same recovery phrase on multiple identities?

**A:** Technically yes, but **strongly discouraged**. If your recovery phrase is compromised, all identities using that phrase are compromised. Best practice: one recovery phrase per identity.

---

### Q: Does MURMUR store my recovery phrase or keys on a server?

**A:** **No.** MURMUR has no servers. Your recovery phrase and keys are stored only on your device (encrypted with your passphrase). If you lose your device and recovery phrase, and did not set up social recovery, your identity is unrecoverable.

---

### Q: How often should I rotate my key?

**Recommended:**
- **Annually** for proactive security (like changing passwords)
- **Immediately** if you suspect compromise (device stolen, malware detected)
- **Never** for routine device upgrades (use multi-device pairing instead)

---

## Getting Help

**If you're still stuck:**

1. Check the MURMUR documentation: [docs/](../docs/)
2. Ask in the MURMUR community (Pulse Map → Community Councils)
3. File an issue: https://github.com/opd-ai/murmur/issues

**Security incidents:**
- If you believe your key was compromised, rotate immediately (don't wait for help)
- If you suspect unauthorized activity, revoke all devices and start fresh

---

**Document Version:** 1.0  
**Last Updated:** 2026-05-06  
**Related Documentation:**
- `docs/MULTI_DEVICE_IDENTITY.md` — Technical specification for multi-device identity
- `docs/SOCIAL_RECOVERY.md` — Technical specification for Shamir secret sharing
- `docs/KEY_ROTATION.md` — Technical specification for key rotation
- `docs/BIP39_RECOVERY_AUDIT.md` — Audit of current recovery UX
