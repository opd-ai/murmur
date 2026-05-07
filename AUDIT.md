# AUDIT

Last updated: 2026-05-06

## Security/Architecture Notes

- Added platform transport boundary (pkg/network) to keep protocol logic reusable while isolating platform-specific network stacks.
- Implemented desktop adapter runtime path in `pkg/network` using `pkg/networking/transport` host construction and `pkg/networking/gossip` topic messaging.
- Implemented WASM adapter runtime path in `pkg/network` using `pion/webrtc` data channels, including lifecycle management and topic publish/subscribe dispatch over a browser-compatible transport primitive.
- Added browser discovery policy in `pkg/network` that prioritizes relay peers before bootstrap peers, deduplicates candidates, and constrains peer selection to configured relay/bootstrap sources (no mDNS dependency).
- Integrated normalized input mapping into Pulse Map interaction flow so mouse-wheel zoom and mouse/touch pan paths route through `pkg/input` action normalization before camera updates.
- Browser deployment path uses static hosting only; no custom backend service introduced.
- Web entrypoint is static and does not embed secrets.
- Build process injects version/commit metadata into wasm binary for traceability.

## Risks and Follow-up

- Browser peer signaling and negotiated remote WebRTC session setup remain pending (policy selection is implemented; remote offer/answer exchange is not yet wired).
- Responsive mobile viewport policy tuning remains pending.

## Validation Performed

- go test ./pkg/network -run TestDesktopAdapter -count=1
- go test ./pkg/network -count=1
- go test ./pkg/input ./pkg/pulsemap -count=1
- GOOS=js GOARCH=wasm go test ./pkg/network -c
- go test -race ./...
- go vet ./...
