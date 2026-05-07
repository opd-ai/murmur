# Security & Privacy Audit Log

This file records security-relevant implementation decisions, verified deviations, and follow-up review items.

## 2026-05-07

- Decision: Tunnel operator registration migrated from plaintext commands to signed `TunnelRegisterCell` verification in framed stream protocol (`pkg/tunneling/protocol/stream.go`, `pkg/tunneling/relay/relay.go`, `pkg/tunneling/initiator/initiator.go`).
- Security impact: Relay now validates Ed25519 operator signature and enforces register timestamp skew bounds before accepting tunnel sessions.
- Decision: Tunnel runtime now uses dedicated tunnel accounting (`pkg/tunneling/accounting`) with explicit quota checks and teardown on quota exceed.
- Security impact: Reduces abuse window for high-bandwidth tunnels and separates tunnel-traffic accounting from social traffic.
- Decision: Added `RequireShroud` policy behavior for tunnel mode selection (`pkg/tunneling/types.go`, `pkg/tunneling/initiator/initiator.go`).
- Security impact: Operators can enforce fail-closed behavior when Shroud relays/circuit are unavailable instead of implicit downgrade.
- Residual risk: Current relay still exposes a TCP ingress prototype path and does not yet route tunnel traffic over libp2p stream transport end-to-end.
- Follow-up: Replace TCP prototype transport with authenticated libp2p stream endpoints and add adversarial integration tests for replay, out-of-order frames, and forced downgrade attempts.
