# BEST PRACTICES AUDIT — 2026-05-08

## Project Conventions Summary
- Mature Go module (`github.com/opd-ai/murmur`) with broad subsystem layout under `pkg/` and multiple binaries under `cmd/` (`go list ./...` returned 86 packages).
- Team conventions are explicitly documented in `CONTRIBUTING.md`: `gofumpt -w -extra .`, `go vet ./...`, `go test -race ./...`, and `pkg/`-first package layout (`/home/runner/work/murmur/murmur/CONTRIBUTING.md:22-30`, `:51-55`).
- Codebase favors strong package documentation and exported symbol docs (go-stats overall doc coverage 81.6%, function docs 92.2%, package docs 87.3%).
- Naming and API style is internally consistent but intentionally diverges from strict Go getter naming in many areas (project-wide `Get*` pattern).

## Go Version and Feature Availability
- `go.mod` declares `go 1.25.7` with `toolchain go1.25.9` (`/home/runner/work/murmur/murmur/go.mod:3-5`).
- Available features include generics, `any`, `slices`/`maps` stdlib helpers, and modern error wrapping idioms.
- No `.golangci*` configuration detected; CI workflows currently do not run `go vet`, `go test`, `staticcheck`, or `golangci-lint` (`/home/runner/work/murmur/murmur/.github/workflows/*.yml`).

## Practices Scorecard
| Category | Adherence | Notes |
|----------|-----------|-------|
| Package Design | ⚠️ | Clear subsystem split and no circular dependencies, but non-idiomatic underscore package names exist. |
| Naming & Style | ⚠️ | Strong documentation and naming consistency overall; widespread exported `Get*` getter methods diverge from Go convention. |
| Error Handling | ✅ | Predominantly idiomatic error returns/wrapping; no systemic panic-for-operational-errors pattern found. |
| Concurrency | ✅ | Context usage and synchronization patterns are broadly idiomatic; no obvious lifecycle anti-pattern cluster found in sampled paths. |
| Interface Design | ⚠️ | Most interfaces are small/consumer-side, but some `interface{}`-based APIs reduce type safety. |
| Testing | ⚠️ | Rich test surface exists, but CI workflows shown do not execute tests/lint gates. |
| API Design | ⚠️ | Public API has several non-idiomatic getter names and some type-erased settings APIs. |
| Module Management | ✅ | Module path matches repo URL and dependency/security workflow exists (`govulncheck`). |

## Findings
### CRITICAL
- [x] None identified.

### HIGH
- [x] CI does not enforce repository-stated Go quality gates (`go test -race` / `go vet`) — `/home/runner/work/murmur/murmur/CONTRIBUTING.md:22-30`, `/home/runner/work/murmur/murmur/.github/workflows/build.yml:47-154`, `/home/runner/work/murmur/murmur/.github/workflows/security.yml:21-39`, `/home/runner/work/murmur/murmur/.github/workflows/ci.yml:1-100`, `/home/runner/work/murmur/murmur/cmd/murmur/main_test.go:16-80` — practice violation: build workflows compile artifacts and run govulncheck but did not run baseline test/lint checks — concrete impact: regressions and style/analysis issues could merge without automated detection. — **Remediation:** Add a dedicated CI job running `go test -race ./...` and `go vet ./...` (or equivalent tagged/headless variants), fail PRs on errors, and keep security scan separate; **Validation:** `ci.yml` now runs `go vet ./...`, `xvfb-run ... go test -race ./...`, and a go-stats regression check with the module toolchain; `TestRunWithConfig` now synchronizes app creation so the race gate passes locally.

### MEDIUM
- [x] Non-idiomatic underscore package names in production packages — `/home/runner/work/murmur/murmur/pkg/networking/transport/onramp_i2p/transport.go:3`, `/home/runner/work/murmur/murmur/pkg/networking/transport/onramp_tor/transport.go:3` — practice violation: Go package names conventionally avoid underscores — concrete impact: reduces idiomatic readability and increases naming inconsistency in imports and generated docs. — **Remediation:** Gradually migrate package names to `onrampi2p`/`onramptor` (or concise alternatives), update imports atomically, and provide migration notes due cross-package scope; **Validation:** package declarations were renamed to `onrampi2p` and `onramptor`, transport host callsites were updated, and `go list ./...` now succeeds with no stale imports.
- [x] Placeholder validating-handler decode path uses `interface{}` with no real unmarshal and dispatches `nil` payloads — `/home/runner/work/murmur/murmur/pkg/networking/gossip/scoring.go:331-339`, `:342-388` — practice violation: typed message handling is replaced by no-op placeholder logic — concrete impact: latent correctness risk if this validating path is wired into runtime (handler contracts expect `*pb.GossipMessage`); future integration could silently drop message content. — **Remediation:** Replace `GossipMessage interface{}` with concrete `pb.GossipMessage`, perform `proto.Unmarshal`, and pass parsed message through all `handle*WithMsg` methods; **Validation:** `handleMessageWithEnvelope` now unmarshals into `pb.GossipMessage` via `proto.Unmarshal` and dispatches parsed payloads to typed handlers; `TestValidatingMessageHandlers_DispatchesParsedMessageToHandlers` verifies non-nil parsed messages reach wave/identity/shroud/pulse handlers; `go test ./pkg/networking/gossip/...` passes.

### LOW
- [ ] Exported getter methods widely use `Get*` naming across public API surface — e.g., `/home/runner/work/murmur/murmur/pkg/anonymous/specters/identity.go:268`, `:278`, `:287`; `/home/runner/work/murmur/murmur/pkg/onboarding/screens/identity.go:790-800`; `/home/runner/work/murmur/murmur/pkg/pulsemap/interaction/input.go:58` — practice deviation: Go style prefers `Value()` over `GetValue()` for getters — concrete impact: lower idiomatic familiarity for Go contributors, but pattern is internally consistent project-wide. — **Remediation:** Treat as gradual migration, prioritizing exported API hot paths first and preserving compatibility wrappers where needed; **Validation:** API docs and callsites show reduced `Get*` getter usage over time.
- [ ] Type-erased settings API (`interface{}` values) weakens compile-time safety — `/home/runner/work/murmur/murmur/pkg/ui/settings.go:32`, `:642`, `:657`, `:669`, `:689` — practice deviation: broad `interface{}` usage where typed structs/generics could be used — concrete impact: runtime type assertions and string conversions become the enforcement point, increasing maintenance risk for settings evolution. — **Remediation:** Introduce typed setting value models (or generic wrappers) for known setting kinds and keep conversion at UI edges only; **Validation:** compile-time typed setting accessors replace `interface{}` storage and existing settings tests remain green.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| “Project should use `internal/` packages” | Rejected because this repository explicitly standardizes on `pkg/` in project conventions (`CONTRIBUTING.md:53`) and follows that consistently. |
| “`panic` usage is broadly incorrect” | Rejected as broad claim; sampled production panics are in `must*` helper/init contexts (e.g., deterministic marshal/font-source initialization) rather than common operational error paths. |
| “`interface{}` use in rendering uniforms is non-idiomatic” | Rejected because several cases are framework-constrained (`ebiten` shader uniforms require map values as interface types), making this an acceptable domain-specific exception. |
| “`go vet` baseline failure indicates code bug” | Rejected as code-quality finding; failure was environment dependency (`X11/Xlib.h` missing) not a Go semantic defect (`tmp/practices-vet-results.txt`). |
