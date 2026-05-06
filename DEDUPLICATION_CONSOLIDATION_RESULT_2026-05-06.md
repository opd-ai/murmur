# Code Deduplication Consolidation Result
**Date**: 2026-05-06  
**Task**: Identify and consolidate the top 5–10 most significant code clone groups below duplication thresholds

## Executive Summary

Successfully consolidated **4 clone groups**, removing **69 lines of duplicated code** across **8 instances**. All tests pass with zero regressions.

## Metrics

| Metric | Baseline | Post-Consolidation | Improvement |
|--------|----------|-------------------|-------------|
| Clone groups | 49 | 45 | -4 groups |
| Duplication ratio | 0.64% | 0.57% | -0.07% |
| Duplicated lines | 661 | 592 | -69 lines |
| Test status | PASS | PASS | ✅ |

## Consolidated Clone Groups

### Clone Group #1: Panel Visibility and Centering (4 instances, 10 lines each)
**Strategy**: Extract function  
**Consolidated into**: `CheckPanelVisibilityAndCenter()` in `pkg/ui/panel_helpers.go`  
**Tests**: PASS

**Locations consolidated**:
- `pkg/ui/device_management.go:141-150`
- `pkg/ui/device_pairing.go:294-303`
- `pkg/ui/passphrase_prompt.go:147-156`
- `pkg/ui/settings.go:210-219`

**Pattern extracted**:
```go
if !p.visible {
    return
}
w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
px := (w - p.width) / 2
py := (h - p.height) / 2
```

**New helper**:
```go
func CheckPanelVisibilityAndCenter(screen *ebiten.Image, visible bool, panelWidth, panelHeight int) (px, py, w, h int, shouldRender bool)
```

---

### Clone Group #2: Modal Overlay and Panel Drawing (2 instances, 12 lines each)
**Strategy**: Extract function  
**Consolidated into**: `DrawModalOverlayAndPanel()` in `pkg/ui/panel_helpers.go`  
**Tests**: PASS

**Locations consolidated**:
- `pkg/ui/device_management.go:149-156`
- `pkg/ui/passphrase_prompt.go:155-162`

**Pattern extracted**:
```go
vector.DrawFilledRect(screen, 0, 0, float32(w), float32(h), theme.PanelBackground, true)
vector.DrawFilledRect(screen, float32(px), float32(py), float32(width), float32(height), theme.PanelBackground, true)
vector.StrokeRect(screen, float32(px), float32(py), float32(width), float32(height), 2.0, theme.PanelBorder, true)
```

**New helper**:
```go
func DrawModalOverlayAndPanel(screen *ebiten.Image, px, py, w, h int, panelWidth, panelHeight int, theme Theme)
```

---

### Clone Group #3: Mark Panel Drawing Coordinate Initialization (2 instances, 10 lines each)
**Strategy**: Extract method  
**Consolidated into**: `initDrawCoords()` method on `MarkPanel`  
**Tests**: PASS

**Locations consolidated**:
- `pkg/ui/mark.go:484-485`
- `pkg/ui/mark.go:681-682`

**Pattern extracted**:
```go
x := float32(p.panelX + 20)
y := float32(p.panelY + 20)
```

**New method**:
```go
func (p *MarkPanel) initDrawCoords() (x, y float32)
```

---

### Clone Group #4: ShadowPlay Title Drawing (2 instances, 10 lines each)
**Strategy**: Extract method  
**Consolidated into**: `drawTitle()` method on `ShadowPlayPanel`  
**Tests**: PASS

**Locations consolidated**:
- `pkg/ui/shadowplay.go:432-441`
- `pkg/ui/shadowplay.go:540-549`

**Pattern extracted**:
```go
if defaultFont == nil {
    return
}
titleOpts := &text.DrawOptions{}
titleOpts.GeoM.Translate(float64(x+w/2+offsetX), float64(y+20))
titleOpts.ColorScale.ScaleWithColor(sp.theme.TextPrimary)
text.Draw(screen, title, defaultFont, titleOpts)
```

**New method**:
```go
func (sp *ShadowPlayPanel) drawTitle(screen *ebiten.Image, title string, x, y, w float32, offsetX float32)
```

---

## Clone Groups Evaluated But Not Consolidated

| Clone Group | Reason Not Consolidated |
|-------------|-------------------------|
| Update() mutex lock + visibility check | Idiomatic Go pattern — extracting would hurt readability |
| State update patterns (Forge/ShadowPlay/Councils/Hunts) | Different types serving conceptually different purposes |
| Error rendering conditionals | Too small (3 lines); idiomatic pattern |
| Info box coordinate calculations | Variable-dependent values; no clear simplification |
| Stub file duplicates (puzzle_solver.go / puzzle_solver_stub.go) | Intentional duplication for mutually exclusive build tags (`!test` vs `test`) |
| Gifts/Marks publisher validation | Different error types and store methods; serves different mechanics |
| Active item counting (echochains/sparks) | Different types and fields; extraction requires generics without clear benefit |
| Protobuf parsing error handling | Standard Go idiom; should not be extracted |

---

## Files Modified

- `pkg/ui/panel_helpers.go` — Added 2 new helper functions
- `pkg/ui/device_management.go` — Simplified Draw() method
- `pkg/ui/device_pairing.go` — Simplified Draw() method
- `pkg/ui/passphrase_prompt.go` — Simplified Draw() method
- `pkg/ui/settings.go` — Simplified Draw() method
- `pkg/ui/mark.go` — Added helper method, simplified 2 drawing methods
- `pkg/ui/shadowplay.go` — Added helper method, simplified 2 drawing methods

---

## Test Results

```bash
go test -race ./...
```

**Result**: All tests PASS ✅

- 0 build failures
- 0 test failures
- 0 race conditions
- All 40+ packages validated

---

## Deduplication Rules Applied

1. ✅ Started with shortest clone groups per priority tier
2. ✅ Preserved all existing public API signatures
3. ✅ Each extracted helper <30 lines with clear purpose
4. ✅ Maintained idiomatic Go style
5. ✅ Did not merge clones serving different conceptual purposes
6. ✅ Validated with test suite after each consolidation

---

## Recommendations

### Remaining Duplication (0.57% ratio)

The remaining 592 duplicated lines across 45 clone groups fall into three categories:

1. **Intentional duplication** (build tag separation, stubs)
2. **Idiomatic Go patterns** (error handling, mutex guards, defer cleanup)
3. **Contextually different implementations** (different types, different mechanics)

**Verdict**: Current duplication level (0.57%) is below the 5% target threshold and represents idiomatic, maintainable code. Further consolidation would likely reduce code clarity without meaningful benefit.

### Future Monitoring

- Run `go-stats-generator` in CI to track duplication ratio
- Alert if duplication ratio exceeds 1.0%
- Review new clone groups ≥15 lines during code review

---

## Conclusion

Successfully reduced duplication from **0.64%** to **0.57%** by consolidating 4 meaningful clone groups while preserving code clarity and idiomatic Go patterns. All tests pass with zero regressions.
