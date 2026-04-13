# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Refactored

- **2026-04-13**: Refactored 5 functions exceeding complexity thresholds
  - `PlaceMark` (marks.go): 8.3 → 5.7 overall (-31%), extracted `createMark`, `signMark`, `storeMark`
  - `drawModeIntro` (mode_screen.go): 7.5 → 5.7 overall (-24%), extracted `drawSurfaceLayerNode`, `drawAnonymousLayerNode`, `drawAnonParticles`, `drawIntroExplanationText`
  - `NewPhantomCouncil` (councils.go): 7.0 → 3.1 overall (-56%), extracted `validateCouncilParams`, `initCouncil`, `addFoundingMember`
  - `drawFirstWavePrompt` (bootstrap_screen.go): 5.7 → 1.3 overall (-77%), extracted `drawWaveInputArea`, `drawWaveSuggestions`, `drawWaveButtons`
  - `New` (gossip.go): 46 → 18 lines (-61%), extracted `buildPeerScoreParams`, `buildDefaultTopicParams`, `buildScoreThresholds`

### Updated

- **2026-04-13**: Updated GAPS.md to reflect current implementation status
  - All 15 original implementation gaps have been resolved
  - Only remaining gap is bootstrap nodes (requires external infrastructure, blocked)
  - Document now serves as historical record of completed work
  - Updated "Documentation vs Implementation Gaps" table showing all inconsistencies resolved

### Fixed

- **2026-04-13**: Fixed flaky `TestValidateInvalidPoW` test in `pkg/content/waves`
  - Root cause: Test assumed PoW at difficulty 8 would always fail validation at difficulty 16
  - Statistical flaw: ~1/256 probability the hash coincidentally has 16+ leading zeros
  - Fix: Corrupt nonce by +1 to deterministically invalidate PoW verification
  - Classified as Cat 2 (Test Spec Error) per complexity-based failure analysis

### Changed

- **2026-04-13**: Updated AUDIT.md bootstrap nodes item to `[~]` (blocked status)
  - The "No bootstrap nodes defined" finding requires live network infrastructure to resolve
  - Cannot be completed through code changes alone; requires community-operated bootstrap nodes
  - Added clarifying note explaining the external dependency

### Added

- **2026-04-13**: Extended stability simulation test infrastructure
  - Created `pkg/app/stability_simulation_test.go` with 1000-node, 72-hour stability test infrastructure
  - TestStabilityShortDuration: 1 minute, 100 nodes (CI-safe quick validation)
  - TestStabilityMediumDuration: 10 minutes, 500 nodes (medium validation)
  - TestStability1000Nodes: 1 hour, 1000 nodes (scale validation)
  - TestStability72Hour: Full 72-hour, 1000 nodes (production validation)
  - StabilityNetwork with metrics: memory tracking, panic recovery, deadlock detection
  - Run full test with: `go test -tags=simulation -run TestStability72Hour -timeout 73h`
  - Completes ROADMAP.md Priority 12 validation requirements

- **2026-04-13**: Test coverage for previously untested packages
  - Created `pkg/config/config_test.go` with tests for default listen addresses and bootstrap peers
  - Created `pkg/identity/declarations/declarations_test.go` with tests for Declaration struct
  - Improved test coverage for core configuration and identity packages

- **2026-04-13**: Phantom Gift visual effects integration for Pulse Map
  - Created `pkg/pulsemap/rendering/effects/gifts.go` with GiftRenderer and GiftEffectConfig
  - Created `pkg/pulsemap/rendering/effects/gifts_test.go` with comprehensive tests
  - TestPhantomGiftVisibility validates ROADMAP.md Priority 8 validation criteria
  - TestResonanceTieredEffects validates Resonance tier requirements (25/50/100)
  - TestGiftExpiration validates 7-day expiration per ANONYMOUS_GAME_MECHANICS.md spec
  - Effect configurations for all 25+ gift effect types (basic, expanded, premium)

### Changed

- **2026-04-13**: Updated ROADMAP.md Priority 12 validation to mark stability test infrastructure complete

## Notes

This changelog was established 2026-04-13 to track implementation progress. Prior work is documented in ROADMAP.md and AUDIT.md.
