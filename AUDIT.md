# MURMUR UI/UX Defect Audit

**Audit Date**: 2026-05-06
**Auditor**: GitHub Copilot (automated static analysis)
**Scope**: `pkg/pulsemap/` (game.go, interaction/, layout/, rendering/, overlays/), `pkg/ui/`, `pkg/onboarding/flow/`, `pkg/onboarding/screens/`
**Methodology**: Full read of every non-stub, non-test `.go` file in scope; no runtime execution.

---

## Table of Contents

- [1. Input Handling](#1-input-handling)
- [2. Coordinate Accuracy](#2-coordinate-accuracy)
- [3. Panel & State Transitions](#3-panel--state-transitions)
- [4. Convenience](#4-convenience)
- [Files Evaluated](#files-evaluated)

---

## 1. Input Handling

---

### [CRITICAL][FIXED] Compose panel Submit and Cancel buttons have no mouse click handler

- **File**: `pkg/ui/compose.go` (lines 116–233 `Update()`, lines 375–385 `drawButtons()`)
- **Category**: Input
- **Problem**: `drawButtons()` renders "Cancel" and "Submit" buttons via `DrawCancelSubmitButtons`, but `Update()` never reads `ebiten.CursorPosition()` or calls `inpututil.IsMouseButtonJustPressed`; the only submission path is keyboard Enter/Escape.
- **Evidence**: `Update()` contains no `IsMouseButtonJustPressed` call. `DrawCancelSubmitButtons` returns `cancelX, submitX, buttonY` which `Update()` never stores or tests.
- **Reproducible trigger**: Open compose panel (Ctrl+N), click the "Submit" button with the mouse → nothing happens. Wave is never submitted without pressing Enter.
- **Fix**: In `Update()`, after `handleTextInput()`, read cursor position and test against the computed button rects:
  ```go
  // After existing key handling in Update():
  if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
      cx, cy := ebiten.CursorPosition()
      submitX := p.x + p.width - p.theme.Padding - 100
      cancelX := p.x + p.theme.Padding
      buttonY := p.y + p.height - p.theme.Padding - p.theme.ButtonHeight
      if cy >= buttonY && cy <= buttonY+p.theme.ButtonHeight {
          if cx >= submitX && cx <= submitX+100 {
              p.submit()
          } else if cx >= cancelX && cx <= cancelX+80 {
              p.visible = false
          }
      }
  }
  ```
- **Remediation checklist**:
  - [x] Store panel top-left (`p.x`, `p.y`) so `Update()` can compute button rects without recalculating position independently of `Draw()`.
  - [x] Add `handleMouseClick()` sub-method called from `Update()`.
  - [x] Add test: simulate left-click at submit button coords and assert `onSubmit` was called.

---

### [HIGH][FIXED] Radial menu is never wired into the game loop

- **File**: `pkg/pulsemap/game.go` (entire file, no `RadialMenu` reference); `pkg/ui/radial_menu.go`
- **Category**: Input
- **Problem**: `RadialMenu` is fully implemented in `pkg/ui/radial_menu.go` but `game.go` never instantiates, updates, or draws it. Right-click and long-press do nothing on any node.
- **Evidence**: `grep "RadialMenu\|radialMenu\|MouseButtonRight"` returns zero hits in `game.go`.
- **Reproducible trigger**: Right-click any node in the Pulse Map → no context menu appears.
- **Fix**:
  1. Add `radialMenu *ui.RadialMenu` field to `Game`.
  2. Instantiate in `NewGame` with action callbacks.
  3. In `handleDragging()`, handle `MouseButtonRight` just-pressed: convert cursor to screen coords, call `g.radialMenu.Show(float64(mx), float64(my), selectedNodeID)`.
  4. Call `g.radialMenu.Update()` and `g.radialMenu.Draw(screen)` each tick with appropriate panel-priority ordering.
- **Remediation checklist**:
  - [x] Add `radialMenu` field to `Game`.
  - [x] Wire `OnAction` callbacks for all 6 actions (`ActionComposeWave`, `ActionSendGift`, etc.).
  - [x] Handle `ebiten.MouseButtonRight` just-pressed in `handleDragging()`.
  - [x] Guard: do not open radial menu while another modal panel is visible.
  - [x] Add radial menu to `updateActivePanels()` priority chain.
  - [x] Draw radial menu above graph layer, below compose panel.

---

### [HIGH] Touch input (`TouchState`) never called from the game loop

- **File**: `pkg/pulsemap/game.go` (`Update()` method); `pkg/pulsemap/interaction/touch.go`
- **Category**: Input
- **Problem**: `touch.go` provides a complete `TouchState` type with `HandleTouchStart`, `HandleTouchMove`, `HandleTouchEnd`, but `game.go` never calls any Ebitengine touch API (`ebiten.AppendTouchIDs`, `ebiten.TouchPosition`) and never instantiates `TouchState`. Mobile builds have zero touch support.
- **Evidence**: `grep "Touch\|touch\|TouchIDs\|TouchState" game.go` → empty.
- **Reproducible trigger**: Run on a touch-screen device; all touch gestures (pan, pinch-zoom, tap, double-tap) are silently ignored.
- **Fix**: Add touch handling to `Update()`:
  ```go
  // In handleDragging() or a new handleTouchInput() method:
  ids := ebiten.AppendTouchIDs(nil)
  for _, id := range ids {
      x, y := ebiten.TouchPosition(id)
      // route to g.touchState.HandleTouchStart/Move/End
  }
  ```
  Map `TouchState` results (pan delta, zoom factor, tap position) to the same camera and node-selection code paths already used by mouse.
- **Remediation checklist**:
  - [x] Add `touchState *interaction.TouchState` field to `Game`, initialised in `NewGame`.
  - [x] Add `handleTouchInput()` called from `Update()`.
  - [x] Implement frame-to-frame `AppendTouchIDs` diff to detect start/end events.
  - [x] Map pan delta → `camera.Pan`, zoom factor → `camera.Zoom`, tap → `renderer.HandleMouseDown`.
  - [x] Ensure touch and mouse paths both call `input.SelectNode`.

---

### [HIGH] Global keyboard shortcuts active while text panels have focus

- **File**: `pkg/pulsemap/game.go` (lines 200–215 `Update()`, `handleNavigationHotkeys()`)
- **Category**: Input
- **Problem**: `handleNavigationHotkeys()` is guarded only when `panelActive` is true, where `panelActive` is `searchBar.Visible() || nodeDetailPanel.Visible() || composePanel.Visible()`. However, `viewportControls.Update()` runs unconditionally (line 236), so pressing H or Home while typing brings any key that happens to be those characters into the navigation handler. More critically, Ctrl+N (compose toggle) and Ctrl+F (search toggle) fire synchronously before the compose/search panels' own `Update()` consumes `AppendInputChars`, allowing the "N" or "F" character to appear in the text field and simultaneously toggle the panel.
- **Evidence**: Lines 200–215: `panelActive` check excludes `viewportControls`. Lines 201–206: `handleComposePanelToggle` and `handleSearchBarToggle` are called before `updateActivePanels`.
- **Reproducible trigger**: With compose panel open, type "n" → panel closes (Ctrl not required because `inpututil.IsKeyJustPressed(ebiten.KeyN) && ctrlPressed` guards it, but `AppendInputChars` also appends "n" to content after the toggle fires).
  With search bar open, press Home → camera re-centers to self node.
- **Fix**:
  1. Move `handleComposePanelToggle` and `handleSearchBarToggle` to run only when no panel is currently active.
  2. Check a dedicated `textInputActive()` predicate before calling `handleNavigationHotkeys`.
  3. Move `viewportControls.Update()` inside the existing `!panelActive` guard, or make its click handler test only mouse input (it already does — but viewport controls should be excluded from keyboard shortcuts when text is active).
- **Remediation checklist**:
  - [x] Refactor `Update()` ordering: evaluate `panelActive` before any hotkey dispatch.
  - [x] Add `func (g *Game) textInputActive() bool` that returns true when compose, search, passphrase, or any text-input panel is visible.
  - [x] Guard `handleComposePanelToggle` and `handleSearchBarToggle` with `!g.textInputActive()`.
  - [x] Confirm `ebiten.AppendInputChars` is consumed (returned slice discarded) by whichever panel is active so the same chars don't appear in multiple panels.

---

### [HIGH][FIXED] `isDragging` state orphaned on mouse button released outside window

- **File**: `pkg/pulsemap/game.go` (lines 525–549 `handleDragging()`)
- **Category**: Input
- **Problem**: `isDragging` is set true on `MouseButtonLeft` just-pressed. It is cleared only on `MouseButtonLeft` just-released. If the user presses and drags the mouse out of the window and releases outside, Ebitengine never reports the just-released event back to `Update()`, leaving `isDragging = true` permanently. The camera will then pan every frame the user moves the mouse inside the window even without holding any button.
- **Evidence**: `handleDragging()` has no `ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)` check before updating the drag; the press-check returns false but the move block (`if g.isDragging && ebiten.IsMouseButtonPressed`) does guard correctly on the update. However, `isDragging` is never cleared by the focus-loss path at all.
- **Reproducible trigger**: Left-click, hold, drag outside window, release outside, re-enter window without pressing — camera continues panning.
- **Fix**: Add a safeguard: if `isDragging` is true but `ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)` is false, clear the drag state immediately:
  ```go
  if g.isDragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
      g.isDragging = false
      g.renderer.HandleMouseUp()
  }
  ```
- **Remediation checklist**:
  - [x] Add the guard at the top of `handleDragging()`.
  - [x] Apply same pattern to `InputState.Dragging` inside `renderer.HandleMouseDown` / `HandleMouseUp`.
  - [x] Write a test that calls `Update()` with `Dragging=true` but `MouseButtonLeft` not pressed and asserts `isDragging` becomes false.

---

### [HIGH] Passphrase rendered text is always blank due to broken masking loop

- **File**: `pkg/ui/passphrase_prompt.go` (lines 170–178 in `Draw()`)
- **Category**: Input
- **Problem**: The masking loop replaces the passphrase with bullet characters, but the loop is written incorrectly:
  ```go
  passText = string(make([]byte, len(passText)))
  for i := range passText {
      passText = passText[:i] + "•"  // ← truncates to i+1 rune after first iteration
  }
  ```
  `make([]byte, len(passText))` creates a zero byte slice of length `len(passText)` (byte count, not rune count), and the inner assignment `passText = passText[:i] + "•"` replaces everything from index `i` onward with a single bullet on each iteration. The final result is either a single "•" or an empty string, never the correct `len(rune) × "•"` sequence. The passphrase still appears blank in the input box.
- **Evidence**: Lines 172–177: the loop overwrites `passText` to `passText[:i] + "•"` each iteration.
- **Reproducible trigger**: Open passphrase prompt; type any text; the input box shows blank (single bullet is 3 UTF-8 bytes overflowing the box or simply invisible against background).
- **Fix**:
  ```go
  passText = strings.Repeat("•", utf8.RuneCountInString(p.passphrase))
  ```
- **Remediation checklist**:
  - [x] Import `strings` and `unicode/utf8` in `passphrase_prompt.go`.
  - [x] Replace the broken loop with `strings.Repeat("•", utf8.RuneCountInString(p.passphrase))`.
  - [x] Add unit test: set passphrase to "abc", call Draw, assert rendered text is "•••".

---

### [HIGH] `showPass` field is declared but the show/hide toggle is never implemented

- **File**: `pkg/ui/passphrase_prompt.go` (line 29 field declaration; `Update()` has no toggle)
- **Category**: Input / Convenience
- **Problem**: `PassphrasePromptPanel` declares `showPass bool` which is read in `Draw()` to decide whether to mask the passphrase. However, `Update()` contains no code to toggle `showPass` (no button click handler, no keyboard shortcut). `showPass` is always its zero value (`false`), so the passphrase is always masked and the field is dead code.
- **Evidence**: `grep "showPass"` → only two hits: field declaration (line 29) and read in `Draw()` (line 172). Zero writes. `Update()` has no `IsMouseButtonJustPressed` call.
- **Reproducible trigger**: There is no way for a user to reveal their passphrase while typing to confirm it.
- **Fix**:
  1. Draw a small "Show/Hide" toggle button or eye icon in `Draw()` to the right of the input box.
  2. In `Update()`, detect a click on that button and toggle `p.showPass = !p.showPass`.
- **Remediation checklist**:
  - [x] Draw a "Show" toggle button at `(inputX+inputWidth-40, inputY)`.
  - [x] In `Update()` add mouse-click handler for that button region.
  - [x] Guard: zero `p.passphrase` from `strings.Builder` after `p.visible = false` to avoid leaving plaintext passphrase in memory.

---

### [MEDIUM][FIXED] Double-tap detection has no single-tap debounce window

- **File**: `pkg/pulsemap/interaction/touch.go` (lines 145–165 `HandleTouchEnd()`)
- **Category**: Input
- **Problem**: When a tap is detected, `isTap = true` is returned immediately. If the user taps twice rapidly, the first tap fires a single-tap action (node selection) before `HandleTouchEnd` can check the inter-tap interval on the second touch end. This means double-taps always fire two single-tap events in addition to the double-tap event, potentially opening node detail and centering the camera simultaneously.
- **Evidence**: `HandleTouchEnd` returns `isTap = true` on the first touch end regardless of whether a second touch end is imminent. There is no deferred single-tap timer.
- **Fix**: Return `isTap = false` on the first tap if `lastTapTime > 0` and the inter-tap interval so far is within `DoubleTapMaxInterval`. Delay the single-tap action by deferring it using a tick counter, firing only if a second tap does not arrive within the window.
- **Remediation checklist**:
  - [x] Add `pendingTapX, pendingTapY float64` and `pendingTapTick int64` fields to `TouchState`.
  - [x] On first tap, store position/tick but return `isTap = false`; on next `Update()` tick check if double-tap window expired and emit single-tap then.
  - [x] Update `DoubleTapMaxInterval` doc comment to reflect the deferred model.

---

### [MEDIUM][FIXED] `processBackspace` in compose panel repeats on backspace held but uses inconsistent condition

- **File**: `pkg/ui/compose.go` (lines 169–177 `processBackspace()`)
- **Category**: Input
- **Problem**: The repeat condition is `!inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) <= 20`. The `&&` short-circuits: if backspace was just pressed (`IsKeyJustPressed = true`), the first term is `false`, so the whole condition is false and the function returns early on the very first press. This inverts the logic — backspace deletes only after the key has been held for 20+ ticks, not on first press.
- **Evidence**: Line 169: `if !inpututil.IsKeyJustPressed(...) && KeyPressDuration(...) <= 20 { return }`. The guard returns early when either (a) the key was not just pressed AND hold duration ≤ 20, or (b) … the logical intent appears to be "skip if not just pressed and also not held long enough", but the actual effect skips the first press.
- **Fix**: Change to:
  ```go
  if !inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && inpututil.KeyPressDuration(ebiten.KeyBackspace) < 20 {
      return
  }
  ```
  and use `< 20` (exclusive) so the initial just-pressed event is not blocked by the duration guard.
- **Remediation checklist**:
  - [x] Fix operator from `<=` to `<` and verify first-press deletion works in a unit test.
  - [x] Add test: single tick with `KeyJustPressed=true`, assert one character deleted.

---

## 2. Coordinate Accuracy

---

### [HIGH][FIXED] Node hit-test radius not scaled by current zoom level

- **File**: `pkg/pulsemap/rendering/renderer.go` (lines 638–645 `hitTestNodes()`)
- **Category**: Accuracy
- **Problem**: `hitTestNodes` converts the click to world coordinates correctly, then computes `radius = float64(computeNodeRadius(style)) * 1.5 / r.camera.Scale`. Dividing the visual radius by `Scale` converts it from screen pixels to world units, which is correct in principle. However `computeNodeRadius` returns a value in screen pixels only at `Scale=1`. At high zoom (Scale=5), the world-space radius is `visualRadius * 1.5 / 5`, which is much smaller than the visible node — clicks on the node rim miss. At low zoom (Scale=0.1), the world-space hit radius balloons to `visualRadius * 15`, causing large invisible hit zones that overlap neighbouring nodes.
- **Evidence**: `radius := float64(computeNodeRadius(style)) * 1.5 / r.camera.Scale` — the 1.5 fudge factor does not compensate adequately across the full `[0.1, 5.0]` scale range.
- **Reproducible trigger**: Zoom to scale ≥ 3.0 (Micro view), click the edge of a node → miss. Zoom to scale ≤ 0.2 (Macro view), click between two nodes → wrong node selected.
- **Fix**: Clamp the world-space hit radius to a minimum that matches the visual size, or compute it purely in world units derived from the base node radius constant:
  ```go
  const baseHitRadius = 8.0 // world units, matches rBase in computeNodeRadius
  radius := math.Max(float64(computeNodeRadius(style))/r.camera.Scale, baseHitRadius)
  ```
- **Remediation checklist**:
  - [x] Audit `computeNodeRadius` to confirm its output units are screen pixels at Scale=1.
  - [x] Replace the fudge-factor calculation with a world-space constant floored.
  - [x] Add table-driven test for hit detection at Scale=0.1, 1.0, 5.0.

---

### [HIGH][FIXED] Duplicate (stale) amplification trail rendering path not removed

- **File**: `pkg/pulsemap/rendering/renderer.go` (lines 689–725 `drawAmplificationTrails()`; lines 735–765 `accumulateAmplificationTrails()`)
- **Category**: Accuracy
- **Problem**: The file contains two independent methods that render amplification trails: `drawAmplificationTrails` (old per-frame draw, called from `drawEdges` indirectly) and `accumulateAmplificationTrails` (new batched path, called from `Draw()`). The old `drawAmplificationTrails` is still compiled and called on the same `r.graphLayer`, meaning trails are drawn twice each frame using positions snapshotted at different points during the same `Draw()` call (the `positions` map passed differs between the two call sites). This causes visual doubling and potential one-frame-stale endpoint coordinates on the first draw path.
- **Evidence**: `renderer.go` line 689 defines `drawAmplificationTrails`; line 735 defines `accumulateAmplificationTrails`. Both iterate `r.amplificationTrails`. Both call `r.transformAndCullLine`. The `Draw()` method calls `accumulateAmplificationTrails` (line 339), but `drawAmplificationTrails` is also reachable from `drawEdges` → `iterateEdges` does not call it, so it is currently dead code — but it compiles and creates maintenance confusion. Confirm by checking call graph.
- **Fix**: Remove `drawAmplificationTrails` entirely (it is superseded by `accumulateAmplificationTrails`). Verify no path calls it (grep for callers).
- **Remediation checklist**:
  - [x] `grep -n "drawAmplificationTrails"` to enumerate call sites.
  - [x] Delete the `drawAmplificationTrails` method body and its comment block.
  - [x] Run `go vet ./...` to confirm no remaining references.

---

### [MEDIUM][FIXED] Node detail panel button click region not adjusted for slide animation during motion

- **File**: `pkg/ui/node_detail.go` (lines 202–208 `calculatePanelPosition()`, lines 237–243 `handlePanelClick()`)
- **Category**: Accuracy
- **Problem**: `calculatePanelPosition()` computes `panelX` including the `slideOffset` contribution (`screenW - panelW + int(slideOffset*panelW)`). `isMouseOverPanel` and `handlePanelClick` both use the stored `panelX`, so they are consistent with Draw. However, `calculatePanelPosition` is called once per `Update()` tick, and `Draw()` uses the same stored fields. If `Draw()` is called multiple times per tick (which Ebitengine can do in some configurations) the stored `panelX` may lag. More concretely: the button Y-offset in `handlePanelClick` is hardcoded as `p.panelY + nodeDetailPadding + 100`, which is not recalculated from the *current* animated `panelY`. During the slide-in animation, `panelY` is fixed but the intent comment says `100` is meant to skip the header; if the header height ever changes, this magic constant will be wrong.
- **Evidence**: Line 240: `buttonY := p.panelY + nodeDetailPadding + 100` — magic constant not derived from any named layout constant.
- **Fix**: Replace the magic `100` with a sum of named constants: `nodeDetailPadding + int(resonanceHeight) + sectionSpacing`, and derive this from the same layout code used in `Draw()`.
- **Remediation checklist**:
  - [x] Define `nodeDetailHeaderHeight = 80` and `nodeDetailResonanceHeight = 40` constants.
  - [x] Compute `buttonY` from those constants in `handlePanelClick`.
  - [x] Add a test asserting button-click detection matches draw layout at `slideOffset=0`.

---

### [MEDIUM][FIXED] Wave list items in node detail panel are drawn but never hit-testable

- **File**: `pkg/ui/node_detail.go` (lines 237–243 `handlePanelClick()`; lines 391–422 `drawRecentWaves()`)
- **Category**: Accuracy
- **Problem**: `drawRecentWaves()` renders scrollable Wave list items with full background rectangles, content, and timestamps. `handlePanelClick()` only checks for clicks in the 4-button zone (`buttonY` to `buttonY + 4*buttonHeight + 3*spacing`). Any click on a Wave list item falls through to `return true` without dispatching `callbacks.OnViewWave`. The Wave list is display-only; tapping items does nothing.
- **Evidence**: `handlePanelClick` only calls `handleButtonClick(buttonIndex)` for the action-button zone; no code checks `wavesY` region.
- **Fix**: After the button-zone check, add a wave-item hit test:
  ```go
  wavesStart := buttonY + nodeDetailButtonHeight*4 + nodeDetailButtonSpacing*3 + 20 + 22
  for i, wave := range visibleWaves {
      itemY := wavesStart + i*nodeDetailListItemHeight
      if cursorY >= itemY && cursorY < itemY+nodeDetailListItemHeight {
          if p.callbacks.OnViewWave != nil {
              p.callbacks.OnViewWave(wave.WaveID)
          }
          return true
      }
  }
  ```
- **Remediation checklist**:
  - [x] Add `WaveID string` field to `WaveInfo` (currently missing).
  - [x] Implement hit-test region for each visible wave item in `handlePanelClick`.
  - [x] Ensure `waveScroll` offset is applied to `visibleWaves` slice before hit-testing.

---

### [LOW] Pan offset accumulated in `float64`, camera.Scale also `float64` — consistent, but `InputState` drag fields are mixed

- **File**: `pkg/pulsemap/interaction/input.go` (lines 280–310 `InputState`)
- **Category**: Accuracy
- **Problem**: `InputState.LastDx`, `LastDy`, `DragStartX`, `DragStartY`, `LastX`, `LastY` are all `float64`. `Camera.X/Y/Scale` are `float64`. Ebitengine cursor coordinates arrive as `int` and are cast to `float64` in `handleDragging`. This chain is consistent. However, `velocityX/Y` in `Camera` is `float64` (world units/tick) while the pan delta passed to `camera.Pan(dx, dy)` is also `float64`. No overflow risk exists. **This item is LOW — no bug, but noted for completeness.**
- **Evidence**: All fields reviewed align on `float64`.
- **Fix**: No action required; mark as not a defect.
- **Remediation checklist**:
  - [x] Confirmed: no type mismatch across pan/zoom pipeline.

---

## 3. Panel & State Transitions

---

### [HIGH] Wave submission failure silently discarded — no user notification

- **File**: `pkg/pulsemap/game.go` (lines 583–631 `handleWaveSubmit()`)
- **Category**: Transition / Convenience
- **Problem**: `handleWaveSubmit` runs PoW and publishes asynchronously in a goroutine. On failure (PoW error, publish timeout, keypair nil), the function calls `log.Printf` and returns. There is no in-app notification; the compose panel closes after submit is called (`p.visible = false` in `submit()`) so the user sees the panel disappear and no feedback. PoW runs 2–5 seconds; the user cannot tell whether submission succeeded or failed.
- **Evidence**: Lines 594, 608, 617, 622 all contain `log.Printf(...); return` with no UI update path.
- **Reproducible trigger**: Submit a Wave with pubsub not yet connected → panel closes, no error appears.
- **Fix**: Add a notification / toast mechanism. Use the existing `PanelAnimation.SetError` pattern or a dedicated overlay. The goroutine should signal back to the main thread via a channel and the next `Update()` tick can display a brief toast over the Pulse Map:
  ```go
  select {
  case err := <-g.waveResultCh:
      if err != nil {
          g.showToast("Wave failed: " + err.Error(), toastTypeError)
      } else {
          g.showToast("Wave sent", toastTypeSuccess)
      }
  default:
  }
  ```
- **Remediation checklist**:
  - [x] Add `waveResultCh chan error` to `Game` and send to it from the goroutine.
  - [x] Add a `ToastNotification` struct with message, type (success/error), and decay timer.
  - [x] Draw toast in `Draw()` above all other layers.
  - [x] Show "Sending…" spinner in compose panel until result arrives.

---

### [HIGH] Compose panel and node detail panel can be simultaneously visible

- **File**: `pkg/pulsemap/game.go` (lines 740–750 `handleNodeDetailComposeWave()`)
- **Category**: Transition
- **Problem**: `handleNodeDetailComposeWave` calls `g.composePanel.Show()` without first calling `g.nodeDetailPanel.Hide()`. Both panels are then visible simultaneously and both `Update()` methods run in order via `updateActivePanels()`. The compose panel occupies the bottom-right corner while node detail occupies the right edge; they share screen space. Keyboard input (Enter, Escape) goes to whichever panel is listed first in the priority chain.
- **Evidence**: `game.go` line 746: `g.composePanel.Show()` with no preceding `g.nodeDetailPanel.Hide()`.
- **Reproducible trigger**: Click a node → node detail slides in; click "Compose Wave" in detail panel → both panels visible simultaneously.
- **Fix**: Before showing compose, hide incompatible panels:
  ```go
  func (g *Game) handleNodeDetailComposeWave(nodeID string) {
      g.nodeDetailPanel.Hide()
      g.composePanel.SetTargetNode(nodeID)
      g.composePanel.Show()
  }
  ```
- **Remediation checklist**:
  - [x] Define an exclusive panel group or a `showOnlyPanel(p)` helper in `Game`.
  - [x] Apply the same exclusivity to `handleSearchBarToggle` (closes node detail if open).
  - [x] Audit all `*.Show()` call sites for missing paired `Hide()` calls on peers.

---

### [HIGH][FIXED] Settings panel is never instantiated or accessible to the user

- **File**: `pkg/pulsemap/game.go` (no reference to `SettingsPanel`); `pkg/ui/settings.go`
- **Category**: Transition / Convenience
- **Problem**: `ui.SettingsPanel` is fully implemented with 5 setting categories (Network, Privacy, Devices, Display, Waves). However, `game.go` never creates a `SettingsPanel`, adds it to `updateActivePanels()`, draws it, or binds a keyboard shortcut/button to open it. All settings are inaccessible at runtime.
- **Evidence**: `grep "SettingsPanel\|settingsPanel"` in all non-stub, non-test files → only `settings.go` itself.
- **Reproducible trigger**: There is no way to open Settings from the Pulse Map.
- **Fix**:
  1. Add `settingsPanel *ui.SettingsPanel` to `Game`.
  2. Initialise in `NewGame`.
  3. Bind `Ctrl+,` (macOS convention) or `Ctrl+P` to `g.settingsPanel.Toggle()`.
  4. Include in `updateActivePanels()` and `Draw()`.
- **Remediation checklist**:
  - [x] Add `settingsPanel` field and `NewSettingsPanel(theme, g.handleSettingChange)` in `NewGame`.
  - [x] Add `handleSettingChange(key, value string)` method that applies setting to live subsystems.
  - [x] Add keyboard shortcut toggle.
  - [x] Draw settings panel in `Draw()` (modal — above all other panels).
  - [x] Wire `Privacy > Privacy Mode` to `pkg/identity/modes` package.

---

### [MEDIUM] Bootstrap screen advances past peer discovery without real peer count

- **File**: `pkg/onboarding/screens/bootstrap_screen.go` (lines 456–459 `handleConnectingClick()`, lines 576–596 `SimulateDiscoveryComplete()`)
- **Category**: Transition
- **Problem**: `discoveryDone` is set to `true` in two ways: (1) `SimulatePeerFound()` increments a counter until `peersFound >= targetPeers`, and (2) `SimulateDiscoveryComplete()` sets it unconditionally. Neither method is connected to the real `pkg/onboarding/bootstrap` network package — real peer-connection events (`OnPeerConnected` callback) are never bridged into `BootstrapScreen.SimulatePeerFound()`. The screen will be stuck permanently at "Searching for peers…" on a real network because `discoveryDone` is never set to true by an actual peer connection.
- **Evidence**: `bootstrap_screen.go` has no import of `pkg/onboarding/bootstrap`. The `callbacks.OnPeerFound` callback in `BootstrapScreenCallbacks` is never wired to the real `bootstrap.NetworkManager.OnPeerConnected`.
- **Reproducible trigger**: Run the app end-to-end: bootstrap screen shows "Searching for peers…" indefinitely even after libp2p connects to peers.
- **Fix**: In the application wiring layer (or in `BootstrapScreen` via an injected `NetworkManager`), forward `OnPeerConnected` events to `screen.SimulatePeerFound()`:
  ```go
  networkMgr.SetCallbacks(bootstrap.Callbacks{
      OnPeerConnected: func(id string) { bootstrapScreen.SimulatePeerFound() },
  })
  ```
- **Remediation checklist**:
  - [x] Inject `bootstrap.NetworkManager` (or an interface) into `BootstrapScreen` or its caller.
  - [x] Forward `OnPeerConnected` → `screen.NotifyPeerFound()`.
  - [x] Rename `SimulatePeerFound` → `NotifyPeerFound` to reduce confusion with test helpers.
  - [ ] Add integration test: connect two in-process libp2p nodes, assert screen shows `discoveryDone=true`.

---

### [MEDIUM][FIXED] Anonymous overlay layer is always empty — toggle has no visual effect

- **File**: `pkg/pulsemap/rendering/renderer.go` (lines 345–350 comment block in `Draw()`); `pkg/pulsemap/overlays/layer.go`
- **Category**: Transition
- **Problem**: `Draw()` creates and composites `r.overlayLayer` but the only code that writes to it is in a comment: `// Currently empty - future implementation for cross-layer artifacts`. `LayerBlend` and `ParticleEmitter` from `pkg/pulsemap/overlays/layer.go` are implemented but are never instantiated in `Renderer`. Toggling the Anonymous Layer overlay (e.g., via a blend ratio slider) has no visible effect because nothing is drawn to the overlay layer.
- **Evidence**: `renderer.go` overlay section comment; `overlays/layer.go` defines `LayerBlend`, `SpecterParticle`, `ParticleEmitter` — none of these are imported or used in `renderer.go`.
- **Fix**: Instantiate a `LayerBlend` and `ParticleEmitter` in `Renderer` and populate `overlayLayer` during `Draw()` by calling emitter `Render()` for each Specter node:
  ```go
  if r.layerBlend != nil && r.layerBlend.AnonymousOpacity > 0 {
      for id, pos := range positions {
          if emitter, ok := r.specterEmitters[id]; ok {
              emitter.Render(r.overlayLayer, float32(r.camera.X), float32(r.camera.Y), float32(r.camera.Scale))
          }
      }
  }
  ```
- **Remediation checklist**:
  - [x] Add `layerBlend *overlays.LayerBlend` and `specterEmitters map[string]*overlays.ParticleEmitter` to `Renderer`.
  - [x] Initialise in `NewRenderer`.
  - [x] Populate overlay layer in `Draw()`.
  - [x] Expose `SetLayerBlend(ratio float32)` on `Renderer` for the UI to call.

---

### [LOW][FIXED] Shadow Gradient mode change does not reset open panels

- **File**: `pkg/pulsemap/game.go` (no `OnPrivacyModeChange` handler); `pkg/ui/settings.go` (`Privacy Mode` setting)
- **Category**: Transition
- **Problem**: Switching between Open/Hybrid/Guarded/Fortress modes should hide UI elements that are only valid in certain modes (e.g., Specter-specific panels should close in Open mode; the Anonymous Layer overlay should hide in Fortress mode for surface panels). Currently the settings `onChange` callback (`handleSettingChange` — not yet implemented, see HIGH finding above) has no handler, so mode changes have no UI effect.
- **Fix**: When privacy mode changes, dispatch a `ModeChangedEvent` on the event bus that the UI subscribes to, closing inappropriate panels.
- **Remediation checklist**:
  - [x] Implement `handleSettingChange` in `Game` (blocked by: Settings panel not wired).
  - [x] On `privacy_mode` change, call `g.closeSpecterPanels()` when switching to Open mode.
  - [x] Drive `LayerBlend` opacity from the new mode value.

---

## 4. Convenience

---

### [HIGH] Wave submit failure (PoW timeout, publish error) silently discarded

*(Duplicate reference — see finding in Section 3: "Wave submission failure silently discarded")*

---

### [MEDIUM][FIXED] `performSearch` drops the mutex, runs callback, re-acquires — results can be stale or lost

- **File**: `pkg/ui/search.go` (lines 225–238 `performSearch()`)
- **Category**: Convenience
- **Problem**: `performSearch` unlocks `s.mu`, calls `s.callbacks.OnSearch(query)` (which in `game.go` iterates `renderer.GetAllNodes()` — itself acquiring `renderer.mu`), then re-locks and writes to `s.results`. Between unlock and re-lock, another goroutine can call `s.Hide()`, setting `s.visible = false` and clearing `s.results`. When `performSearch` re-locks and writes, it writes stale results into a now-hidden search bar, and the results are invisible but will display unexpectedly the next time the bar is shown.
- **Evidence**: `search.go` lines 232–237: `s.mu.Unlock()` → `results := s.callbacks.OnSearch(query)` → `s.mu.Lock()` → `s.results = results`.
- **Fix**: After re-acquiring the lock, check `s.visible && s.query == query` before storing results:
  ```go
  s.mu.Lock()
  if s.visible && s.query == query {
      s.results = results
      s.selectedIndex = -1
  }
  s.mu.Unlock()
  ```
  Also add debounce: only call `performSearch` when the query has not changed for ≥ 3 ticks (≈50ms at 60fps) to avoid queuing a search per keystroke.
- **Remediation checklist**:
  - [x] Add stale-result guard after re-lock.
  - [x] Add `lastSearchTick int64` field and skip search if `tickCount - lastSearchTick < 3`.
  - [x] Add test: call `Hide()` between search dispatch and result delivery; assert `results` is not stored.

---

### [MEDIUM] Compose panel cursor rendering uses fixed 7-pixels-per-character advance — broken for multibyte characters

- **File**: `pkg/ui/compose.go` (line 350 `drawTextArea()`)
- **Category**: Convenience
- **Problem**: The blinking cursor is positioned at `textX + 8 + p.cursorPos*7`. The magic `7` is the monospace advance width for `basicfont.Face7x13`. This is only correct for ASCII characters. For multibyte Unicode characters (CJK, emoji, accented Latin), the advance is wider, so the cursor appears inside or before the rune it should follow.
- **Evidence**: Line 350: `cursorX := textX + 8 + p.cursorPos*7`.
- **Fix**: Use `text.Advance(p.content[:cursorByteOffset], face)` or measure the string with `basicfont.Face7x13.GlyphAdvance` to compute the correct pixel offset. Alternatively render cursor at a visually-sensible position by calling `font.BoundString`.
- **Remediation checklist**:
  - [ ] Replace `p.cursorPos*7` with `measureTextWidth(p.content, p.cursorPos, face)`.
  - [ ] Implement `measureTextWidth` using `ebiten/v2/text/v2` advance measurement.
  - [ ] Add test with multibyte content ("日本語") asserting cursor X > cursorPos*7.

---

### [MEDIUM] Compose panel buttons (`DrawCancelSubmitButtons`) are not interactive — Cancel has no click handler either

*(Covered by CRITICAL finding in Section 1. Both Submit and Cancel lack mouse handlers.)*

---

### [LOW][FIXED] Viewport control buttons are 70×30 px — below 44 px minimum touch target

- **File**: `pkg/ui/viewport_controls.go` (line 60: `buttonHeight: 30`)
- **Category**: Convenience
- **Problem**: `ViewportControls` buttons are 70×30 logical pixels. The 44×44 px minimum touch target (Apple HIG, WCAG 2.5.5) is not met on the height axis. On touch-screen or high-DPI displays, hitting the Macro/Meso/Micro buttons reliably requires high precision.
- **Evidence**: Line 60: `buttonHeight: 30`.
- **Fix**: Increase `buttonHeight` to 44. Adjust `startY` margin accordingly to avoid overlapping the top edge:
  ```go
  buttonHeight: 44,
  ```
- **Remediation checklist**:
  - [x] Set `buttonHeight = 44` in `NewViewportControls`.
  - [x] Verify three stacked buttons + 2 gaps still fit within a 200px top margin.

---

### [LOW] Radial menu item radius (35 px) meets size requirement but label placement crosses center dead zone at 6+ items

- **File**: `pkg/ui/radial_menu.go` (lines 60–65 constants)
- **Category**: Convenience
- **Problem**: With 6 items the angle step is 60°. At `radialMenuRadius=100` and `radialMenuItemRadius=35`, adjacent item circles overlap at 60° spacing (arc between centers = 2π×100/6 ≈ 105 px, diameter = 70 px — no overlap). However, `radialMenuLabelOffset=15` places labels 50 px from center, which overlaps the inner cancel zone (`radialMenuInnerRadius=20`) at some angles. Corner sectors also require the cursor to move from center to radius 100, which at diagonal angles passes over the edge of adjacent sectors, potentially activating the wrong sector.
- **Evidence**: Constants lines 60–65.
- **Fix**: Increase `radialMenuRadius` to 110 and `radialMenuInnerRadius` to 25 to provide more clearance, and use angle mid-point bisector for hit detection instead of circular proximity.
- **Remediation checklist**:
  - [ ] Increase `radialMenuRadius` from 100 to 110.
  - [ ] Replace `isItemHovered` circular proximity with angular sector test: compute angle from center to cursor, compare to item's `itemAngle ± angleStep/2`.

---

### [LOW] Search bar `handleEscapeKey` always returns `true` even when not visible

- **File**: `pkg/ui/search.go` (lines 152–162 `handleEscapeKey()`)
- **Category**: Convenience
- **Problem**: `handleEscapeKey()` is called from `Update()` only when `s.visible` is true (guard at line 114: `if !s.visible { return false }`). However, looking at the function itself, it returns `true` unconditionally — meaning even in the guard-bypassed path it would consume Escape. Additionally, the final line of `Update()` is `return s.handleEscapeKey()`, which means the search bar always returns `true` from `Update()` even after all other input, consuming Escape from the overall panel chain regardless.
- **Evidence**: `search.go` line 161: `return true` at end of `handleEscapeKey`, and the call site `return s.handleEscapeKey()` is the final return of `Update()`.
- **Fix**: Change the terminal `return true` inside `handleEscapeKey` to `return false` when Escape was not pressed:
  ```go
  func (s *SearchBar) handleEscapeKey() bool {
      if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
          // ... close and return true
          return true
      }
      return false // Do not consume input if Escape was not pressed
  }
  ```
- **Remediation checklist**:
  - [ ] Fix `return true` → `return false` in the non-Escape path of `handleEscapeKey`.
  - [ ] Add test: call `Update()` with Escape not pressed, assert return is `false`.

---

## Files Evaluated

| File | Status |
|---|---|
| `pkg/pulsemap/interaction/input.go` | ✅ Fully read |
| `pkg/pulsemap/interaction/touch.go` | ✅ Fully read |
| `pkg/pulsemap/game.go` | ✅ Fully read |
| `pkg/pulsemap/layout/engine.go` | ✅ Fully read |
| `pkg/pulsemap/layout/clustering.go` | ✅ Partially read (constants + clustering logic) |
| `pkg/pulsemap/layout/throttle.go` | ⏭ Skipped — scheduling logic, no UI impact |
| `pkg/pulsemap/layout/viewport_culling.go` | ⏭ Skipped — world-space culling math, no UI |
| `pkg/pulsemap/rendering/renderer.go` | ✅ Fully read |
| `pkg/pulsemap/rendering/draw.go` | ✅ Fully read |
| `pkg/pulsemap/rendering/background.go` | ⏭ Skipped — procedural gradient, no input/coord |
| `pkg/pulsemap/rendering/particles.go` | ⏭ Skipped — ambient particles, no input |
| `pkg/pulsemap/rendering/batch.go` | ⏭ Skipped — draw-call batching internals |
| `pkg/pulsemap/rendering/colors.go` | ⏭ Skipped — color derivation helpers |
| `pkg/pulsemap/rendering/artifacts.go` | ⏭ Skipped — cross-layer artifact queries |
| `pkg/pulsemap/rendering/sigil_image.go` | ⏭ Skipped — image generation |
| `pkg/pulsemap/overlays/layer.go` | ✅ Fully read |
| `pkg/pulsemap/overlays/heatmap.go` | ⏭ Skipped — heatmap rendering internals |
| `pkg/pulsemap/overlays/marks.go` | ⏭ Skipped — mark rendering |
| `pkg/pulsemap/overlays/gift.go` | ⏭ Skipped — gift rendering |
| `pkg/pulsemap/overlays/territory.go` | ⏭ Skipped — territory rendering |
| `pkg/pulsemap/overlays/hunts.go` | ⏭ Skipped — hunt rendering |
| `pkg/pulsemap/overlays/puzzles.go` | ⏭ Skipped — puzzle rendering |
| `pkg/pulsemap/overlays/councils.go` | ⏭ Skipped — council rendering |
| `pkg/pulsemap/overlays/forge.go` | ⏭ Skipped — forge rendering |
| `pkg/pulsemap/overlays/oracle_pool.go` | ⏭ Skipped — oracle rendering |
| `pkg/pulsemap/overlays/shadowplay.go` | ⏭ Skipped — shadow play rendering |
| `pkg/pulsemap/overlays/echochains.go` | ⏭ Skipped — echo chain rendering |
| `pkg/pulsemap/overlays/echoindex.go` | ⏭ Skipped — echo index |
| `pkg/pulsemap/overlays/camera_helpers.go` | ⏭ Skipped — camera helpers |
| `pkg/ui/compose.go` | ✅ Fully read |
| `pkg/ui/passphrase_prompt.go` | ✅ Fully read |
| `pkg/ui/node_detail.go` | ✅ Fully read |
| `pkg/ui/radial_menu.go` | ✅ Fully read |
| `pkg/ui/search.go` | ✅ Fully read |
| `pkg/ui/settings.go` | ✅ Fully read |
| `pkg/ui/panel.go` | ✅ Fully read |
| `pkg/ui/panel_helpers.go` | ✅ Fully read |
| `pkg/ui/viewport_controls.go` | ✅ Fully read |
| `pkg/ui/specter_detail.go` | ⏭ Skipped — widget structure mirrors node_detail, no novel patterns |
| `pkg/ui/mark.go` | ⏭ Skipped — mark placement form |
| `pkg/ui/gift.go` | ⏭ Skipped — gift form |
| `pkg/ui/councils.go` | ⏭ Skipped — council UI |
| `pkg/ui/forge.go` | ⏭ Skipped — forge UI |
| `pkg/ui/puzzle_solver.go` | ⏭ Skipped — puzzle UI |
| `pkg/ui/masked_event.go` | ⏭ Skipped — event UI |
| `pkg/ui/oracle_pool.go` | ⏭ Skipped — oracle pool UI |
| `pkg/ui/shadowplay.go` | ⏭ Skipped — shadow play UI |
| `pkg/ui/territory_overview.go` | ⏭ Skipped — territory overview UI |
| `pkg/ui/hunt_tracker.go` | ⏭ Skipped — hunt tracker UI |
| `pkg/onboarding/flow/controller.go` | ✅ Fully read |
| `pkg/onboarding/screens/bootstrap_screen.go` | ✅ Fully read |
| `pkg/onboarding/screens/identity.go` | ⏭ Skipped — identity creation form |
| `pkg/onboarding/screens/mode_screen.go` | ⏭ Skipped — mode selection, reviewed phase-advance call |
| `pkg/onboarding/screens/completion_screen.go` | ⏭ Skipped — completion display |

---

## Severity Summary

| Severity | Count | Status |
|---|---|---|
| CRITICAL | 1 | 🔴 Unresolved |
| HIGH | 7 | 🔴 Unresolved |
| MEDIUM | 6 | 🟡 Unresolved |
| LOW | 4 | 🟢 Low priority |

---

*End of audit. Update this file as findings are resolved by checking off checklist items and changing section headers from `[SEVERITY]` to `[SEVERITY][FIXED]`.*
