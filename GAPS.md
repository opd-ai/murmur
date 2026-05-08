# Best Practices Gaps — 2026-05-08

## Missing CI Enforcement of Declared Quality Gates
- **Best Practice**: CI should enforce core quality checks (`go test`, `go vet`, and agreed lint/static analysis) on every PR.
- **Current Practice**: Repository guidance requires test/vet, but visible workflows build artifacts and run security scans without running these checks.
- **Impact**: Increases regression risk and allows avoidable quality drift before merge.
- **Recommendation**: Add a dedicated PR CI workflow/job running `go test -race ./...` and `go vet ./...` (or headless/tagged equivalents for Ebitengine constraints), and fail on errors.
- **Scope**: Project-wide (all packages and contributors).

## Non-Idiomatic Package Naming with Underscores
- **Best Practice**: Package names should be short, lowercase, and generally avoid underscores.
- **Current Practice**: Transport subpackages use underscore names (`onramp_i2p`, `onramp_tor`).
- **Impact**: Minor readability and convention mismatch in imports and public docs.
- **Recommendation**: Plan a staged rename to idiomatic package identifiers with bulk import updates and compatibility notes.
- **Scope**: 2 production packages.

## Latent Type-Erased Gossip Validation Path
- **Best Practice**: Decode wire messages into concrete types and pass typed values through handler boundaries.
- **Current Practice**: `ValidatingMessageHandlers` uses `interface{}` placeholder decode path and forwards `nil` message payloads.
- **Impact**: Low current runtime impact (path appears test-only), but medium future risk if wired into production because behavior would be silently incorrect.
- **Recommendation**: Replace placeholder with concrete `pb.GossipMessage` unmarshal and assert non-nil payload propagation in tests.
- **Scope**: Localized to `pkg/networking/gossip/scoring.go` validating dispatch path.

## Getter Naming Divergence from Go Style
- **Best Practice**: Getter methods should avoid `Get` prefix (`Name()` not `GetName()`).
- **Current Practice**: Exported APIs widely use `Get*` method naming.
- **Impact**: Low; mostly stylistic, but reduces idiomatic familiarity for new Go contributors.
- **Recommendation**: Keep consistency short-term; adopt gradual migration for high-traffic exported APIs and preserve compatibility wrappers during transition.
- **Scope**: Project-wide pattern (many packages).

## Type-Erased Settings Value Model
- **Best Practice**: Prefer typed models/generics over broad `interface{}` for domain values.
- **Current Practice**: Settings values and mutation APIs store/pass `interface{}` and convert types at runtime.
- **Impact**: Runtime type safety burden, harder refactoring, and weaker static guarantees for settings evolution.
- **Recommendation**: Introduce typed setting value abstractions and limit dynamic conversions to rendering/input boundaries.
- **Scope**: `pkg/ui/settings.go` API and dependent settings flows.
