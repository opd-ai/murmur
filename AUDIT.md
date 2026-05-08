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
- Pending full test run and cross-target build checks (including wasm).
