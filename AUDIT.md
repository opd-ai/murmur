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
- [ ] None confirmed on the same critical path after the handoff fix.

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
