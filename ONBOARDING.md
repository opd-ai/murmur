# Onboarding

**Category:** Interface — First-Run Experience
**Version:** 0.4
**Status:** Draft

---

## Overview

Onboarding is the sequence of screens, interactions, and processes that a new user encounters from the moment they first launch the MURMUR application to the moment they are fully participating in the network. The onboarding experience must accomplish several things simultaneously: it must generate cryptographic identity material, connect the user to the peer-to-peer network, explain MURMUR's unusual concepts (decentralization, dual-layer architecture, anonymity modes, the Pulse Map), and make the user feel welcomed and oriented rather than overwhelmed and confused.

MURMUR is not a conventional social application. It has no server, no account creation form, no email verification, no password. Identity is a keypair. The network is a living mesh of peers. The interface is a spatial map, not a feed. Every one of these properties requires explanation, but the onboarding must not feel like a lecture. The design philosophy is to teach through action — the user learns by doing, with minimal text and maximal interaction. Explanations are layered: the onboarding provides just enough understanding to begin, and deeper comprehension develops through use.

The onboarding flow has six phases: Welcome, Identity Creation, Mode Selection, Network Bootstrap, Guided Exploration, and First Action. The entire flow takes approximately 3–5 minutes for a user who reads every screen, or under 2 minutes for a user who skips optional explanations.

---

## Phase 1: Welcome

### First Launch Screen

The first screen the user sees is a full-bleed dark background with a single animated element: a softly glowing dot at the center of the screen, pulsing gently. The dot represents the user's future node — a single point of light in an empty space. Below the dot, the application name "MURMUR" is displayed in a clean, understated typeface. Below the name, a single line of text reads: "A network that belongs to no one."

There are no login fields, no sign-up buttons, no terms-of-service checkboxes. The screen has one interactive element: a button labeled "Begin." The simplicity is deliberate — it signals that MURMUR is different from the platforms the user is accustomed to, and it creates a moment of focused attention before the onboarding sequence begins.

### Philosophy Screen

After tapping "Begin," a brief philosophy screen appears. The screen displays three short statements, presented one at a time with a gentle fade-in animation, each visible for 3 seconds before the next appears.

"No servers. The network lives on the devices of its participants."

"No accounts. Your identity is a cryptographic key that you own."

"No algorithms. You see what the network shows, shaped by topology, not by corporate interest."

After the three statements, a "Continue" button appears. This screen is skippable — a small "Skip" link in the corner allows the user to proceed directly to Identity Creation. The philosophy screen exists to set expectations and prime the user's mental model before they encounter the technical onboarding steps.

---

## Phase 2: Identity Creation

### Keypair Generation

The identity creation phase begins with a screen displaying the heading "Creating Your Identity" and a brief animated visualization of the key generation process. The visualization is abstract — a swirl of particles coalescing into a solid point of light — and runs for 2–3 seconds while the application generates the user's Ed25519 keypair in the background.

The keypair generation is instantaneous on modern hardware (under 1 millisecond), but the animation runs for a fixed duration to create a sense of ceremony. The user is witnessing the birth of their identity — the moment deserves more weight than an instant flash.

When the animation completes, the screen transitions to display the user's newly generated identity: the procedurally generated sigil (a geometric visual derived from the public key hash), the public key fingerprint (displayed as a truncated hexadecimal string, e.g., "A7F3:2B91:..."), and a prompt to set a display name.

### Display Name

The display name prompt reads: "Choose a name others will see." A text input field accepts a display name of 1–64 UTF-8 characters. The display name is optional — the user can leave it blank and proceed with only their sigil and fingerprint as identifiers. A note below the input explains: "You can change this anytime. Your sigil and fingerprint are permanent — they are derived from your cryptographic key."

If the user enters a display name, the Pulse Map node preview (a small rendering of what the user's node will look like on the map) updates in real time, showing the display name below the sigil. This live preview gives the user immediate visual feedback and introduces the concept that they will appear as a node on a map.

### Key Backup Prompt

After the display name step, a key backup screen appears. The heading reads: "Your Private Key." The screen explains, in plain language, that the private key is the user's sole proof of identity, that MURMUR does not store the key on any server, that if the key is lost, the identity is lost permanently, and that no one — including MURMUR's developers — can recover a lost key.

The screen offers two options for key backup.

"Save Key File" generates an encrypted key file (a JSON file containing the Ed25519 private key, encrypted with a user-chosen passphrase using Argon2id key derivation and ChaCha20-Poly1305 encryption). The user is prompted to choose a passphrase (with a strength indicator), and the encrypted file is saved to the device's file system or shared via the system share sheet (for mobile users who want to send it to cloud storage, email, or another device).

"Write Down Recovery Phrase" displays the private key encoded as a BIP-39-compatible mnemonic phrase (24 English words). The user is instructed to write the words on paper and store them securely. A confirmation step requires the user to re-enter 3 randomly selected words from the phrase to verify they have recorded it correctly.

A third option, "Skip for Now," is available but visually de-emphasized (gray text, smaller font). Tapping it displays a warning dialog: "If you skip backup and lose access to this device, your identity will be permanently lost. Are you sure?" The user must confirm to proceed without backup. The application stores a local flag indicating that the key has not been backed up and displays periodic reminder notifications (once per week, dismissable) until the user completes a backup.

### Identity Publication

After key backup, the application publishes the user's identity to the network. This happens automatically once the node has connected to at least one peer (which occurs in Phase 4: Network Bootstrap). The identity publication is an Identity Announcement message on the Surface Layer gossip topic containing the user's public key, display name (if set), and a PoW stamp.

The onboarding flow does not wait for identity publication to complete before proceeding — the publication happens in the background while the user continues through the remaining phases. A small status indicator in the corner of subsequent screens shows the publication status ("Publishing identity..." → "Identity published ✓").

---

## Phase 3: Mode Selection

### Mode Introduction

The mode selection screen introduces the concept of MURMUR's dual-layer architecture and the three participation modes. The screen heading reads: "Choose How You Participate."

The introduction is presented as a short animated sequence. The animation begins with the user's node glowing in warm tones on a dark background — this represents the Surface Layer. Then, a second layer fades in behind the first, rendered in cool blue-purple tones with translucent nodes and drifting particles — this represents the Anonymous Layer. The two layers overlap, coexisting in the same space but visually distinct. The animation lasts 5 seconds and plays automatically.

Below the animation, a brief text explains: "MURMUR has two layers. The Surface Layer is public — your identity is visible. The Anonymous Layer is private — you participate through an anonymous identity that cannot be linked to you."

### Mode Cards

Below the explanation, three mode cards are presented side by side (on desktop) or as a horizontally scrollable carousel (on mobile). Each card contains the mode name, a visual icon, a one-sentence description, and a bulleted list of 3–4 key properties.

**Open Mode.** Icon: a single bright circle. Description: "Participate publicly on the Surface Layer." Properties: your identity is visible to everyone; you can publish Waves, form connections, and appear on the Pulse Map; you cannot access the Anonymous Layer; simplest experience, lowest complexity.

**Hybrid Mode.** Icon: two overlapping circles, one warm and one cool. Description: "Participate on both layers with separate identities." Properties: you have a public identity on the Surface Layer and an anonymous Specter identity on the Anonymous Layer; your Specter is cryptographically unlinked from your public identity; you can access all features on both layers; your Anonymous Layer traffic is routed through Shroud relays for privacy; recommended for most users.

**Fortress Mode.** Icon: a single cool-toned circle with a faint shield outline. Description: "Participate exclusively on the Anonymous Layer." Properties: you have no public identity; your only presence is your Specter; all traffic is routed through Shroud relays; maximum anonymity; you cannot interact with the Surface Layer; advanced mode for users who prioritize privacy above all else.

Each card has a "Select" button. The currently selected card is highlighted with a glowing border.

### Mode Selection Guidance

Below the mode cards, a context-sensitive guidance panel provides additional information based on which card the user is hovering over or has selected.

For Open mode, the guidance reads: "Open mode is ideal if you want a straightforward social experience and do not need anonymity. You can upgrade to Hybrid mode later without losing your identity."

For Hybrid mode, the guidance reads: "Hybrid mode gives you the full MURMUR experience. You can participate publicly and anonymously with separate identities. Your anonymous identity is protected by cryptographic separation and network-level routing. This is the recommended mode for most users."

For Fortress mode, the guidance reads: "Fortress mode provides the strongest anonymity but limits your experience to the Anonymous Layer. You cannot form public connections or be visible on the Surface Layer. Choose this mode if anonymity is your primary concern. You cannot upgrade from Fortress mode to Open or Hybrid mode — Fortress mode does not generate a Surface Layer identity."

### Specter Identity Generation

If the user selects Hybrid or Fortress mode, the application generates a second Ed25519 keypair — the Specter keypair. The generation is accompanied by a variant of the identity creation animation: instead of particles coalescing into a warm point of light, dark blue-purple particles swirl and converge into a translucent, shimmering node. The Specter's procedurally generated sigil (derived from the Specter public key hash, using the Anonymous Layer's restricted cool-tone palette) and two-word pseudonym (derived from the Specter public key hash using the Specter wordlist) are displayed.

The screen reads: "This is your Specter — your anonymous identity on the Anonymous Layer." The Specter's pseudonym is prominently displayed (e.g., "Hollow Beacon"). Below the pseudonym, the Specter sigil is rendered at full size.

A key backup prompt for the Specter key follows the same pattern as the main identity key backup (encrypted file, mnemonic phrase, or skip). If the user is in Fortress mode, this is their only identity key and the skip option displays a stronger warning: "This is your only identity in MURMUR. If you lose this key, you permanently lose access to the network."

### Mode Confirmation

After mode selection and Specter generation (if applicable), a confirmation screen summarizes the user's chosen configuration. For Hybrid mode, the screen shows both the Surface Layer identity (display name, sigil, fingerprint) and the Specter identity (pseudonym, sigil) side by side, with a clear visual separation between the two layers. For Fortress mode, only the Specter identity is shown. For Open mode, only the Surface Layer identity is shown.

The user confirms by tapping "Enter the Network." This transitions to Phase 4.

---

## Phase 4: Network Bootstrap

### Connection Establishment

The network bootstrap phase connects the user's node to the MURMUR peer-to-peer network. The screen displays a visualization of the connection process: the user's node (the glowing dot from the Welcome screen) sits at the center, and new peer connections appear as lines extending outward to other dots that fade in from the darkness.

The bootstrap sequence is as follows. The application selects 2–3 bootstrap nodes from the hardcoded list and initiates connections. As each connection is established, a new dot appears on the visualization with a line connecting it to the user's node, accompanied by a brief pulse animation. The application then runs the peer exchange protocol, discovering additional peers from the bootstrap nodes' peer lists. New connections appear on the visualization as additional dots and lines.

The target is to establish at least 5 peer connections before proceeding. On a typical residential internet connection, this takes 3–10 seconds. A progress indicator below the visualization shows the connection count: "Connected to 1 peer... 3 peers... 5 peers."

If the application cannot connect to any bootstrap nodes within 30 seconds (network issues, all bootstrap nodes offline, firewall blocking), the screen displays a troubleshooting message: "Unable to connect to the network. Check your internet connection, or enter a peer address manually." A text input field allows the user to enter a peer's multiaddress (obtainable from a friend or community resource). A "Retry" button re-attempts bootstrap node connections.

### Initial Gossip Subscription

Once connected to at least 5 peers, the application subscribes to the appropriate gossip topics based on the user's mode. For Open mode, the application subscribes to Surface Layer topics. For Hybrid mode, the application subscribes to both Surface Layer and Anonymous Layer topics. For Fortress mode, the application subscribes to Anonymous Layer topics only.

The gossip subscription is not displayed to the user — it happens in the background. The user sees the connection visualization transition smoothly into the first glimpse of the Pulse Map: the dots representing peers begin to move into their force-directed layout positions, additional nodes appear as gossip data arrives, and the dark background gradually takes on the Pulse Map's atmospheric particle field and noise texture.

### Shroud Circuit Establishment

For Hybrid and Fortress mode users, the application constructs an initial Shroud circuit during the bootstrap phase. The application discovers available Shroud Nodes from Beacon Waves received through gossip, selects 3 nodes for the circuit, and performs the key exchange sequence. Circuit establishment typically takes 2–5 seconds after Shroud Node discovery.

A subtle indicator appears during this process: a small shield icon in the corner of the screen with a spinning animation and the label "Establishing anonymous route..." When the circuit is established, the icon solidifies and the label changes to "Anonymous route active ✓." This indicator reassures Hybrid and Fortress users that their Anonymous Layer traffic is being routed through the Shroud Network.

If no Shroud Nodes are available (possible in a very small or new network), the application warns the user: "No anonymous relay nodes are currently available. Your Anonymous Layer traffic will be published directly until relay nodes are found. This reduces your anonymity." The application continues to search for Shroud Nodes in the background and establishes a circuit as soon as nodes become available.

---

## Phase 5: Guided Exploration

### Pulse Map Introduction

The guided exploration phase transitions the user into the live Pulse Map with an interactive tutorial overlay. The overlay is a series of contextual tooltips and highlight regions that draw the user's attention to specific elements of the Pulse Map and explain their meaning.

The first tooltip appears near the user's own node, which is positioned at the center of the viewport in ego-centric view. The tooltip reads: "This is you. Your node on the network. Everything radiates outward from here." The user's node is highlighted with a glowing ring animation. Tapping the tooltip or the node advances to the next step.

### Node Explanation

The second tooltip highlights a nearby peer node (one of the nodes discovered during bootstrap). The tooltip reads: "Each dot is a person. The map shows everyone on the network, arranged by how they are connected." The highlighted node displays its label (display name or fingerprint) and sigil. If no suitable peer node is nearby (very sparse network), the tooltip instead points to a cluster of nodes in the distance and explains: "Other participants appear as dots. The closer they are, the more connected they are to each other."

### Connection Explanation

The third tooltip highlights a connection line between two nodes. The tooltip reads: "Lines between nodes are connections — social bonds. Waves and messages travel along these connections." A brief pulse animation plays along the highlighted connection to illustrate the concept of information flowing through the network.

### Wave Publishing Tutorial

The fourth tooltip introduces Wave publishing. The tooltip appears near the user's node and reads: "Ready to say something? Publish a Wave — it will ripple outward through the network." A compose button is highlighted. When the user taps the compose button, a compose panel slides in (identical to the standard Wave compose panel) with a pre-filled suggestion: "Hello, MURMUR." The user can edit the text or use the suggestion. When the user publishes the Wave, the standard publication animation plays (node pulse, radial shockwave), and the next tooltip appears.

The fifth tooltip appears near the published Wave's content card and reads: "Your Wave is now propagating through the network. Anyone connected to your neighborhood will see it." If any peer amplifies the Wave (unlikely during first-run but possible), an additional tooltip explains amplification.

### Layer Introduction (Hybrid and Fortress Only)

For Hybrid-mode users, an additional tutorial sequence introduces the Anonymous Layer. After the Wave publishing tutorial, the layer blend slider appears with a tooltip: "MURMUR has two layers. Slide to reveal the Anonymous Layer." As the user drags the slider, Specter nodes and Anonymous Layer visual elements (cool-toned nodes, particle emissions, translucent styling) fade in. A tooltip near a visible Specter node reads: "These are Specters — anonymous identities. No one knows who they really are."

A second tooltip highlights the user's own Specter node (which appears near their Surface Layer node in the blended view): "This is your Specter. Your anonymous self. On the Anonymous Layer, you appear as this identity — completely unlinked from your public self."

For Fortress-mode users, the layer introduction is simpler: the Pulse Map already shows only the Anonymous Layer, so the tooltip reads: "You are on the Anonymous Layer. Everyone here is a Specter — an anonymous identity. No one can link your Specter to your real identity."

### Anonymous Mechanics Preview (Hybrid and Fortress Only)

A brief tooltip sequence previews the key anonymous mechanics without requiring the user to interact with them (most mechanics are gated by Specter Resonance milestones the user has not yet reached).

A tooltip near the user's Specter node reads: "As you participate, your Specter earns Resonance — a measure of your anonymous presence. Higher Resonance unlocks new abilities." A small Resonance meter is highlighted in the Specter node's detail panel, currently showing Resonance 0.

A second tooltip reads: "You will unlock Phantom Gifts, Specter Duels, Masked Events, and more as your Resonance grows. Explore the Anonymous Layer to begin."

### Connection Suggestion

The final onboarding step suggests initial connections. The screen displays a panel titled "Find Your People" with three options for forming initial connections.

"Invite a Friend" generates a connection invitation — a compact string (encoded multiaddress and public key fingerprint) that the user can share via any communication channel (text message, email, QR code). When the friend installs MURMUR and enters the invitation string, a mutual connection is formed.

"Browse Nearby" (available only if mDNS discovery has found local peers) shows a list of MURMUR nodes on the same local network, allowing the user to send connection requests to people physically nearby.

"Explore the Map" dismisses the connection suggestion panel and releases the user into the full Pulse Map, where they can navigate, discover nodes, and send connection requests organically.

After the user selects any option (or dismisses the panel), the onboarding overlay fades out and the user enters Phase 6.

---

## Phase 6: First Action

The onboarding concludes with a prompt to take a first action: publish a Wave, form a connection, or explore the map freely.

### First Wave Prompt

The screen displays a compose panel titled "Send Your First Wave" with a suggested message: "Hello, MURMUR" pre-filled but fully editable. The user can modify or clear this text.

Tapping "Publish" triggers the Proof of Work computation, accompanied by a brief animation: particles swirling around the compose panel with text "Your message is being sealed..." The PoW typically completes in 2–5 seconds on modern hardware.

Once the PoW completes, the Wave is broadcast to the network. The Pulse Map visualization shows the propagation ripple expanding from the user's node — concentric rings of light spreading outward through the mesh, fading as they reach distant peers. This visual confirms that the user's message is traveling through the network.

### Tutorial Completion

After the first action (publishing a Wave, forming a connection, or explicitly choosing "Explore freely"), the onboarding tutorial overlay fully dismisses and the user has complete control of the interface.

A small, dismissable hint system remains available for the next few sessions, providing contextual tooltips when the user encounters unfamiliar UI elements for the first time. These hints can be permanently disabled in settings.

A "Help" button remains accessible in the viewport controls, allowing the user to reopen tutorial tooltips on demand at any time.

---

## Post-Onboarding Nudges

### First-Week Engagement

During the user's first 7 days, the application displays contextual nudges — brief, non-intrusive suggestions that encourage exploration of features the user has not yet tried. Nudges appear as small tooltip bubbles near relevant UI elements, triggered by context.

If the user has not published a second Wave within 24 hours of their first, a nudge appears near the compose button: "Your first Wave traveled the network. Send another — the more you publish, the more your node comes alive."

If the user has not formed any connections within 48 hours, a nudge appears near a nearby node on the Pulse Map: "Tap a node to see who they are. Send a connection request to start building your neighborhood."

If the user is in Hybrid mode and has not visited the Anonymous Layer within 72 hours, a nudge appears near the layer blend slider: "Curious about the Anonymous Layer? Slide to explore."

If the user's Specter reaches a Resonance milestone (25, 50, 100), a celebratory nudge appears: "Your Specter reached [milestone name]. New abilities unlocked." The nudge links to a brief explanation of the newly available mechanics.

### Nudge Frequency and Dismissal

Nudges appear at most once per day to avoid irritation. Each nudge type appears at most once — dismissed nudges do not recur. After 7 days, all contextual nudges cease regardless of whether the user has engaged with them. The user can disable nudges entirely in settings.

---

## Returning User Experience

### Existing Identity Detection

When the application launches and detects an existing identity (private key present in local storage), the onboarding flow is skipped entirely. The application proceeds directly to network bootstrap (connecting to peers, subscribing to gossip topics, establishing Shroud circuits if applicable) and then to the Pulse Map.

The returning user experience begins with a brief loading screen showing the user's sigil and display name (or Specter pseudonym for Fortress mode) centered on the dark background, with a progress indicator showing peer connection status. This loading screen typically displays for 2–5 seconds while the network bootstrap completes.

### Identity Recovery

If the application launches with no existing identity (fresh install on a new device, or after application data was cleared), the first screen offers two paths: "Create New Identity" (which enters the standard onboarding flow) and "Restore Existing Identity" (which enters the recovery flow).

The recovery flow accepts the identity material in two formats. "Import Key File" allows the user to select an encrypted key file (produced during the key backup step of onboarding). The user provides the passphrase, and the application decrypts and imports the private key. "Enter Recovery Phrase" allows the user to type the 24-word mnemonic phrase. The application derives the private key from the mnemonic and imports it.

After import, the application verifies the key by deriving the public key, computing the fingerprint, and displaying the sigil and display name associated with the key. The user confirms that the displayed identity is correct. The application then publishes a fresh identity announcement (to update the network with the node's new network address) and proceeds to the Pulse Map.

For Hybrid-mode users, the recovery flow handles two keys: the main identity key and the Specter key. Both keys are backed up and recovered independently. The recovery screen prompts for both keys in sequence: "First, restore your public identity" followed by "Now, restore your Specter identity." If the user has only one key (lost the other), they can recover the available identity and generate a new keypair for the missing one, with the understanding that the new keypair creates a new identity on the lost layer (new Specter or new main identity with no history or connections from the previous one).

### Offline Recovery

Identity recovery does not require network connectivity. The private key is derived entirely from local data (key file or mnemonic phrase). The user can recover their identity on an air-gapped device if desired, then connect to the network later. This property is important for security-conscious users who want to verify their recovery material without exposing the key to a networked environment.

---

## Edge Cases and Error Handling

### Extremely Small Network

If the user is among the first participants in a new MURMUR network (or if the network has very few active nodes), the onboarding experience adapts. The bootstrap phase may find only 1–2 peers instead of the target 5. The Pulse Map will be sparse, with only a handful of nodes visible. The guided exploration tooltips will reference the sparse state: "The network is small right now. As more people join, the map will grow."

The Shroud circuit establishment may fail if no Shroud Nodes are available. The application warns the user and operates in reduced-privacy mode until Shroud Nodes appear. The user can volunteer their own node as a Shroud Node (if in Hybrid+ mode) to bootstrap the Shroud Network, though this option is not presented during onboarding (it is available in settings).

### Network Unavailability

If the device has no internet connectivity at launch, the onboarding flow proceeds through Identity Creation and Mode Selection without interruption (these phases do not require network access). The Network Bootstrap phase displays a waiting screen with the message: "Waiting for network connection. Your identity has been created and is ready." The application monitors connectivity and automatically proceeds when a connection becomes available.

If connectivity is intermittent (connects and disconnects repeatedly), the application tolerates brief interruptions gracefully — peer connections are maintained with keepalive messages, and brief disconnections (under 30 seconds) do not reset the bootstrap process.

### Very Slow Devices

On devices with limited CPU (older smartphones, low-power tablets), the PoW computation for the identity announcement and first Wave may take longer than the typical 0.5 seconds. The application detects slow PoW computation and adjusts the UI: the publish button shows a progress indicator during PoW computation, and a tooltip explains: "Preparing your message. This may take a moment on this device." The onboarding flow does not stall — PoW computation runs in a background thread, and the user can continue exploring the Pulse Map while their first Wave is being stamped.

### Interrupted Onboarding

If the user closes the application mid-onboarding, the application saves the onboarding state. When the application is next launched, it resumes from the last completed phase. If the user completed Identity Creation but not Mode Selection, the application skips directly to Mode Selection. If the user completed Mode Selection but not Network Bootstrap, the application proceeds to bootstrap. The user is never forced to repeat a completed phase.

If the application crashes during keypair generation (before the key is persisted to storage), the key is lost and a new keypair is generated on the next launch. This is acceptable because the key had not yet been backed up or published — no identity was established.

---

## Invitation Flow

### Generating an Invitation

An existing MURMUR user can generate an invitation to share with a potential new user. The invitation is a compact data structure containing the inviter's Peer ID (for bootstrap purposes — the new user can connect directly to the inviter rather than relying on bootstrap nodes), the inviter's public key fingerprint (for connection verification), and an optional welcome message (up to 128 characters).

The invitation is encoded as a URL-safe Base64 string, prefixed with `murmur://invite/`. The encoded invitation is approximately 100–150 characters long, compact enough to share via text message, social media post, QR code, or email.

The inviter generates the invitation from the Pulse Map's menu: "Invite a Friend" produces the invitation string and offers sharing options (copy to clipboard, share via system share sheet, display as QR code).

### Accepting an Invitation

A new user who receives an invitation string can enter it during onboarding (in the "Find Your People" step of the guided exploration phase) or at any time via the Pulse Map's search bar (which accepts invitation strings as input).

When an invitation is accepted, the application connects to the inviter's Peer ID as an additional bootstrap peer (supplementing the hardcoded bootstrap nodes). Once connected, the application automatically sends a connection request to the inviter. The inviter receives the connection request and can accept it, forming the new user's first social connection.

If the invitation is accepted before the new user has installed MURMUR (for example, a `murmur://` URL clicked on a device without the application), the URL scheme triggers an app store redirect or a landing page with installation instructions. After installation, the invitation is preserved (via OS-level deep linking or clipboard detection on first launch) and processed during onboarding.

### Invitation Expiration

Invitations do not expire at the protocol level — the encoded data (Peer ID, fingerprint, welcome message) remains valid as long as the inviter's node is active. However, if the inviter's node is offline when the new user attempts to connect, the bootstrap connection will fail. The application falls back to standard bootstrap nodes in this case, and the new user can send the connection request later when both nodes are online.

---

## Accessibility During Onboarding

### Screen Reader Support

All onboarding screens are fully accessible to screen readers. Every visual element (sigil, node visualization, animation, Pulse Map preview) has an associated accessibility label describing its meaning. The animated sequences (key generation, network bootstrap, Pulse Map transition) have text alternatives that describe what is happening.

The key generation animation is described as: "Generating your cryptographic identity. A unique visual symbol is being created from your key." The network bootstrap visualization is described as: "Connecting to the network. Currently connected to [n] peers." The Pulse Map transition is described as: "Entering the network map. Your node is at the center, surrounded by [n] other participants."

### Reduced Motion

Users with the system-level "reduce motion" accessibility setting enabled see simplified animations during onboarding. The key generation swirl is replaced by a simple fade-in. The network bootstrap visualization uses static dots instead of animated connections. The Pulse Map transition is a cross-fade rather than a spatial zoom. All functionality is preserved; only the animation complexity is reduced.

### Large Text and High Contrast

Onboarding screens respect the system-level text size setting. All text elements scale proportionally, and layouts reflow to accommodate larger text without truncation or overlap. In high-contrast mode (either system-level or MURMUR's built-in high-contrast setting), onboarding screens use maximum-contrast color pairs (white text on true black background) and high-visibility button styles (thick borders, bold labels).

### Keyboard Navigation

On desktop platforms, the entire onboarding flow is navigable by keyboard. Tab moves focus between interactive elements. Enter activates the focused element. Escape dismisses tooltips and optional panels. Arrow keys navigate the mode selection cards and the connection suggestion options. Focus indicators (visible outlines on focused elements) are present on all interactive elements.

---

## Telemetry and Privacy During Onboarding

### No Telemetry

MURMUR does not collect any telemetry during onboarding or at any other time. No analytics events are sent. No usage data is transmitted. No crash reports are filed. The application makes no network requests other than peer-to-peer MURMUR protocol traffic. There are no tracking pixels, no fingerprinting scripts, and no third-party analytics SDKs.

This zero-telemetry policy is a core design commitment consistent with MURMUR's privacy architecture. The developers receive no data about how many users complete onboarding, where they drop off, which mode they select, or how they interact with the tutorial. Product improvement relies on community feedback, open-source issue reports, and developer testing.

### Local-Only State

All onboarding state (completed phases, selected mode, generated keys, backup status, nudge history) is stored locally on the user's device. No onboarding state is transmitted to any peer or server. If the user uninstalls the application and reinstalls it, the onboarding state is lost (along with the identity keys, unless backed up).

---

## Design Principles

The onboarding experience is guided by five design principles.

**Ceremony over efficiency.** Identity creation is a meaningful moment. The animations, the pacing, the single-action screens — all of these create a sense of occasion. The user is not filling out a form; they are creating a cryptographic identity that will represent them in a decentralized network. The onboarding respects the gravity of this act.

**Action over explanation.** The user learns MURMUR by using MURMUR. The tutorial teaches Wave publishing by having the user publish a Wave. It teaches navigation by having the user navigate. It teaches the layer system by having the user slide the layer blend control. Text explanations are minimal and serve only to give the user enough context to understand what they are about to do.

**Progressive disclosure.** The onboarding reveals complexity gradually. Phase 1 shows a single dot. Phase 2 shows the user's identity. Phase 3 introduces layers and modes. Phase 4 shows the network. Phase 5 shows the full Pulse Map with all its visual richness. At no point is the user confronted with the full system all at once. Each phase adds one layer of understanding.

**Safety by default.** The default mode recommendation (Hybrid) provides a strong balance of functionality and privacy. The key backup prompts are persistent and difficult to skip. The Shroud circuit indicator is visible during and after onboarding, reassuring the user that their Anonymous Layer traffic is protected. The application makes safe choices on the user's behalf unless the user explicitly overrides them.

**Graceful degradation.** The onboarding handles every edge case — no network, slow device, sparse network, interrupted flow — without crashing, stalling, or confusing the user. Error states are communicated in plain language with actionable next steps. The user is never left in a dead end.