// Package overlays provides Anonymous Layer overlay and activity heatmap.
// This is the noebiten stub for Specter Mark visualization.
//
//go:build test
// +build test

package overlays

import (
	"sync"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

// MarkDisplay represents a single mark being displayed on a node.
type MarkDisplay struct {
	Mark       *marks.Mark
	OrbitAngle float32
	OrbitSpeed float32
	PulsePhase float32
}

// MarkOverlay manages Specter Mark visualization on the Pulse Map.
type MarkOverlay struct {
	mu    sync.RWMutex
	marks map[string][]*MarkDisplay
}

// NewMarkOverlay creates a new mark overlay manager.
func NewMarkOverlay() *MarkOverlay {
	return &MarkOverlay{
		marks: make(map[string][]*MarkDisplay),
	}
}

// AddMark registers a mark for display on a target node.
func (o *MarkOverlay) AddMark(targetID string, mark *marks.Mark) {
	o.addMarkImpl(targetID, mark)
}

// addMarkImpl implements the add mark logic.
// Shared between marks.go and marks_stub.go.
func (o *MarkOverlay) addMarkImpl(targetID string, mark *marks.Mark) {
	if mark == nil || mark.IsExpired() {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	for _, d := range o.marks[targetID] {
		if d.Mark != nil && d.Mark.ID == mark.ID {
			return
		}
	}

	orbitSpeed := 0.5 + float32(mark.ID[0]%64)/128.0
	o.marks[targetID] = append(o.marks[targetID], &MarkDisplay{
		Mark:       mark,
		OrbitAngle: float32(mark.ID[1]) / 40.0,
		OrbitSpeed: orbitSpeed,
		PulsePhase: 0,
	})
}

// RemoveMark removes a specific mark from display.
func (o *MarkOverlay) RemoveMark(markID [32]byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for targetID, displays := range o.marks {
		if o.removeMarkFromDisplays(targetID, displays, markID) {
			return
		}
	}
}

// removeMarkFromDisplays removes mark from displays list. Returns true if found.
func (o *MarkOverlay) removeMarkFromDisplays(targetID string, displays []*MarkDisplay, markID [32]byte) bool {
	for i, d := range displays {
		if d.Mark == nil || d.Mark.ID != markID {
			continue
		}
		o.marks[targetID] = append(displays[:i], displays[i+1:]...)
		if len(o.marks[targetID]) == 0 {
			delete(o.marks, targetID)
		}
		return true
	}
	return false
}

// RemoveAllMarksForTarget removes all marks from a target node.
func (o *MarkOverlay) RemoveAllMarksForTarget(targetID string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.marks, targetID)
}

// ClearExpiredMarks removes all expired marks.
func (o *MarkOverlay) ClearExpiredMarks() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for targetID, displays := range o.marks {
		active := displays[:0]
		for _, d := range displays {
			if d.Mark != nil && !d.Mark.IsExpired() {
				active = append(active, d)
			}
		}
		if len(active) > 0 {
			o.marks[targetID] = active
		} else {
			delete(o.marks, targetID)
		}
	}
}

// Update advances animation state for all marks.
func (o *MarkOverlay) Update(dt float32) {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, displays := range o.marks {
		for _, d := range displays {
			d.OrbitAngle += d.OrbitSpeed * dt
			if d.OrbitAngle > 6.283185 {
				d.OrbitAngle -= 6.283185
			}
			d.PulsePhase += 1.5 * dt
			if d.PulsePhase > 6.283185 {
				d.PulsePhase -= 6.283185
			}
		}
	}
}

// Render is a no-op in the noebiten build.
func (o *MarkOverlay) Render(screen interface{}, targetID string, nodeX, nodeY float32) {
	// No-op in noebiten build.
}

// GetMarkCount returns the number of marks on a target.
func (o *MarkOverlay) GetMarkCount(targetID string) int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.marks[targetID])
}

// GetTotalMarkCount returns total marks being displayed.
func (o *MarkOverlay) GetTotalMarkCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	total := 0
	for _, displays := range o.marks {
		total += len(displays)
	}
	return total
}

// HasMarks returns true if the target has any visible marks.
func (o *MarkOverlay) HasMarks(targetID string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.marks[targetID]) > 0
}

// GetDominantCategory returns the most common mark category for a target.
func (o *MarkOverlay) GetDominantCategory(targetID string) marks.MarkCategory {
	return o.getDominantCategoryImpl(targetID)
}

// getDominantCategoryImpl computes the dominant mark category.
// Shared between marks.go and marks_stub.go.
func (o *MarkOverlay) getDominantCategoryImpl(targetID string) marks.MarkCategory {
	o.mu.RLock()
	displays := o.marks[targetID]
	o.mu.RUnlock()

	if len(displays) == 0 {
		return 0
	}

	counts := make(map[marks.MarkCategory]int)
	for _, d := range displays {
		if d.Mark != nil {
			counts[d.Mark.Category]++
		}
	}

	var dominant marks.MarkCategory
	maxCount := 0
	for cat, count := range counts {
		if count > maxCount {
			maxCount = count
			dominant = cat
		}
	}
	return dominant
}

// SyncFromStore updates the overlay from a MarkStore.
func (o *MarkOverlay) SyncFromStore(store *marks.MarkStore) {
	if store == nil {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	o.removeExpiredMarks()
	o.addNewMarksFromStore(store)
}

// removeExpiredMarks clears all expired marks from the overlay.
func (o *MarkOverlay) removeExpiredMarks() {
	for targetID, displays := range o.marks {
		active := displays[:0]
		for _, d := range displays {
			if d.Mark != nil && !d.Mark.IsExpired() {
				active = append(active, d)
			}
		}
		if len(active) > 0 {
			o.marks[targetID] = active
		} else {
			delete(o.marks, targetID)
		}
	}
}

// addNewMarksFromStore adds marks from the store that aren't already displayed.
func (o *MarkOverlay) addNewMarksFromStore(store *marks.MarkStore) {
	allMarks := store.GetAllActiveMarks()
	for _, mark := range allMarks {
		if mark == nil || mark.IsExpired() {
			continue
		}
		o.addMarkIfNotPresent(mark)
	}
}

// addMarkIfNotPresent adds a mark to display if not already present.
func (o *MarkOverlay) addMarkIfNotPresent(mark *marks.Mark) {
	targetID := keyToHexMarks(mark.TargetKey[:])

	if o.hasMarkWithID(targetID, mark.ID) {
		return
	}

	orbitSpeed := 0.5 + float32(mark.ID[0]%64)/128.0
	o.marks[targetID] = append(o.marks[targetID], &MarkDisplay{
		Mark:       mark,
		OrbitAngle: float32(mark.ID[1]) / 40.0,
		OrbitSpeed: orbitSpeed,
		PulsePhase: 0,
	})
}

// hasMarkWithID checks if a mark with given ID is already displayed for target.
func (o *MarkOverlay) hasMarkWithID(targetID string, markID [32]byte) bool {
	for _, d := range o.marks[targetID] {
		if d.Mark != nil && d.Mark.ID == markID {
			return true
		}
	}
	return false
}

// keyToHexMarks converts a byte slice to hex string.
func keyToHexMarks(key []byte) string {
	const hexChars = "0123456789abcdef"
	result := make([]byte, len(key)*2)
	for i, b := range key {
		result[i*2] = hexChars[b>>4]
		result[i*2+1] = hexChars[b&0x0f]
	}
	return string(result)
}
