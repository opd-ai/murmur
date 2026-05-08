# GAPS

Date: 2026-05-07
Context: Post-remediation gap tracking for Ebitengine UI/UX audit issues.

## Closed Gaps

1. Shared button hit target too small (36px) in default theme.
- Status: Closed
- Resolution: Raised to 44px in pkg/ui/panel.go.

2. Onboarding default button hit target too small (40px).
- Status: Closed
- Resolution: Raised to 44px in pkg/onboarding/screens/helpers.go.

3. Councils proposal list scroll/selection coupling caused poor UX and over-scroll.
- Status: Closed
- Resolution: Split selection from scroll and clamp against visible rows.

4. UTF-8 backspace corruption in passphrase/recovery inputs.
- Status: Closed
- Resolution: All affected backspace paths now operate on rune slices.

5. Compose/Search narrow-screen width overflow.
- Status: Closed
- Resolution: Width clamped to viewport-safe bounds in compose/search layout code.

6. Recovery key-file masked passphrase overflow.
- Status: Closed
- Resolution: Added width-based clipping with ellipsis.

7. Hot-path fmt.Sprintf in territory overview draw path.
- Status: Closed
- Resolution: Replaced with strconv/string formatting helpers.

## Open Gaps

None identified from the targeted audit findings.

## Verification Gap

Final verification is pending:
- Full test suite pass
- Multi-target compile pass including wasm

This section will be updated after validation execution.
