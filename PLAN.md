# PLAN

Last updated: 2026-05-07

## Scope

This file tracks only incomplete items pulled from ROADMAP.md.

## Milestone v1.0 — Production Readiness

### Testing

- [ ] Wave TTL expiration correctness (end-to-end validation)
- [ ] Mini-game network propagation end-to-end
- [ ] **Ebitengine headless mode** screenshot comparison tests for rendering

### Documentation

- [ ] API documentation for all exported types and functions
- [ ] Architecture decision records (ADRs) for key design choices

### Deployment

- [ ] Bootstrap node infrastructure (8–12 community-operated nodes)
- [ ] Docker container image for bootstrap/relay operators
- [ ] Platform-specific binary releases (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64)
- [ ] Android APK distribution
- [ ] iOS xcframework distribution
- [ ] Version upgrade protocol — dual-subscription migration (v1 + v2 topics)

## Cross-Cutting Concerns

### Anti-Sybil & Spam Resistance

- [ ] Resonance gating on all privileged actions (gifts, marks, games, councils)
- [ ] Connection pruning for consistently low-score peers
- [ ] PoW requirement for identity creation
- [ ] Sybil defense: PoW cost scales linearly per identity

### Protocol Versioning

- [ ] Topic versioning in GossipSub topic strings
- [ ] MurmurEnvelope version field handling (currently always 1)
- [ ] Protocol negotiation via multistream-select
- [ ] Gradual migration: new-version nodes subscribe to both v1 and v2 topics
- [ ] Breaking change consensus mechanism

### Accessibility & UX

- [ ] Compose panel — Wave input interface with character count
- [ ] Settings panel — configuration options (privacy mode, difficulty, filters)
- [ ] Help button — reopen onboarding tutorials at any time
- [ ] Modal dialogs — confirmations and warnings for destructive actions
- [ ] Status indicators — identity publication, Shroud circuit, connection count, PoW progress

## Release Candidate Follow-Up

### Next Steps for v0.1 Release Candidate

- [ ] v0.1.0-rc1 tag (ready for tagging)
