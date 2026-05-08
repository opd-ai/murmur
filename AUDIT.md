# AUDIT

Date: 2026-05-07
Scope: Ebitengine UI/UX remediation from latest audit findings.

## Changes Applied

1. Touch target baseline raised in shared UI theme.
- File: pkg/ui/panel.go
- Change: DefaultTheme ButtonHeight changed from 36 to 44.
- Outcome: Shared panel buttons now meet 44px minimum target height.

2. Onboarding button baseline raised.
- File: pkg/onboarding/screens/helpers.go
- Change: DefaultButtonStyle Height changed from 40 to 44.
- Outcome: Onboarding CTA/back actions now meet mobile touch target guidance.

3. Passphrase prompt input and controls hardened.
- File: pkg/ui/passphrase_prompt.go
- Changes:
  - Backspace now removes the last rune (UTF-8 safe) instead of byte slicing.
  - Button draw height now uses theme ButtonHeight.
- Outcome: No UTF-8 corruption while deleting; button hit area tracks global touch target policy.

4. Recovery screen input and layout hardened.
- File: pkg/onboarding/screens/recovery_screen.go
- Changes:
  - Added explicit back button dimensions (120x44) and reused them for hit tests and drawing.
  - Mnemonic and passphrase backspace paths made UTF-8 rune-safe.
  - Added clipped masked passphrase rendering with ellipsis to prevent overflow in fixed-width input box.
- Outcome: Correct Unicode deletion behavior, consistent button hit boxes, no passphrase text bleed.

5. Compose/search small-screen layout overflow fixed.
- Files:
  - pkg/ui/compose.go
  - pkg/ui/search.go
- Changes:
  - Compose panel width now clamps to viewport bounds even on very narrow windows.
  - Search bar width now clamps to viewport-safe max width on narrow windows.
- Outcome: Controls remain on-screen and centered at small resolutions.

6. Council proposal/member scrolling corrected.
- Files:
  - pkg/ui/councils.go
  - pkg/ui/councils_draw.go
- Changes:
  - Added separate proposal selection index (`selectedProposal`) from scroll offset.
  - Proposal scroll now follows selected row and clamps by visible row count.
  - Member list rendering now respects scroll window and active-member filtering.
- Outcome: No empty trailing viewport from over-scroll; keyboard navigation and highlighting stay coherent.

7. Hot-path formatting cleanup in territory overview.
- File: pkg/ui/territory_overview.go
- Changes:
  - Replaced per-frame fmt.Sprintf usage in draw/status formatting with strconv-based formatting helpers.
- Outcome: Reduced hot-path formatting overhead in Draw.

## Security/Privacy Notes
- No cryptographic or networking behavior changed.
- Changes are UI/input-only.

## Validation Status
- Tests:
  - `go test ./...` passed.
- Native desktop build:
  - `GOOS=linux GOARCH=amd64 go build ./cmd/murmur` passed.
- Cross-target compile coverage:
  - `GOOS=<target> GOARCH=<arch> CGO_ENABLED=0 go build -tags noebiten ./cmd/murmur`
    passed for:
    - linux/amd64
    - linux/arm64
    - darwin/amd64
    - darwin/arm64
    - windows/amd64
- WASM build:
  - `GOOS=js GOARCH=wasm CGO_ENABLED=0 go build ./cmd/wasm` passed.

## Update - 2026-05-07 (CI Native GUI Build Policy)

### Change Summary
- Updated CI artifact matrix to run native builds on each platform runner (`ubuntu-latest`, `macos-latest`, `windows-latest`).
- Removed cross-compilation in the CI matrix (`GOOS/GOARCH` target overrides removed).
- Removed `noebiten` build path from artifact packaging; executables are now built from `./cmd/murmur` with Ebitengine GUI enabled (`CGO_ENABLED=1`).

### Security/Operational Impact
- Improves release confidence by validating real platform toolchains and GUI-linked builds instead of headless/cross-compiled substitutes.
- Reduces mismatch risk between released binaries and runtime graphics/input dependencies.
- Linux runner explicitly installs required GUI system libraries to ensure deterministic desktop build success.

### Follow-up
- Keep wasm publishing in dedicated web workflows; CI executable artifact matrix remains native desktop only.

## Update - 2026-05-08 (Release Action Migration)

### Change Summary
- Replaced the release creation action in `.github/workflows/build.yml` from `softprops/action-gh-release@v3` to `ncipollo/release-action@v1`.
- Mapped `files` to `artifacts` and `generate_release_notes` to `generateReleaseNotes` to preserve release packaging behavior.

### Security/Operational Impact
- Reduces dependency on an older release publishing action and standardizes release automation on the selected maintained action.
- Scope is limited to CI workflow configuration; no runtime networking, cryptography, storage, or application logic changed.

### Validation Status
- Prior baseline commands executed before change:
  - `make lint` failed in this environment due to missing `X11/Xlib.h` during Ebitengine GLFW native compilation.
  - `make test` partially passed but failed in GUI-linked packages for the same missing Linux X11 headers.
  - `make build` failed for the same missing X11 development headers.
- Workflow syntax validated post-change:
  - `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/build.yml")'` passed.
- Note: The baseline lint/test/build failures above are environment-specific (missing Linux X11 development headers for Ebitengine native GUI compilation) and are unrelated to the release-action migration itself.

## Update - 2026-05-08 (gomobile Entrypoint Fix)

### Change Summary
- Added `cmd/murmur-mobile` as a dedicated mobile application entrypoint.
- Added a mobile-only blank import of `golang.org/x/mobile/app` so `gomobile` recognizes the package as an app target.
- Updated `scripts/build-mobile.sh` to build `github.com/opd-ai/murmur/cmd/murmur-mobile`.

### Security/Operational Impact
- No runtime cryptography, networking protocol behavior, or persistence logic changed.
- Impact is limited to mobile build tooling and dependency resolution for `golang.org/x/mobile`.

### Validation Status
- `go list -tags=android -f '{{join .Imports "\n"}}' ./cmd/murmur-mobile` includes `golang.org/x/mobile/app`.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/murmur` still fail in this Linux environment because Ebitengine's native GLFW build cannot find `X11/Xlib.h`; this pre-existing environment issue is unrelated to the mobile entrypoint fix.
