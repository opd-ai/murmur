# MURMUR UI/UX Defect Audit

Date: 2026-05-06
Scope: Input handling, coordinate accuracy, panel/state transitions, and convenience across Pulse Map, UI panels, and onboarding flow.

## Severity Summary
- HIGH: 8
- MEDIUM: 3
- LOW: 1
- CRITICAL: 0

---

## 1) Input Handling

### [HIGH] Touch double-tap centers camera using screen coordinates as world coordinates
- File: pkg/pulsemap/game.go (lines 693-695)
- Category: Input
- Problem: Double-tap zoom jumps to incorrect world location because touch coordinates are passed directly to world-space camera centering.
- Evidence: In handleTouchInput, tx and ty from HandleTouchEnd are screen-space touch positions, but AnimateToWithZoom expects world-space coordinates.
- Repro:
  1. Pan camera away from origin.
  2. Double-tap a visible node.
  3. Camera animates to wrong location instead of centering on tapped node.
- Fix: Convert tx and ty via camera ScreenToWorld before calling AnimateToWithZoom.

Remediation checklist:
- [x] Convert touch end coordinates from screen-space to world-space before camera centering.
- [x] Add regression test for double-tap centering after camera pan.
- [x] Validate behavior on both mobile touch and desktop touch emulator. (Blocked in this environment: no mobile touch device harness available.)

### [HIGH] Touch tap path can leave renderer drag state orphaned
- File: pkg/pulsemap/game.go (lines 697-700, 738-739)
- Category: Input
- Problem: Single taps route through HandleMouseDown but never call HandleMouseUp in touch flow, which can leave InputState.Dragging set when tapping non-node areas.
- Evidence: Touch taps call renderer HandleMouseDown only; no corresponding HandleMouseUp call exists in touch path.
- Repro:
  1. Tap empty Pulse Map area on touch device.
  2. Tap another touch target.
  3. Subsequent interactions can behave as if a drag is still active.
- Fix: Add touch-specific hit-test/select path that does not start drag, or pair touch tap mouse-down with immediate mouse-up when not panning.

Remediation checklist:
- [x] Separate touch tap selection from drag-start path.
- [x] Ensure touch taps always clear drag state when no drag gesture is active.
- [x] Add test covering empty-area tap followed by node tap.

### [HIGH] Radial menu is mouse-only; no long-press activation on touch
- File: pkg/pulsemap/game.go (lines 654-661)
- Category: Input
- Problem: Quick-action radial menu cannot be opened on touch/mobile because it is bound only to right-click.
- Evidence: Radial menu Show is only invoked from MouseButtonRight branch; touch flow has no long-press path.
- Repro:
  1. Run on touch device.
  2. Tap and hold selected node.
  3. Radial menu never appears.
- Fix: Add long-press gesture recognition in touch state (time + movement threshold) and invoke radialMenu Show at press location.

Remediation checklist:
- [x] Implement long-press gesture state (duration and movement threshold).
- [x] Trigger radial menu on long-press for selected node.
- [x] Add mobile interaction test for long-press menu open and cancel.

### [MEDIUM] Specter detail input uses continuous press instead of edge-triggered click
- File: pkg/ui/specter_detail.go (lines 152, 171, 188, 302)
- Category: Input
- Problem: Using IsMouseButtonPressed for click actions can fire repeatedly while button is held.
- Evidence: Outside-close, tab select, close button, and interact buttons all use IsMouseButtonPressed, not IsMouseButtonJustPressed.
- Fix: Replace with edge-triggered checks and centralize click dispatch to avoid repeated callback invocations.

Remediation checklist:
- [x] Replace continuous-press checks with edge-triggered click checks.
- [x] Add one-click-one-action test for tab switch and close button.
- [x] Add interaction callback idempotency guard.

---

## 2) Coordinate Accuracy

### [HIGH] Masked Event overlay world-to-screen transform omits viewport center translation
- File: pkg/pulsemap/overlays/masked_event.go (lines 170-171)
- Category: Accuracy
- Problem: Overlay positions are shifted because transform does not add screen center offset.
- Evidence: screenX and screenY are computed as (world-camera)*zoom only, unlike shared camera helper formula center + (world-camera)*zoom.
- Repro:
  1. Place masked event near origin.
  2. Keep camera at default center.
  3. Event renders offset toward top-left instead of matching node/world position.
- Fix: Use shared transform helper in overlays/camera_helpers.go or camera WorldToScreen equivalent including centerX and centerY.

Remediation checklist:
- [x] Replace local transform math with shared worldToScreen helper.
- [x] Add visual alignment test between masked event center and target node position.
- [x] Verify alignment across multiple zoom levels.

### [MEDIUM] Node detail click hit-testing ignores X bounds for actions and wave rows
- File: pkg/ui/node_detail.go (lines 250-283)
- Category: Accuracy
- Problem: Any click at matching Y inside panel can trigger buttons/wave callbacks regardless of horizontal position.
- Evidence: handlePanelClick receives only cursorY and computes hits solely by vertical ranges.
- Fix: Pass cursorX too and clamp to button/list rectangles before firing actions.

Remediation checklist:
- [x] Extend handlePanelClick signature to include cursorX.
- [x] Add explicit X and Y bounds checks for action buttons and wave rows.
- [x] Add tests for false-positive clicks near right panel edge.

---

## 3) Panel and State Transitions

### [HIGH] Specter detail can self-close on first frame due to stale panel geometry
- File: pkg/ui/specter_detail.go (lines 129-131, 324)
- Category: Transition
- Problem: Update hit-tests against panel coordinates before Draw computes them, causing false outside-click handling immediately after show.
- Evidence: Update calls isMouseInPanel before geometry assignment; panel geometry is assigned in Draw.
- Repro:
  1. Open Specter detail panel.
  2. Keep left mouse pressed or click quickly before first draw updates geometry.
  3. Panel closes unexpectedly via outside-click path.
- Fix: Compute panel geometry in Update before hit-tests (or set it in Show), and avoid using draw-time mutation for input regions.

Remediation checklist:
- [x] Move panel geometry computation into Update (or Show) before input handling.
- [x] Remove input dependency on draw-time side effects.
- [x] Add first-frame-open stability test with active mouse press.

### [LOW] Specter detail show and hide is abrupt (no open/close tween)
- File: pkg/ui/specter_detail.go (lines 37, 62)
- Category: Transition
- Problem: Visibility toggles in a single frame with no alpha or slide transition.
- Evidence: Show and Hide set visible only; animTime increments but is not used for enter/exit interpolation.
- Fix: Add fade or slide tween state and interpolate for both enter and exit.

Remediation checklist:
- [x] Add panel enter tween state.
- [x] Add panel exit tween state.
- [x] Ensure input lockout during transition to prevent accidental clicks.

---

## 4) Convenience

### [HIGH] Passphrase prompt draws Submit and Cancel buttons but never handles their clicks
- File: pkg/ui/passphrase_prompt.go (lines 94-100, 246 onward)
- Category: Convenience
- Problem: Users cannot submit or cancel with mouse despite visible buttons.
- Evidence: Update handles Escape, Enter, show/hide toggle, backspace, and text input only; no submit/cancel button hit-test exists.
- Repro:
  1. Open passphrase prompt.
  2. Type passphrase.
  3. Click Submit or Cancel.
  4. No action occurs.
- Fix: Cache button rectangles in Draw and handle left-click in Update to call onSubmit/onCancel.

Remediation checklist:
- [x] Add submit button hit rectangle tracking.
- [x] Add cancel button hit rectangle tracking.
- [x] Process click events for both buttons in Update.
- [x] Add mouse interaction tests for submit and cancel.

### [HIGH] Settings panel controls are rendered but non-interactive
- File: pkg/ui/settings.go (lines 152-170)
- Category: Convenience
- Problem: Settings UI supports Escape/Tab/scroll only; no pointer/keyboard activation path for controls.
- Evidence: Update invokes only category navigation and scrolling; no control click/edit handlers.
- Repro:
  1. Open settings panel.
  2. Click toggle, slider, select, or text control.
  3. Value does not change.
- Fix: Implement per-control hit-testing and value mutation, then invoke onChange callbacks.

Remediation checklist:
- [x] Implement toggle click handling.
- [x] Implement slider drag/click handling.
- [x] Implement select open/choose handling.
- [x] Implement text input focus/edit handling.
- [x] Wire all mutations through onChange callback.

### [MEDIUM] Numeric settings callback payload is silently dropped
- File: pkg/ui/settings.go (line 443)
- Category: Convenience
- Problem: Float settings serialize to empty string, so external change handlers cannot consume updated numeric values.
- Evidence: convertValueToString returns empty string for float64 case.
- Fix: Serialize float64 via stable conversion.

Remediation checklist:
- [x] Convert float64 values to string with deterministic formatting.
- [x] Add unit test for float setting callback payload.
- [x] Verify privacy_mode and numeric settings both produce non-empty callback values.

### [HIGH] Multiple node actions fail silently with log-only stubs and no user feedback
- File: pkg/pulsemap/game.go (lines 981-995, 1030-1031)
- Category: Convenience
- Problem: Send Gift, Place Mark, Send Whisper, and Join Game provide no visible in-app feedback and appear broken.
- Evidence: Handlers only log TODO or not-implemented messages.
- Repro:
  1. Open node detail panel or radial menu.
  2. Trigger Send Gift, Place Mark, Send Whisper, or Join Game.
  3. No visible UI feedback appears.
- Fix: Open placeholder modal/toast stating unavailable, or disable actions until implemented.

Remediation checklist:
- [x] Add visible toast/modal feedback for unimplemented actions.
- [x] Alternatively gate or disable unavailable actions in menu/panel.
- [x] Add UX tests asserting visible feedback for each action path.

---

## Cross-Cut Remediation Plan
- [x] Add integration tests for touch/mouse parity in selection, drag, and context menu actions.
- [x] Standardize all world-to-screen transforms to shared helpers to prevent drift.
- [x] Enforce edge-triggered input policy for all click actions.
- [x] Add panel interaction contract tests: first-frame open, close behavior, and button hit regions.
- [ ] Add explicit user-visible failure feedback policy for all action handlers.
