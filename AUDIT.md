# AUDIT

Last updated: 2026-05-07

## Security Notes

- 2026-05-07: Added standalone bootstrap server command at `cmd/bootstrap` with optional ngrok, Tor, and I2P listeners.
- Threat surface change: bootstrap/reseed host can now expose `/peers.json` across multiple ingress transports simultaneously.
- Controls implemented: required signed peers file path (`-peers-file`), JSON shape validation before startup, non-empty peer list enforcement, and graceful listener shutdown for all transports.
- Residual risks: this command serves a static signed peer bundle and does not enforce capability-based reseed authorization yet; operators should deploy behind restricted access paths where appropriate.
- Follow-up needed: add explicit capability/token checks and rate limiting for reseed-grade operation per `docs/RESEED_SEMANTICS.md`.
