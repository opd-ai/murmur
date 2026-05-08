# PLAN

Last updated: 2026-05-08

## Scope

This file tracks only incomplete items pulled from ROADMAP.md.

## Milestone v1.0 — Production Readiness

### Testing

- [x] Wave TTL expiration correctness (end-to-end validation) — six end-to-end TTL lifecycle tests added in `pkg/content/storage/ttl_e2e_test.go` (2026-05-08)
- [x] Mini-game network propagation end-to-end — publish→network→receive→state-update tests added in `pkg/anonymous/mechanics/hunts/network_propagation_e2e_test.go` (2026-05-08)
- [x] **Ebitengine headless mode** screenshot comparison tests for rendering — `TestMain` game-loop harness + 4 pixel-comparison tests in `pkg/pulsemap/rendering/screenshot_comparison_test.go` (build tag `ebitentest`, run via `xvfb-run`) (2026-05-08)

### Documentation

- [x] API documentation for all exported types and functions — added doc comments to 13 previously undocumented exported identifiers across `pkg/pulsemap/overlays`, `pkg/anonymous/shroud`, `pkg/anonymous/mechanics/oracle` (2026-05-08)
- [x] Architecture decision records (ADRs) for key design choices — added D-011 through D-015 (cryptographic primitives, proto3, BBolt, Ebitengine, pkg/ layout) to `docs/DECISIONS.md` (2026-05-08)

### Deployment

- [x] Bootstrap node infrastructure (8–12 community-operated nodes) — `cmd/bootstrap` server, `docs/BOOTSTRAP_OPERATION.md`, `Dockerfile.bootstrap`, and `docker-compose.bootstrap.example.yml` already implemented (pre-existing)
- [x] Docker container image for bootstrap/relay operators — `Dockerfile.bootstrap` multi-stage alpine image and `docker-compose.bootstrap.example.yml` already implemented (pre-existing)
- [x] Platform-specific binary releases (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) — fully implemented in `.github/workflows/build.yml` with ncipollo/release-action (pre-existing)
  - [x] CI release publishing action migrated to `ncipollo/release-action` in `.github/workflows/build.yml` (2026-05-08)
- [x] Android APK distribution — implemented via `.github/workflows/mobile.yml` and `scripts/build-mobile.sh` (pre-existing)
- [x] Dedicated `cmd/murmur-mobile` gomobile entrypoint now imports `golang.org/x/mobile/app` to satisfy mobile package detection (2026-05-08)
- [x] iOS xcframework distribution — implemented via `.github/workflows/mobile.yml` and `scripts/build-mobile.sh` (macOS runner) (pre-existing)
- [x] Version upgrade protocol — dual-subscription migration (v1 + v2 topics) — `DualTopicManager`, v2 topic constants, `SubscribeBoth`, `PublishVersioned` in `pkg/networking/gossip/version_upgrade.go` (2026-05-08)

## Cross-Cutting Concerns

### Anti-Sybil & Spam Resistance

- [ ] Resonance gating on all privileged actions (gifts, marks, games, councils)
- [ ] Connection pruning for consistently low-score peers
- [ ] PoW requirement for identity creation
- [ ] Sybil defense: PoW cost scales linearly per identity

### Protocol Versioning

- [x] Topic versioning in GossipSub topic strings — v2 constants and `topicUpgradePairs` map in `pkg/networking/gossip/version_upgrade.go` (2026-05-08)
- [ ] MurmurEnvelope version field handling (currently always 1)
- [ ] Protocol negotiation via multistream-select
- [x] Gradual migration: new-version nodes subscribe to both v1 and v2 topics — `DualTopicManager.SubscribeBoth` in `pkg/networking/gossip/version_upgrade.go` (2026-05-08)
- [ ] Breaking change consensus mechanism

### Accessibility & UX

- [ ] Compose panel — Wave input interface with character count
- [ ] Settings panel — configuration options (privacy mode, difficulty, filters)
- [ ] Help button — reopen onboarding tutorials at any time
- [ ] Modal dialogs — confirmations and warnings for destructive actions
- [ ] Status indicators — identity publication, Shroud circuit, connection count, PoW progress
- [x] Bubble Tea terminal UI baseline scaffold (`cmd/murmur-tui`) with tabbed domain views, key/mouse navigation, and feature matrix artifact (2026-05-08)
- [x] Bubble Tea terminal parity expansion for key P0 surfaces: Pulse Map search/detail/minimap, onboarding bootstrap progress, Waves thread/reply preview, settings/layer blend controls (2026-05-08)
- [x] Bubble Tea terminal parity closure: all remaining P1 matrix rows implemented and marked done, including anonymous mechanics boards, onboarding hint/resume branches, and overlay/config surfaces (2026-05-08)
- [x] Bubble Tea terminal parity completion: all P2 matrix rows implemented with terminal-safe equivalents; no remaining parity gaps in `docs/TUI_FEATURE_MATRIX.md` (2026-05-08)

## Release Candidate Follow-Up

### Next Steps for v0.1 Release Candidate

- [ ] v0.1.0-rc1 tag (ready for tagging)
