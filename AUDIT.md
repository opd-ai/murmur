# SECURITY AUDIT — 2026-05-08

## Project Security Profile
MURMUR is a decentralized P2P Go application (desktop/mobile/TUI plus bootstrap HTTP service). Primary trust boundaries are: (1) unauthenticated network input from libp2p/GossipSub peers, (2) unauthenticated HTTP input to bootstrap and optional health endpoints, (3) local operator input via CLI flags/env vars. The project claims privacy-by-structure, cryptographic identity, and signed bootstrap discovery. Authentication is predominantly cryptographic verification (Ed25519 signatures) rather than user session auth.

## Security Surface Inventory
| Package | HTTP Handlers | DB Queries | Exec Calls | File I/O | Crypto | Auth |
|---------|--------------:|-----------:|-----------:|---------:|-------:|:----:|
| cmd/bootstrap | 7 | 0 | 0 | 4 | 10 | ❌ |
| pkg/networking/health | 5 | 0 | 0 | 0 | 0 | ❌ |
| pkg/networking/discovery | 4 | 5 | 0 | 3 | 22 | ✅ |
| pkg/identity | 0 | 0 | 13 | 11 | 242 | ✅ |
| pkg/app | 0 | 1 | 0 | 6 | 24 | ✅ |
| pkg/store | 0 | 29 | 0 | 1 | 1 | ❌ |
| pkg/content/waves | 0 | 0 | 0 | 0 | 57 | ✅ |
| pkg/anonymous/shroud | 0 | 0 | 0 | 0 | 45 | ✅ |
| pkg/tunneling/initiator | 3 | 0 | 0 | 0 | 6 | ❌ |
| pkg/tunneling/relay | 0 | 0 | 0 | 0 | 0 | ✅ |

Supporting baseline metrics (`go-stats-generator`, measured 2026-05-08): 57,187 LOC, 1,657 functions, 79 packages, 393 Go files.

## Dependency Vulnerability Check
- `govulncheck ./...` was attempted but failed in this environment due to missing X11 headers required by Ebitengine transitive build (`X11/Xlib.h`), so full call-graph vulnerability confirmation is blocked.
- GitHub issue/advisory search for this repository found no open security issues or published repository advisories.
- `gh-advisory-database` reported vulnerabilities for `github.com/hashicorp/vault v1.21.4`; this codebase imports only `github.com/hashicorp/vault/shamir`, so exploitability for the reported Vault server issues is currently unproven and requires targeted dependency scoping/upgrade review.

## Findings
### CRITICAL
No confirmed critical findings after data-flow validation.

### HIGH
No confirmed high findings after data-flow validation.

### MEDIUM
- [ ] Unbounded remote bootstrap response reads can cause memory-exhaustion DoS — `pkg/networking/discovery/http_resolver.go:52`, `pkg/networking/discovery/ipfs_gateway_resolver.go:101`, `pkg/networking/discovery/ipfs_gateway_resolver.go:132` — **Evidence/data flow:** external HTTP responses (Gist/Pages/IPFS) are read with `io.ReadAll(resp.Body)` without byte caps; this path is reachable when remote bootstrap is enabled in `pkg/app/murmur.go:567-572` — **Impact:** a malicious/compromised bootstrap endpoint (or oversized response) can force large allocations and crash or degrade nodes during bootstrap — **Remediation:** replace `io.ReadAll` with bounded reads (`io.LimitReader` + explicit maximum, e.g., 1 MiB), reject over-limit bodies before JSON parsing, and validate with `go test ./pkg/networking/discovery/...` plus a test case serving oversized bodies.

### LOW
- [ ] Bootstrap HTTP endpoint exposes operational metadata without authentication and with wildcard CORS default — `cmd/bootstrap/main.go:162`, `cmd/bootstrap/main.go:168`, `cmd/bootstrap/main.go:284-288`, `cmd/bootstrap/main.go:318-323`, `cmd/bootstrap/main.go:86` — **Evidence/data flow:** unauthenticated remote clients can call `/health`; default config binds HTTP service and sets `allow-origin` to `*`, and health payload includes `state_dir` and node metadata — **Impact:** reconnaissance and environment disclosure for publicly reachable bootstrap nodes — **Remediation:** make `--allow-origin` opt-in with explicit origins, remove `state_dir` from external health responses (or gate detailed health behind local-only/authenticated mode), and verify via integration test that default health output excludes local filesystem paths.

## False Positives Considered and Rejected
| Candidate Finding | Reason Rejected |
|-------------------|----------------|
| Command injection in Linux share flow (`pkg/identity/share_linux.go:19,24`) | Shell invocation exists, but data flow for `ShareText` is `Invitation.EncodeURI()` Base64URL output (`pkg/identity/invitation.go:100-124`), which does not carry shell metacharacters needed for practical command injection in current call paths. |
| Weak RNG via `math/rand` in PEX (`pkg/networking/discovery/pex.go:14,196`) | Used only for peer-sample shuffling, not cryptographic key/token generation; no security boundary depends on unpredictability here. |
| Hardcoded key in `pkg/networking/discovery/verify_key.go:11-16` | Embedded value is a public verification key by design, not a secret; threat model expects public-key distribution in clients. |
