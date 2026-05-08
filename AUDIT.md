# IMPLEMENTATION GAP AUDIT — 2026-05-07

## Remediation Update
This report was updated after applying code changes to close the implementation gaps identified in the prior audit pass.

Validation executed after remediation:
- `go build ./...` -> PASS
- `go test ./...` -> PASS
- `GOOS=js GOARCH=wasm go build ./cmd/wasm` -> PASS

Additional matrix checks attempted:
- `GOOS=linux GOARCH=amd64 go build ./cmd/murmur ./cmd/bootstrap ./cmd/desktop` -> PASS
- `GOOS=linux GOARCH=arm64 ...` -> blocked by upstream Ebitengine/OpenGL cross-compilation symbols in this environment
- `GOOS=darwin GOARCH=amd64 ...` -> blocked by upstream Ebitengine/OpenGL/Metal cross-compilation symbols in this environment

## Project Architecture Overview
MURMUR remains organized around six subsystems (networking, identity, content, anonymous mechanics, Pulse Map, onboarding) and a dual runtime strategy (desktop + WASM). The remediation focused on production execution paths that were previously placeholder or partially wired:
- WASM runtime readiness
- Anonymous mechanics/waves inbound processing
- Identity key encryption-at-rest and legacy migration
- Bootstrap fallback resolver-chain wiring and signature key hardening
- EventBus nudge integration
- Onboarding display-name persistence into identity declarations
- Metrics wiring for bootstrap and runtime memory/cache/circuit observability

## Gap Summary (Post-Remediation)
| Category | Count | Critical | High | Medium | Low |
|----------|-------|----------|------|--------|-----|
| Stubs/TODOs | 0 | 0 | 0 | 0 | 0 |
| Dead Code | 1 | 0 | 0 | 1 | 0 |
| Partially Wired | 2 | 0 | 1 | 1 | 0 |
| Interface Gaps | 1 | 0 | 0 | 1 | 0 |
| Dependency Gaps | 0 | 0 | 0 | 0 | 0 |

## Findings
### RESOLVED
- [x] WASM runtime placeholder removed; startup now initializes runtime state and returns readiness without blocking (`pkg/game/runtime_wasm.go`)
- [x] Anonymous waves/mechanics handlers now parse payloads (envelope or raw), validate/route, and update typed metrics (`pkg/app/handlers.go`)
- [x] Identity keypair storage now encrypted at rest with passphrase materialized in data dir and legacy plaintext migration (`pkg/app/murmur.go`)
- [x] Bootstrap resolver chain now wired in startup path; fallback discovery executes through configured chain (`pkg/app/murmur.go`, `pkg/networking/discovery/dht.go`)
- [x] Bootstrap verify key placeholder replaced with embedded key and strict key-size checks in resolvers (`pkg/networking/discovery/verify_key.go`, `pkg/networking/discovery/http_resolver.go`, `pkg/networking/discovery/ipfs_gateway_resolver.go`)
- [x] Nudge events now use typed EventBus payload (`EventNudge`) and dispatch path (`pkg/app/eventbus.go`, `pkg/app/nudges.go`)
- [x] Onboarding display-name callback now persists display name and signed identity declaration (`pkg/app/ui.go`)
- [x] Pulse Map TODO placeholders removed for bookmark/search naming behavior (`pkg/pulsemap/game.go`)
- [x] Browser transport shim TODO paths replaced with concrete bookkeeping behavior (`pkg/networking/transport/browser_host.go`)
- [x] Runtime metrics wiring added for bootstrap attempt/success and memory/cache/circuit gauges (`pkg/app/murmur.go`)

### OPEN (MEDIUM)
- [ ] Cross-compilation of desktop command targets for all non-host OS/ARCH combinations is blocked in this environment by upstream Ebitengine/OpenGL/Metal cross-compile constraints, not by local TODO/stub logic. Validation currently confirms host-native build and wasm command target only.
- [ ] `pkg/app/interfaces.go` remains an abstraction-only contract surface without direct injection wiring in `App`; this is architectural debt, not runtime breakage.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| TODO markers in test files | Excluded by remediation scope; production code TODO set is now zero. |
| Placeholder mentions in UI text comments | Cosmetic/documentation language; not execution-path stubs. |
| Generated protobuf `return nil` patterns | Generated code, not implementation gaps. |

## Current Risk Posture
The previously critical runtime and anonymous-mechanics implementation gaps are closed in production code paths. Remaining risk is primarily environmental (cross-compile toolchain/graphics backend) and architectural cleanup (unused app interfaces), not unfinished core logic.
