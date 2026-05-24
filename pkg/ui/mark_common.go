// Package ui - Shared Specter Mark logic (no build tags).
// This file contains logic common to both mark.go (!test) and mark_stub.go (test).

package ui

import (
	"fmt"

	"github.com/opd-ai/murmur/pkg/anonymous/mechanics/marks"
)

// initShowState initializes the panel state for Show().
// This logic is shared between mark.go and mark_stub.go implementations.
func (p *MarkPanel) initShowState() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check resonance requirement.
	resonance := 0
	if p.callbacks.GetMyResonance != nil {
		resonance = p.callbacks.GetMyResonance()
	}

	if resonance < marks.MarkMinResonance {
		p.visible = true
		p.mode = MarkModeError
		p.errorMsg = fmt.Sprintf("Resonance %d required (you have %d)", marks.MarkMinResonance, resonance)
		return
	}

	// Check active mark limit.
	activeMarks := 0
	if p.callbacks.GetActiveMarkCount != nil {
		activeMarks = p.callbacks.GetActiveMarkCount()
	}

	if activeMarks >= 5 {
		p.visible = true
		p.mode = MarkModeError
		p.errorMsg = "Maximum 5 active marks reached"
		return
	}

	// Initialize panel state.
	p.visible = true
	p.mode = MarkModeCategorySelect
	p.selectedCategory = 0
	p.selectedTarget = 0
	p.targetScroll = 0
	p.note = ""
	p.noteFocused = false
	p.errorMsg = ""
	p.successMsg = ""

	// Load targets.
	if p.callbacks.GetTargets != nil {
		p.targets = p.callbacks.GetTargets()
	}
}
