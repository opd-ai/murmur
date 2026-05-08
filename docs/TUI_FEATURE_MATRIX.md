# TUI Feature Matrix (Phase 0 Discovery)

This matrix inventories user-facing behavior from current Ebitengine paths and authoritative docs, and defines the TUI parity strategy for `cmd/murmur-tui`.

Status legend: `todo` (not started), `done` (implemented), `deferred` (explicitly omitted with rationale).

## Pulse Map

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| PM-001 | Pulse Map | pkg/pulsemap/game.go | Main network visualization canvas | Main TUI viewport showing graph snapshot and focused node | P0 | todo |
| PM-002 | Pulse Map | pkg/pulsemap/layout/engine.go | Force-directed layout updates node positions | Reuse layout engine output; render node grid with ANSI dots/glyphs | P0 | todo |
| PM-003 | Pulse Map | pkg/pulsemap/interaction/input.go | Camera pan in world space | `h/j/k/l` + arrows pan camera offsets | P0 | todo |
| PM-004 | Pulse Map | pkg/pulsemap/game.go (wheel handling) | Mouse wheel zoom in/out centered on cursor | `+/-`, wheel, and PgUp/PgDn zoom around focus | P0 | todo |
| PM-005 | Pulse Map | pkg/pulsemap/interaction/input.go | Min/max zoom clamping | Same clamp, numeric zoom indicator in status bar | P0 | todo |
| PM-006 | Pulse Map | pkg/pulsemap/game.go (left-click) | Click selects node | Enter/click selects focused node and opens detail pane | P0 | todo |
| PM-007 | Pulse Map | pkg/pulsemap/game.go (drag state) | Dragging pans camera | Mouse drag pans; keyboard pan fallback | P0 | todo |
| PM-008 | Pulse Map | pkg/pulsemap/game.go (touch tap/double-tap) | Tap select, double-tap center/zoom | Mouse double-click + `z` for center/zoom focus action | P1 | todo |
| PM-009 | Pulse Map | pkg/pulsemap/game.go (long-press radial) | Long-press opens context menu | `m`/right-click opens action menu for selected node | P1 | todo |
| PM-010 | Pulse Map | pkg/pulsemap/game.go (Home/H) | Recenter to ego node | `h` (when not in vim-pan mode via modifier) and Home recenter command | P0 | todo |
| PM-011 | Pulse Map | pkg/pulsemap/game.go (N fit network) | Fit full network in viewport | `n` fits bounds in terminal viewport | P1 | todo |
| PM-012 | Pulse Map | pkg/pulsemap/game.go + overlays/minimap.go | Minimap overlay for orientation | Right-side minimap ASCII box with viewport marker | P1 | todo |
| PM-013 | Pulse Map | pkg/pulsemap/game.go + settings.go | Macro/Meso/Micro viewport controls | `1/2/3` preset zoom levels and indicator | P1 | todo |
| PM-014 | Pulse Map | pkg/pulsemap/game.go | Search bar and select result center | `/` search mode with result list + jump | P0 | todo |
| PM-015 | Pulse Map | pkg/pulsemap/game.go | Node detail slide-in panel | Side pane with identity, fingerprint, activity, actions | P0 | todo |
| PM-016 | Pulse Map | pkg/pulsemap/game.go | Toast notifications after async operations | Top transient message line with timeout | P1 | todo |
| PM-017 | Pulse Map | pkg/pulsemap/game.go | Bookmark add/remove/jump | Bookmark list panel; hotkeys preserved | P1 | todo |
| PM-018 | Pulse Map | pkg/pulsemap/rendering/draw.go | Edge rendering between nodes | Unicode line approximation with density fallback | P1 | todo |
| PM-019 | Pulse Map | pkg/pulsemap/rendering/colors.go | Color-coded node states | Lipgloss style palette mapped to mode/activity | P1 | todo |
| PM-020 | Pulse Map | pkg/pulsemap/rendering/effects/visual.go | Glow/pulse animations | ANSI color pulsing + spinner indicators | P2 | todo |
| PM-021 | Pulse Map | pkg/pulsemap/rendering/effects/ripples.go | Ripple effects on activity | Temporal glyph ring animation around focused node | P2 | todo |
| PM-022 | Pulse Map | pkg/pulsemap/rendering/effects/blur.go | Blur/composite post-processing | Visual-only, omitted; replaced by contrast styles | P2 | deferred |
| PM-023 | Pulse Map | pkg/pulsemap/rendering/effects/gpu_particles.go | GPU particles around events | CPU-safe text particle counter and spark glyphs | P2 | todo |
| PM-024 | Pulse Map | pkg/pulsemap/rendering/background.go | Dynamic background gradient | Static terminal theme blocks per mode | P2 | todo |

## Identity

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| ID-001 | Identity | pkg/onboarding/screens/identity.go | Generate keypair in onboarding | Guided identity creation step in TUI onboarding | P0 | todo |
| ID-002 | Identity | pkg/identity/keys/* + screens/recovery_screen.go | BIP-39 recovery path | Recovery wizard with mnemonic input + validation | P0 | todo |
| ID-003 | Identity | pkg/identity/sigils/generator.go | 64x64 sigil rendering | ANSI block sigil renderer in detail/onboarding panes | P0 | todo |
| ID-004 | Identity | pkg/identity/modes/state.go | Open/Hybrid/Guarded/Fortress transitions | Mode selector with transition guard/cooldown feedback | P0 | todo |
| ID-005 | Identity | pkg/identity/modes/state.go | Cooldown and invalid transition errors | Inline validation and countdown timer | P1 | todo |
| ID-006 | Identity | pkg/identity/declarations/* | Display name + declaration state | Editable profile form + declaration status row | P1 | todo |
| ID-007 | Identity | pkg/onboarding/screens/identity.go | Identity fingerprint display | Fingerprint shown in onboarding/profile panel | P1 | todo |
| ID-008 | Identity | pkg/identity/modes/state.go | Traffic padding state in guarded/fortress | Status badge in privacy panel | P1 | todo |
| ID-009 | Identity | pkg/onboarding/screens/completion_screen.go | Invite code generation summary | Show invite code and copy helper text | P1 | todo |
| ID-010 | Identity | pkg/identity/sigils/generator.go (GenerateSpecter/GenerateMaskedEvent) | Distinct specter/masked sigils | Alternate ANSI themes for specter/masked sigils | P1 | todo |

## Waves

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| WV-001 | Waves | pkg/pulsemap/game.go (compose panel) | Compose Wave modal panel | Dedicated compose view with multiline editor | P0 | todo |
| WV-002 | Waves | pkg/content/waves/types.go | Surface Wave type (0x01) | Wave type selector option | P0 | todo |
| WV-003 | Waves | pkg/content/waves/types.go | Reply Wave type (0x02) | Reply action in thread/detail view | P0 | todo |
| WV-004 | Waves | pkg/content/waves/veiled.go | Veiled Wave type (0x03) | Veiled compose mode with recipients field | P1 | todo |
| WV-005 | Waves | pkg/content/waves/types.go | Specter Wave type (0x04) | Specter compose mode in anonymous context | P0 | todo |
| WV-006 | Waves | pkg/content/waves/sigil.go | Sigil Wave type (0x05) | Sigil-update compose action in identity view | P1 | todo |
| WV-007 | Waves | pkg/content/waves/abyssal.go | Abyssal Wave type (0x06) | Fortress-only compose option with warning | P1 | todo |
| WV-008 | Waves | pkg/content/waves/masked.go | Masked Wave type (0x07) | Masked compose mode with visibility note | P1 | todo |
| WV-009 | Waves | pkg/content/waves/beacon.go | Beacon Wave type (0x08) | Coordination signal compose template | P1 | todo |
| WV-010 | Waves | pkg/content/pow/* | PoW compute and difficulty indicators | Non-blocking spinner + difficulty/nonce/status text | P0 | todo |
| WV-011 | Waves | pkg/content/waves/types.go (TTL) | TTL and expiration handling | TTL picker with expiry countdown label | P0 | todo |
| WV-012 | Waves | pkg/content/threads/* | Thread view/reconstruction | Thread panel with parent/reply indentation | P0 | todo |
| WV-013 | Waves | pkg/content/waves/amplification.go | Amplify action | Amplify key/action in thread list | P1 | todo |
| WV-014 | Waves | pkg/onboarding/tutorials/guide.go | First-wave guidance prompt | Contextual first-wave helper banner | P1 | todo |

## Anonymous Layer

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| AN-001 | Anonymous Layer | pkg/anonymous/specters/identity.go | Specter creation | Specter create flow with generated name/sigil preview | P0 | todo |
| AN-002 | Anonymous Layer | pkg/anonymous/specters/connection.go | Switch between Specters | Specter switcher list with active badge | P0 | todo |
| AN-003 | Anonymous Layer | pkg/anonymous/shroud/circuit.go | Shroud circuit build status | Circuit health widget (primary/backup/age) | P0 | todo |
| AN-004 | Anonymous Layer | pkg/anonymous/shroud/advertisement.go | Relay discovery | Relay table with last-seen and quality | P1 | todo |
| AN-005 | Anonymous Layer | pkg/anonymous/shroud/whisper.go | Whisper routing state | Whisper send status and route summary | P1 | todo |
| AN-006 | Anonymous Layer | pkg/anonymous/resonance/score.go + docs/RESONANCE_SYSTEM.md | Resonance score meter | Progress bar + numeric score + trend glyph | P0 | todo |
| AN-007 | Anonymous Layer | docs/RESONANCE_SYSTEM.md | Resonance milestones and unlocks | Milestone ladder list with unlocked markers | P0 | todo |
| AN-008 | Anonymous Layer | pkg/anonymous/resonance/echo_index.go | Echo Index display | Dedicated metric row in resonance panel | P1 | todo |
| AN-009 | Anonymous Layer | pkg/anonymous/mechanics/gifts/gifts.go | Phantom Gifts | Gift send flow from node/action menus | P0 | todo |
| AN-010 | Anonymous Layer | pkg/anonymous/mechanics/marks/marks.go | Specter Marks | Mark placement form + map overlay annotations | P0 | todo |
| AN-011 | Anonymous Layer | pkg/anonymous/mechanics/puzzles/puzzles.go | Cipher Puzzles | Puzzle menu + active challenge list | P1 | todo |
| AN-012 | Anonymous Layer | pkg/anonymous/mechanics/hunts/hunts.go | Specter Hunts | Hunt board and join/track actions | P1 | todo |
| AN-013 | Anonymous Layer | pkg/anonymous/mechanics/territory/territory.go | Territory Drift | Territory status table and influence bars | P1 | todo |
| AN-014 | Anonymous Layer | pkg/anonymous/mechanics/oracle/oracle.go | Oracle Pools | Oracle pool list and contribution actions | P1 | todo |
| AN-015 | Anonymous Layer | pkg/anonymous/mechanics/forge/forge.go | Sigil Forge | Forge menu and craft progress indicators | P1 | todo |
| AN-016 | Anonymous Layer | pkg/anonymous/mechanics/shadowplay/shadowplay.go | Shadow Play | Matchmaking/status card with session state | P1 | todo |
| AN-017 | Anonymous Layer | pkg/anonymous/mechanics/councils/councils.go | Phantom Councils | Council eligibility and session roster | P1 | todo |
| AN-018 | Anonymous Layer | pkg/anonymous/mechanics/sparks/sparks.go | Sparks mini-game | Sparks quick-action panel | P2 | todo |
| AN-019 | Anonymous Layer | pkg/anonymous/mechanics/pulse_beats.go | Pulse Beats mechanics | Event ticker panel for pulse beat events | P2 | todo |
| AN-020 | Anonymous Layer | pkg/pulsemap/overlays/* | Anonymous mechanics overlays on map | Overlay toggles with textual legends | P1 | todo |

## Onboarding

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| OB-001 | Onboarding | pkg/onboarding/flow/controller.go | Phase 1 Welcome | Welcome screen with progress indicator | P0 | todo |
| OB-002 | Onboarding | pkg/onboarding/flow/controller.go | Phase 2 Identity Creation | Identity step (name + keypair + sigil) | P0 | todo |
| OB-003 | Onboarding | pkg/onboarding/flow/controller.go | Phase 3 Mode Selection | Mode selection carousel/list | P0 | todo |
| OB-004 | Onboarding | pkg/onboarding/flow/controller.go | Phase 4 Network Bootstrap | Bootstrap status and retries | P0 | todo |
| OB-005 | Onboarding | pkg/onboarding/flow/controller.go | Phase 5 Guided Exploration | Guided map tutorial steps in TUI | P0 | todo |
| OB-006 | Onboarding | pkg/onboarding/flow/controller.go | Phase 6 First Wave | First wave guided compose | P0 | todo |
| OB-007 | Onboarding | pkg/onboarding/flow/controller.go | Skip and resume state | Persist and resume onboarding progress | P1 | todo |
| OB-008 | Onboarding | pkg/onboarding/tutorials/guide.go | Contextual hints manager | Bottom hint bar with dismiss/ack actions | P1 | todo |
| OB-009 | Onboarding | pkg/onboarding/bootstrap/network.go | Bootstrap progress detail | Peer count, elapsed/remaining, status message | P0 | todo |
| OB-010 | Onboarding | pkg/onboarding/bootstrap/invitation.go | Invitation warm-start flow | Invitation-aware bootstrap step indicator | P1 | todo |
| OB-011 | Onboarding | pkg/onboarding/screens/recovery_screen.go | Recovery onboarding branch | Mnemonic recovery branch in onboarding menu | P1 | todo |
| OB-012 | Onboarding | pkg/onboarding/screens/completion_screen.go | Completion summary and invite code | Completion summary pane + invite code view | P1 | todo |
| OB-013 | Onboarding | pkg/app/nudges.go | First-week nudges via EventNudge | Nudge center panel and startup nudge toast | P0 | todo |
| OB-014 | Onboarding | pkg/onboarding/screens/returning_screen.go | Returning-user continue screen | Lightweight “welcome back” terminal view | P1 | todo |

## Networking

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| NW-001 | Networking | pkg/networking/transport/* | Host transport runtime state | Transport panel (noise/quic/tcp and mode) | P1 | todo |
| NW-002 | Networking | pkg/networking/gossip/* | Gossip topic joins and throughput | Topic status table with publish/recv counters | P0 | todo |
| NW-003 | Networking | pkg/networking/discovery/* | DHT bootstrap/discovery state | DHT panel with routing table stats | P0 | todo |
| NW-004 | Networking | pkg/networking/mesh/* | Mesh health and target peers | Mesh health badge + peer range indicator | P0 | todo |
| NW-005 | Networking | pkg/networking/relay/* | NAT traversal / relay fallback | Relay/NAT diagnostics line | P1 | todo |
| NW-006 | Networking | pkg/networking/health/* + pkg/app/murmur.go | Health endpoint status | Health endpoint status row in settings/network | P1 | todo |
| NW-007 | Networking | pkg/networking/priority/* | Priority/rate-limit state | Rate-limit and queue-depth indicators | P0 | todo |
| NW-008 | Networking | pkg/app/eventbus.go | Peer connected/disconnected events | Live peer activity feed in sidebar | P1 | todo |
| NW-009 | Networking | docs/NETWORK_ARCHITECTURE.md | Canonical topics/protocol IDs visibility | Protocol/topic reference help overlay | P2 | todo |

## Settings

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| ST-001 | Settings | pkg/pulsemap/game.go (Ctrl+,) | Toggle settings panel | `Ctrl+,` toggles settings modal | P0 | todo |
| ST-002 | Settings | pkg/config/* | User configuration surface | Config editor/list with validation feedback | P1 | todo |
| ST-003 | Settings | pkg/pulsemap/settings.go | Overlay toggles | Checkbox list for overlays/features | P1 | todo |
| ST-004 | Settings | assets/themes/* | Theme hooks and palette usage | Theme selector (dark/light/high-contrast terminal styles) | P2 | todo |
| ST-005 | Settings | assets/wordlists/* | Procedural name source visibility | Wordlist source and regenerate action for names | P2 | todo |
| ST-006 | Settings | pkg/identity/modes/state.go + game mode wiring | Privacy mode setting from panel | In-panel mode selector with immediate apply | P0 | todo |

## Cross-Layer Visibility & Event Surfaces

| Feature ID | Domain | Source Location | Ebitengine Behavior | TUI Strategy | Priority (P0/P1/P2) | Status |
|---|---|---|---|---|---|---|
| CL-001 | Cross-Layer Visibility | pkg/pulsemap/overlays/layer.go | Surface/Anonymous blend slider | Blend ratio selector and split-pane emphasis | P0 | todo |
| CL-002 | Cross-Layer Visibility | pkg/pulsemap/game.go (applyModeToOverlays) | Fortress forces anonymous visibility | Enforce fortress-only anonymous views in TUI | P0 | todo |
| CL-003 | Cross-Layer Visibility | pkg/pulsemap/overlays/heatmap.go | Activity heatmap overlay | Intensity map using shaded character density | P1 | todo |
| CL-004 | Cross-Layer Visibility | pkg/pulsemap/overlays/marks.go | Marks overlay visualization | Inline mark badges on nodes/list rows | P1 | todo |
| CL-005 | Cross-Layer Visibility | pkg/pulsemap/overlays/gift.go | Gift overlay indicator | Gift indicator glyph in node detail/map | P1 | todo |
| CL-006 | Cross-Layer Visibility | pkg/pulsemap/overlays/echoindex.go | Echo Index overlay annotation | Echo index annotation next to specter nodes | P1 | todo |
| CL-007 | Cross-Layer Visibility | pkg/app/eventbus.go | UI subscribes/emits Event* messages | Bridge event bus to `tea.Msg` in `pkg/tui/bridge` | P0 | todo |
| CL-008 | Cross-Layer Visibility | pkg/app/eventbus.go (EventUserAction) | UI-origin actions emitted on bus | Emit TUI actions via EventUserAction for parity | P0 | todo |

## Keyboard/Mouse Binding Parity Map

Source bindings audited from `pkg/pulsemap/game.go`, `pkg/onboarding/screens/*.go`, and onboarding tutorial hints.

| Ebitengine Binding | Current Behavior | TUI Mapping |
|---|---|---|
| Mouse left click | Select node / activate button | Mouse left click (tea mouse), Enter fallback |
| Mouse right click | Open radial menu on node | Mouse right click opens action menu |
| Mouse drag (left) | Pan Pulse Map camera | Mouse drag pans viewport |
| Mouse wheel | Zoom Pulse Map | Wheel zoom; `+/-` fallback |
| Touch single tap | Select node | Mouse click / Enter on focused node |
| Touch double tap | Center/zoom node | Mouse double click / `z` action |
| Touch long press | Open node radial menu | Right click / `m` action |
| `Ctrl+N` | Open compose panel | `Ctrl+N` open compose |
| `Ctrl+F` | Open search bar | `Ctrl+F` open search |
| `Ctrl+,` | Toggle settings panel | `Ctrl+,` toggle settings |
| `Home` / `H` | Recenter to ego node | `Home` / `H` recenter |
| `N` | Fit network in viewport | `N` fit graph |
| `Ctrl+B` | Add/update bookmark | `Ctrl+B` add bookmark |
| `Ctrl+Shift+B` | Remove bookmark | `Ctrl+Shift+B` remove bookmark |
| `Ctrl+1..9` | Jump to bookmark index | `Ctrl+1..9` jump bookmark |
| `Esc` (panels/screens) | Close modal/back step | `Esc` close modal/back |
| `Enter` / Numpad Enter | Confirm/continue | `Enter` confirm |
| `Backspace` | Edit text fields | `Backspace` edit text |
| Left/Right arrows (mode screen) | Cycle privacy mode in onboarding | Left/Right cycle mode selection |
| `Space`/Enter/click (returning screen) | Continue to app | `Space`/Enter/click continue |
| Tutorial hint: `W` | Compose first Wave hint | `W` opens compose (alias to Ctrl+N) |
| Required TUI keys | N/A in Ebitengine | Add `h/j/k/l`, arrows, Tab cycle, numeric shortcuts, `?` help, `q`/`Ctrl+C` quit |

### Deferred Visual-Only Items (explicit)
- GPU/post-processing effects (blur/composite shaders) are mapped to terminal-safe styling and not pixel-equivalent.
- Particle and glow fidelity is approximated by ANSI color pulses/spinners because terminal rendering lacks shader pipelines.

