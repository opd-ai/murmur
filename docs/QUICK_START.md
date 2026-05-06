# MURMUR Quick Start Guide

**Version:** v0.1.0-rc1  
**Time to First Wave:** ~5 minutes  
**Target Audience:** New users, early adopters

---

## Prerequisites

- **Desktop computer** (Linux, macOS, or Windows)
- **Internet connection** (broadband recommended)
- **Display** (GUI required; headless not supported in RC1)
- **Disk space**: ~150 MB minimum

---

## Installation

### Option 1: Binary Release (Recommended)

1. **Download** the binary for your platform:
   - [Linux (amd64)](https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-linux-amd64.tar.gz)
   - [Linux (arm64)](https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-linux-arm64.tar.gz)
   - [macOS (amd64)](https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-darwin-amd64.tar.gz)
   - [macOS (arm64)](https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-darwin-arm64.tar.gz)
   - [Windows (amd64)](https://github.com/opd-ai/murmur/releases/download/v0.1.0-rc1/murmur-windows-amd64.zip)

2. **Extract** the archive:
   ```bash
   # Linux/macOS
   tar -xzf murmur-*.tar.gz
   
   # Windows
   # Right-click murmur-windows-amd64.zip → Extract All
   ```

3. **Run** the binary:
   ```bash
   # Linux/macOS
   ./murmur
   
   # Windows
   murmur.exe
   ```

### Option 2: Build from Source

**Prerequisites:** Go 1.22+ installed

```bash
# Clone the repository
git clone https://github.com/opd-ai/murmur.git
cd murmur
git checkout v0.1.0-rc1

# Build
go build -o murmur ./cmd/murmur

# Run
./murmur
```

---

## First Launch: 6-Phase Onboarding

MURMUR guides you through a 6-phase onboarding flow on first launch. This takes about **5 minutes**.

### Phase 1: Welcome

You'll see a welcome screen explaining MURMUR's core concepts:
- No servers (peer-to-peer architecture)
- Spatial interface (Pulse Map)
- Dual-layer identity (Surface + Specter)
- Ephemeral content (Waves expire)

**Action:** Press `Enter` or click "Continue"

---

### Phase 2: Identity Creation

MURMUR generates your cryptographic identity:

1. **Ed25519 Keypair** — Your Surface identity (32-byte public key, 32-byte private key)
2. **Visual Sigil** — Deterministic icon generated from your public key hash
3. **Display Name** — Choose a name (changeable later)

**Recovery Phrase:**
- You'll receive a **24-word BIP-39 recovery phrase**
- **Write this down and store it securely** (offline, ideally in multiple locations)
- This is the **only way** to recover your identity if you lose your keystore files or forget your passphrase

**Passphrase:**
- Choose a strong passphrase to encrypt your keystore files
- Passphrase is **not recoverable** — if you forget it, your keys are lost (unless you have your recovery phrase)
- Minimum 8 characters recommended

**Action:** Write down recovery phrase → Choose passphrase → Confirm passphrase

---

### Phase 3: Privacy Mode Selection

Choose your default privacy mode (**Shadow Gradient**):

| Mode | Description | Surface Identity | Specter Identity |
|---|---|---|---|
| **Open** | Maximum visibility | ✅ Yes | ❌ No |
| **Hybrid** | Balanced | ✅ Yes | ✅ Yes |
| **Guarded** | Default-anonymous | ⚠️ Optional | ✅ Yes |
| **Fortress** | Maximum anonymity | ❌ No | ✅ Yes (via Shroud) |

**Recommendation for first-time users:** Start with **Open** or **Hybrid**. You can change modes later.

**Action:** Select mode → Confirm

---

### Phase 4: Bootstrap (Network Connection)

MURMUR connects you to the peer-to-peer network. Two methods:

#### Method A: Invitation Link (Recommended)

If someone already on MURMUR gave you an invitation link:

1. Paste the `murmur://invite/...` link into the input field
2. Press "Connect"
3. You'll bootstrap directly through their node (**warm start**)
4. You'll appear in their neighborhood on the Pulse Map

**Advantages:**
- Instant connection to someone you know
- Your first node is a familiar face
- Faster bootstrap (no DHT discovery)

#### Method B: Public DHT (Cold Start)

If you don't have an invitation:

1. Click "Connect via DHT"
2. MURMUR connects to hardcoded bootstrap nodes
3. Kademlia DHT discovery finds peers
4. You'll appear in a random region of the Pulse Map

**Note:** If the network is very small, you may see only a few nodes initially.

**Action:** Enter invitation link (if you have one) → Click "Connect" **OR** Click "Connect via DHT"

---

### Phase 5: Pulse Map Introduction

You'll see the **Pulse Map** for the first time — a force-directed graph where:
- **Glowing nodes** = users (you and your connections)
- **Edges** = relationships (connections between users)
- **Colors** = privacy modes (blue=Open, purple=Hybrid, green=Guarded, red=Fortress)

**Controls:**
- **Pan**: Click and drag (left mouse button) or arrow keys
- **Zoom**: Mouse wheel or `+`/`-` keys
- **Select node**: Single-click
- **Center on node**: Double-click

**Tutorial:** MURMUR highlights key features:
1. Your node (glowing in your mode's color)
2. Connections (edges to other nodes)
3. Navigation controls

**Action:** Follow the guided tour → Press "Continue" when ready

---

### Phase 6: Publish Your First Wave

**What is a Wave?**
- Signed, ephemeral text message (≤2048 bytes UTF-8)
- SHA-256 Proof of Work (2-5 second computation)
- Configurable TTL (default 7 days, max 30 days)
- Propagates via GossipSub across the mesh

**Steps:**

1. **Compose** — Type your message (≤2048 bytes)
   - Example: "Hello, MURMUR! Exploring the Pulse Map for the first time."
   
2. **Choose TTL** — How long should this Wave last?
   - 1 day (minimum)
   - 7 days (default)
   - 30 days (maximum)
   
3. **Compute PoW** — Click "Publish"
   - MURMUR computes SHA-256 Proof of Work (20-bit difficulty)
   - Takes 2-5 seconds on modern hardware
   - Progress bar shows computation status
   
4. **Watch Propagation** — Your Wave ripples outward!
   - Visual ripple animation from your node
   - Wave propagates through edges to connected nodes
   - You'll see "Wave published" confirmation

**Action:** Type message → Set TTL → Click "Publish" → Watch propagation animation

---

## Post-Onboarding: Exploring MURMUR

### Main Interface

After onboarding, you're in the **main MURMUR interface**:

- **Center**: Pulse Map (force-directed graph)
- **Top-right**: Status bar (peer count, network health)
- **Bottom-left**: Wave composition panel (when active)
- **Right panel**: Selected node details (when a node is selected)
- **Menu**: Press `Esc` to open

---

### Navigation Controls

| Action | Input |
|---|---|
| Pan | Left-click drag or arrow keys |
| Zoom in | Mouse wheel up or `+` key |
| Zoom out | Mouse wheel down or `-` key |
| Select node | Single left-click |
| Center on node | Double left-click |
| Deselect | Click empty space |
| Open menu | `Esc` key |
| Switch to Anonymous Layer | `Tab` key |
| Compose Wave | `Spacebar` |
| Close panel | `Esc` key |

---

### Key Actions

#### Publishing a Wave

1. Press `Spacebar` or click "New Wave" button
2. Type your message (≤2048 bytes)
3. Choose TTL (1-30 days)
4. Click "Publish" (PoW computation starts)
5. Wait 2-5 seconds for PoW completion
6. Watch ripple animation

#### Replying to a Wave

1. Select a node showing a Wave
2. Right-click → "Reply"
3. Type your reply
4. Click "Publish" (creates a Reply Wave, type 0x02)
5. Reply is threaded with the original Wave

#### Connecting with Someone

1. Select their node on the Pulse Map
2. Right-click → "Send Connection Request"
3. They receive a notification
4. If they accept, an edge forms between your nodes

#### Inviting a Friend

1. Press `Esc` → "Invite a Friend"
2. Copy the `murmur://invite/...` link **OR** show the QR code
3. Share via any channel (text, email, etc.)
4. When they accept, they'll appear in your neighborhood

---

### Creating a Specter (Anonymous Identity)

**Requirements:** Hybrid, Guarded, or Fortress mode

1. Switch to Anonymous Layer (press `Tab` key)
2. Press `Esc` → "Create Specter"
3. MURMUR generates:
   - Curve25519 keypair (anonymous identity)
   - Procedural name (e.g., "Crimson Whisper")
   - Procedural sigil (deterministic visual icon)
4. Your Specter appears as a gray node on the Anonymous Layer overlay

**Specter Actions:**
- Publish Specter Waves (type 0x04)
- Participate in mini-games
- Send Phantom Gifts (requires Resonance ≥25)
- Place Specter Marks (requires Resonance ≥50)

---

### Building Resonance (Anonymous Reputation)

**What is Resonance?**
- Locally-computed reputation metric for Anonymous Layer participation
- Unlocks mechanics at milestones (25, 50, 75, 100, 200, 500)
- Non-transferable, non-financialized

**How to Earn Resonance:**
- Publish Specter Waves (+1-5 per Wave)
- Participate in mini-games (+5-20 per game)
- Send Phantom Gifts (+2-10 per gift)
- Place Specter Marks (+3-8 per mark)
- Win mini-game tournaments (+20-50)

**Milestones:**
1. **Shade (25)** — Unlock Phantom Gifts
2. **Wraith (50)** — Unlock Specter Marks
3. **Shade-Wraith (75)** — Unlock mini-game tournaments
4. **Phantom (100)** — Unlock Masked Events
5. **Council-Eligible (200)** — Unlock Phantom Councils
6. **Abyss (500)** — Fortress-mode only

---

### Participating in Mini-Games

MURMUR has **10 anonymous mini-games**:

1. **Cipher Puzzles** — Decrypt challenges for Resonance
2. **Specter Hunt** — Find hidden Specters on the Pulse Map
3. **Territory Drift** — Claim regions via Resonance density
4. **Oracle Pools** — Collaborative prediction markets
5. **Sigil Forge** — Customize your Specter's visual identity
6. **Shadow Play** — Real-time anonymous charades
7. **Phantom Councils** — Anonymous governance (Resonance ≥200 required)
8. **Masked Events** — Temporary ultra-anonymous spaces (Resonance ≥100 required)
9. **Whisper Chains** — Multi-hop encrypted message games
10. **Echo Index** — Recursive reputation verification

**To Start a Game:**
1. Switch to Anonymous Layer (`Tab` key)
2. Press `Esc` → "Mini-Games"
3. Select a game
4. Follow on-screen instructions

See [ANONYMOUS_GAME_MECHANICS.md](../ANONYMOUS_GAME_MECHANICS.md) for detailed rules.

---

## Tips for New Users

### 1. Start with a Friend Group

MURMUR works best with **4-8 people** who know each other:
- One person launches MURMUR first
- They generate invitation links for the group
- Everyone joins via invitation (warm start)
- You all appear in the same Pulse Map region

### 2. Explore the Pulse Map

Don't stay static — **pan, zoom, click**:
- Find interesting nodes (high activity = brighter glow)
- Follow edges to discover connections
- Double-click to center on a node
- Use `Tab` to switch between Surface and Anonymous layers

### 3. Set Reasonable TTLs

Don't default to 30-day TTL for everything:
- **1 day** — Time-sensitive announcements
- **7 days** — Typical conversations (default)
- **30 days** — Important announcements, event invitations

Shorter TTLs reduce storage and improve privacy (less long-term metadata).

### 4. Build Resonance Gradually

Don't rush to Council-Eligible (200):
- Start with simple actions (Specter Waves, Cipher Puzzles)
- Unlock Phantom Gifts at 25, Marks at 50
- Play mini-games to accelerate (5-20 Resonance per game)
- Resonance decays over time (encourages sustained participation)

### 5. Experiment with Privacy Modes

Try switching modes to see the differences:
- **Open → Hybrid** — Create a Specter, see the Anonymous Layer
- **Hybrid → Guarded** — Your Surface identity becomes optional (selective reveals)
- **Guarded → Fortress** — All traffic routes through Shroud circuits (maximum anonymity)

You can switch back anytime (Settings → Privacy Mode).

### 6. Use Invitation Links

Cold-start via DHT is functional but impersonal:
- **Warm start** (invitation link) → You appear near someone you know
- **Cold start** (DHT) → You appear in a random region

Always prefer invitation links when available.

### 7. Keep Your Recovery Phrase Safe

Your 24-word BIP-39 recovery phrase is **the only way** to recover your identity:
- Write it down (pen and paper, no digital copies)
- Store in multiple secure locations (safe, safety deposit box)
- Never share it with anyone
- Test recovery once (reinstall MURMUR, recover from phrase, then reinstall again)

### 8. Monitor Peer Count

Check the status bar (top-right) for peer count:
- **<6 peers** — Mesh is sparse; Waves may propagate slowly
- **6-12 peers** — Target range; optimal gossip propagation
- **>12 peers** — No benefit; MURMUR caps at ~12 for bandwidth efficiency

If you have <6 peers for extended periods:
- Check firewall (allow port 4001)
- Use invitation links (bypasses DHT discovery)
- Enable relay fallback (automatic in v0.1.0-rc1)

---

## Troubleshooting

### MURMUR won't start

**Error:** `Display not found` (Linux)
- **Solution:** Install a display server (X11 or Wayland) or use `xvfb-run`:
  ```bash
  xvfb-run -a ./murmur
  ```

**Error:** `Permission denied` (Linux/macOS)
- **Solution:** Make the binary executable:
  ```bash
  chmod +x murmur
  ./murmur
  ```

### Can't connect to any peers

**Symptoms:** Peer count stays at 0, Pulse Map is empty

**Solutions:**
1. **Check internet connection** — Ping a public server
2. **Allow port 4001** — Check firewall (TCP/UDP)
3. **Use invitation link** — Bypasses DHT discovery
4. **Check logs** — See `~/.murmur/murmur.log` for errors

### Pulse Map rendering is slow

**Symptoms:** <30fps, choppy animation

**Solutions:**
1. **Reduce window size** — Settings → Display → Resolution
2. **Disable effects** — Settings → Effects → Off
3. **Update GPU drivers** — Especially on Linux
4. **Close other GPU-heavy apps** — Browsers, games

### I forgot my passphrase

**Unfortunately, there is no recovery** unless you saved your BIP-39 recovery phrase:

1. **If you have your 24-word recovery phrase:**
   - Reinstall MURMUR (or delete `~/.murmur/`)
   - During onboarding, choose "Recover from phrase"
   - Enter your 24 words
   - Create a new passphrase

2. **If you did NOT save your recovery phrase:**
   - Your keys are lost permanently
   - You must create a new identity
   - This is by design (self-sovereign identity)

### Waves aren't propagating

**Symptoms:** Published Waves don't reach other nodes

**Solutions:**
1. **Check peer count** — Need ≥6 peers for reliable gossip
2. **Verify PoW completion** — Check for "Wave published" confirmation
3. **Increase PoW timeout** — Settings → Content → PoW Timeout → 10 seconds
4. **Check TTL** — Very short TTLs (<1 hour) may expire before propagation

---

## Next Steps

Once you're comfortable with the basics:

1. **Read the Documentation**
   - [DESIGN_DOCUMENT.md](../DESIGN_DOCUMENT.md) — Complete specification
   - [WAVES.md](../WAVES.md) — 8 Wave types in detail
   - [RESONANCE_SYSTEM.md](../RESONANCE_SYSTEM.md) — Reputation mechanics
   - [ANONYMOUS_GAME_MECHANICS.md](../ANONYMOUS_GAME_MECHANICS.md) — Mini-game rules

2. **Explore Advanced Features**
   - Create a Specter identity (Hybrid/Guarded/Fortress mode)
   - Participate in mini-games (Cipher Puzzles, Specter Hunts)
   - Send Phantom Gifts (requires Resonance ≥25)
   - Join Phantom Councils (requires Resonance ≥200)

3. **Contribute**
   - Report bugs: [GitHub Issues](https://github.com/opd-ai/murmur/issues)
   - Request features: [GitHub Discussions](https://github.com/opd-ai/murmur/discussions)
   - Submit PRs: See [CONTRIBUTING.md](../CONTRIBUTING.md)

4. **Invite Friends**
   - Generate invitation links (Menu → "Invite a Friend")
   - Form a friend group (4-8 people)
   - Explore the network together

---

## Help & Support

- **FAQ**: [docs/FAQ.md](FAQ.md)
- **GitHub Issues**: [github.com/opd-ai/murmur/issues](https://github.com/opd-ai/murmur/issues)
- **GitHub Discussions**: [github.com/opd-ai/murmur/discussions](https://github.com/opd-ai/murmur/discussions)
- **Developer Chat**: Join the MURMUR network, find the `#dev` Wave thread

---

**Welcome to MURMUR. Join the mesh. Explore the Pulse Map. Become a Specter.**
