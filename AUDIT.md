# AUDIT

# BREAKING BUG AUDIT — 2026-05-07

## Observed Failure
`go run ./cmd/desktop` on a valid first-run configuration started normally, completed onboarding, printed `Onboarding complete, transitioning to Pulse Map...`, and then shut down the process instead of leaving the user in a usable Pulse Map session. The user-visible symptom included repeated onboarding completion logs followed by subsystem shutdown lines such as `[SHUTDOWN] Event bus goroutine exited` with no persistent UI.

## Root Cause
`pkg/app/ui.go:105` and `pkg/app/ui.go:130-134`, `runOnboardingUI`: the onboarding callback only logged the intended Pulse Map transition, but the actual code path returned from `ebiten.RunGame()` into unconditional app cancellation. On first run, the desktop path therefore exited instead of handing off into the Pulse Map.

## Fix Applied
Updated `pkg/app/ui.go` so first-run onboarding performs an in-loop handoff into the Pulse Map: it now persists `first_run_complete` and swaps from the onboarding game to the Pulse Map game inside the same `ebiten.RunGame()` session, avoiding forced termination/restart behavior.

## Verification
- `go test ./pkg/app/murmur_test.go ./pkg/app/ui_test.go`
- `go build ./...`
- `go test -race ./...`
- `go-stats-generator analyze . --skip-tests --format json --output post-fix.json --sections functions,patterns`
- `go-stats-generator diff baseline.json post-fix.json`

All targeted and full-suite tests passed. Full interactive first-run GUI validation could not be automated end-to-end in this session because the desktop binary has no scripted onboarding driver; the fix was validated by the repaired execution path plus the passing build and race suite.

## Other Blocking Bugs Found
- [x] Returning-user continuation callback now one-shot and transition-safe.
- [x] Onboarding key/specter generation no longer re-enters from Draw().
- [x] Pulse Map global shortcuts now blocked while modal UI is active.
- [x] `Ctrl+N` compose toggle conflict with network recenter resolved.
- [x] UTF-8 backspace correctness restored for onboarding/settings text inputs.
- [x] Minimap static content now cached instead of fully redrawn each frame.

## 2026-05-07 — Ebitengine UI Transition/Input Remediation

- Fixed a CRITICAL returning-screen transition hazard where the continuation callback could fire every frame and close an already-closed channel in the app handoff path.
- Removed Draw-path goroutine spawning for onboarding identity/specter generation and moved generation trigger logic to one-shot Update guards.
- Hardened Pulse Map input-mode isolation by blocking global shortcuts while modal UI is visible and deconflicting `Ctrl+N` from plain `N` network-view behavior.
- Fixed rune/UTF-8 deletion correctness in onboarding and settings text entry backspace handling.
- Eliminated read-lock mutation patterns in compose/passphrase draw paths by using write locks for geometry cache updates.
- Added minimap static-layer caching to reduce per-frame redraw work and transition-time hitch risk.

Verification:
- `go test ./pkg/app ./pkg/onboarding/screens ./pkg/pulsemap ./pkg/pulsemap/overlays ./pkg/ui`

## 2026-05-07 — UI Audit Follow-up Fixes

- Fixed returning-user RunGame lifecycle race by removing asynchronous RunGame handoff in `pkg/app/ui.go` and running the welcome screen synchronously before Pulse Map startup.
- Reduced stale-target action risk by resolving radial-menu targets from pointer/touch hit-test positions in `pkg/pulsemap/game.go` via `Renderer.NodeAtScreen`, with selected-node fallback.
- Preserved transition continuity by continuing renderer/world tick updates while modal UI consumes input, while explicitly resetting drag/touch transient state to prevent input leakage.
- Made onboarding first-wave backspace rune-safe (`pkg/onboarding/screens/bootstrap_screen.go`) to avoid UTF-8 corruption.
- Made Search caret blink deterministic using update ticks instead of static TPS-derived gating (`pkg/ui/search.go`).
- Updated camera interpolation/momentum integration to be time-based in `pkg/pulsemap/interaction/input.go` for more consistent motion under frame-rate variance.

Verification:
- `go test ./pkg/app ./pkg/pulsemap ./pkg/pulsemap/interaction ./pkg/onboarding/screens ./pkg/ui`

## 2026-05-07 — Transition/Input Audit Resolution Pass

- Resolved onboarding transition continuity issue by sequencing active onboarding UI by controller phase in `pkg/app/ui.go` (`Screen` -> `ModeScreen` -> `BootstrapScreen`) and transitioning to Pulse Map only once `flow.Controller.IsComplete()` is true.
- Hardened input isolation for search overlay: `SearchBar.Update()` now consumes input whenever visible to prevent click-through pan/selection in the Pulse Map.
- Removed per-frame minimap projection allocation in `pkg/pulsemap/game.go` by reusing a `minimapNodes` scratch buffer.
- Fixed ComposePanel click-hit race window by recomputing panel origin in `Update()` before mouse hit-testing (no longer dependent on prior `Draw()` cache timing).
- Made RecoveryScreen controls responsive to current viewport dimensions (dynamic center/box/button placement and matching hit-test coordinates) instead of fixed 800x600 anchors.

Verification:
- `go test ./pkg/app ./pkg/pulsemap/... ./pkg/ui ./pkg/onboarding/screens`

## 2026-05-07 — Transition/Input Closure Follow-up

- Resolved node-detail continuity issue by clearing `lastSelectedNode` on panel close, allowing same-node reopen without forced intermediate selection.
- Eliminated SearchBar stale-ghost transition state by zeroing opacity on hide/toggle-off so Draw cannot keep rendering a non-interactive overlay.
- Added display-name focus guard in onboarding identity flow so Enter does not advance phases while text input is active.
- Converted mode-selection cards to responsive layout logic shared across draw/click/hover paths (4-column desktop, 2-column narrow viewports) to prevent card clipping and mismatched hit targets.
- Hardened touch reset logic to clear deferred pending-tap debounce state, preventing stale tap emission after modal/state transitions.
- Added full-frame background clear in RecoveryScreen Draw path to prevent stale previous-frame artifacts during recovery state transitions.
- Switched onboarding shared text helper to scalable GoText face source and honoring requested text size for improved legibility and consistent typographic intent.
- Reduced transition-time frame hitch risk by limiting expensive cross-layer artifact store queries to Micro zoom where artifact detail is actually visible.

Verification:
- `go test ./pkg/pulsemap/... ./pkg/ui ./pkg/onboarding/screens ./pkg/app`

## 2026-05-07 — UI Clarity Remediation Batch

- Completed direct remediation of remaining high-friction Ebitengine UX paths identified in the static clarity audit.
- Restored full pointer-based interaction in device management (`pkg/ui/device_management.go`) for add-device, revoke-device, and confirmation dialog flows.
- Improved settings readability (`pkg/ui/settings.go`) by rendering row labels/descriptions and displaying live slider/select/text values.
- Reconnected Pulse Map orientation aid by integrating minimap initialization/update/draw in active game flow (`pkg/pulsemap/game.go`).
- Removed returning-screen forced auto-continue and required explicit user progression with visible prompt (`pkg/onboarding/screens/returning_screen.go`).
- Added keyboard parity across onboarding phases (`pkg/onboarding/screens/identity.go`, `pkg/onboarding/screens/mode_screen.go`, `pkg/onboarding/screens/bootstrap_screen.go`, `pkg/onboarding/screens/completion_screen.go`) via Enter/Space progression and Escape back-step where appropriate.
- Improved node-inspection feedback by replacing silent placeholders with explicit informational rows and connection rendering (`pkg/pulsemap/game.go`, `pkg/ui/node_detail.go`).
- Replaced radial-menu placeholder dots with actual configured glyph icon rendering (`pkg/ui/radial_menu.go`).
- Changed wave-result enqueue semantics in Pulse Map to avoid silent feedback loss under full-channel pressure (`pkg/pulsemap/game.go`).

Verification:
- `go test ./pkg/ui ./pkg/pulsemap/... ./pkg/onboarding/screens`
- `go vet ./pkg/ui ./pkg/pulsemap/... ./pkg/onboarding/screens`

## Discarded Candidates
| Candidate | Reason Discarded |
|-----------|-----------------|
| Repeated `Onboarding: Phase Identity Creation complete` logging | User-visible noise, but not the blocking bug. Basic utility was blocked by the missing onboarding-to-Pulse-Map handoff and unconditional cancellation path. |

## 2026-05-07

- Bootstrap Docker build hardening for DNS-constrained environments (`Dockerfile.bootstrap`, `docker-compose.bootstrap.example.yml`, `docs/BOOTSTRAP_OPERATION.md`).
- Security impact: low. Changes affect build-time dependency resolution only; runtime networking and trust boundaries are unchanged.
- Verification: `docker compose -f docker-compose.bootstrap.example.yml config` after compose update.

- Transport integration test repair in `pkg/networking/transport/integration_test.go`.
- Security impact: none. Changes are test-only and do not modify runtime transport, cryptography, or trust boundaries.
- Verification: compiled package with integration build tag and ran daemon-independent protocol parsing test.

- Added `UI_AUDIT.md`, a documentation-only audit prompt for Ebitengine UI review with emphasis on discoverability, onboarding clarity, Pulse Map navigation, and first-time user comprehension.
- Security impact: none. This change adds no runtime code, protocol changes, or trust-boundary modifications.
- Verification: file creation and content diff review.

## 2026-05-07 — WASM Boot Sequence Event-Driven Callback Implementation

**Observed Issue**: WASM browser build froze on "Starting runtime..." during boot.js initialization sequence.

**Root Cause**: Architectural deadlock between boot.js and cmd/wasm/main.go:
- boot.js called `await go.run()` expecting the function to return after Go initialization
- cmd/wasm/main.go ended with `select {}` (blocking forever to keep runtime alive for event handling)
- go.run() wraps the Go main() function and cannot return until main() completes
- Therefore, the await in boot.js stalled indefinitely, never reaching "Runtime started." message

**Implementation**: Replaced synchronous await model with proper event-driven callback architecture:
1. web/boot.js and web-dist/boot.js now create a Promise before calling go.run()
2. Setup window.murmur.onRuntimeReady callback that resolves the Promise
3. Call go.run() without awaiting (fire-and-forget pattern)
4. Await the Promise to complete the loading sequence
5. cmd/wasm/main.go now calls triggerReadyCallback(success, errMsg) after runtime.Run() completes
6. triggerReadyCallback() invokes window.murmur.onRuntimeReady() if set, allowing boot.js to proceed

**Security Impact**: None. This is a boot-sequence bug fix. No cryptography, protocol, or trust boundaries affected. The callback mechanism is uncontested initialization synchronization.

**Verification**:
- WASM binary compiles without errors: `GOOS=js GOARCH=wasm go build -v ./cmd/wasm`
- No blocking code in init() functions detected
- wasm_exec.js matches Go 1.25.9 toolchain version (575 lines, valid syntax)
- Callback flows correctly: successful init → onRuntimeReady() invoked → Promise resolves → "Runtime started." displayed
- Error cases handled: if runtime.Run() fails, errMsg passed to callback
- Graceful degradation: if callback not set (boot.js not loaded), Go runtime continues normally

**Files Modified**:
- cmd/wasm/main.go: Added triggerReadyCallback() function, modified main() to call it after initialization
- web/boot.js: Event-driven Promise + callback pattern instead of await
- web-dist/boot.js: Same pattern (deployed version)
