# PLAN

Last updated: 2026-05-07

## Active Track: Desktop + WASM Deployment

- [x] Create browser entry scaffolding under cmd/wasm with version metadata.
- [x] Add shared runtime layer for platform selection in pkg/game.
- [x] Define transport abstraction in pkg/network for desktop libp2p and browser WebRTC adapters.
- [x] Define input normalization contract in pkg/input for touch + keyboard/mouse parity.
- [x] Add static web shell under web/ (index.html, boot.js, style.css).
- [x] Add reproducible site build script (scripts/build-wasm-site.sh).
- [x] Add GitHub Pages workflow to build and deploy WASM bundle.

## Next Steps

- [x] Repair the desktop first-run onboarding handoff so completing onboarding enters the Pulse Map immediately instead of canceling the app.
- [x] Implement concrete desktop adapter in pkg/network backed by pkg/networking/transport + GossipSub.
- [x] Implement concrete wasm adapter in pkg/network backed by pion/webrtc data channels.
- [x] Add relay/bootstrap discovery policy for browser peers (no mDNS dependency).
- [x] Integrate input mapper with Pulse Map interaction handlers.
- [x] Add responsive layout policies for mobile viewport breakpoints.
- [x] Add desktop-browser interop integration tests.
- [x] Add dynamic bootstrap server command (`cmd/bootstrap`) with DHT server-mode participation, automatic peer learning/distribution, and multi-listener support for HTTP/ngrok/Tor/I2P ingress.
- [x] Add container deployment assets for the dynamic bootstrap server (`Dockerfile.bootstrap`, Compose example, and operator docs covering configurable ngrok domains and announced public libp2p addresses).
- [x] Repair corrupted transport integration tests in `pkg/networking/transport/integration_test.go` and realign to current diagnostics/host APIs.
- [x] Harden bootstrap Docker build path for restricted DNS environments with host-network compose builds and configurable Go module proxy args.
- [x] Add a MURMUR-specific UI audit prompt in `UI_AUDIT.md` focused on first-run comprehension, discoverability, and obvious Pulse Map interaction guidance.
- [x] Resolve Ebitengine transition/input audit findings: one-shot scene transitions, modal-safe shortcut routing, UTF-8 text deletion correctness, and minimap redraw caching.
- [x] Apply follow-up UI audit fixes: synchronous returning-screen handoff, pointer-based radial-menu targeting, continuous world ticking during modal input consumption, tick-based caret blink, and time-based camera interpolation.
- [x] Resolve remaining UI clarity audit issues in active UX paths: interactive device-management controls, visible settings labels/values, minimap integration in Pulse Map draw path, explicit returning-screen continuation, onboarding keyboard parity, explicit node-detail empty-state feedback, and radial-menu glyph icon rendering.
- [x] Resolve transition/input audit findings end-to-end: onboarding phase sequencing to Pulse Map, visible-overlay input isolation for SearchBar, minimap Draw allocation reuse, ComposePanel Update-time hit-test positioning, and viewport-responsive RecoveryScreen hit targets/layout.
- [x] Close follow-up transition/input findings: same-node detail reopen continuity, SearchBar hide ghost state, display-name Enter focus guard, responsive mode-card hit-testing layout, touch deferred-tap reset clearing, recovery-frame background clearing, scalable onboarding text sizing, and micro-zoom gating for cross-layer artifact queries.
