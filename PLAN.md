# PLAN

Last updated: 2026-05-06

## Active Track: Desktop + WASM Deployment

- [x] Create browser entry scaffolding under cmd/wasm with version metadata.
- [x] Add shared runtime layer for platform selection in pkg/game.
- [x] Define transport abstraction in pkg/network for desktop libp2p and browser WebRTC adapters.
- [x] Define input normalization contract in pkg/input for touch + keyboard/mouse parity.
- [x] Add static web shell under web/ (index.html, boot.js, style.css).
- [x] Add reproducible site build script (scripts/build-wasm-site.sh).
- [x] Add GitHub Pages workflow to build and deploy WASM bundle.

## Next Steps

- [x] Implement concrete desktop adapter in pkg/network backed by pkg/networking/transport + GossipSub.
- [x] Implement concrete wasm adapter in pkg/network backed by pion/webrtc data channels.
- [x] Add relay/bootstrap discovery policy for browser peers (no mDNS dependency).
- [x] Integrate input mapper with Pulse Map interaction handlers.
- [x] Add responsive layout policies for mobile viewport breakpoints.
- [x] Add desktop-browser interop integration tests.
- [x] Add freestanding bootstrap server command (`cmd/bootstrap`) with multi-listener support for TCP, ngrok, Tor, and I2P transports.
- [x] Add container deployment assets for the bootstrap server (`Dockerfile.bootstrap`, Compose example, and operator docs covering configurable ngrok domains).
