// Package app provides glue code to integrate the onboarding flow.
// This file bridges pkg/app and pkg/onboarding/flow without circular dependencies.
package app

import (
	"time"

	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// flowControllerAdapter adapts flow.Controller to onboardingFlowController interface.
type flowControllerAdapter struct {
	controller *flow.Controller
}

// Start begins the onboarding flow.
func (f *flowControllerAdapter) Start() {
	f.controller.Start()
}

// CurrentPhase returns the current onboarding phase.
func (f *flowControllerAdapter) CurrentPhase() onboardingPhase {
	return phaseAdapter{f.controller.CurrentPhase()}
}

// CompleteCurrentPhase marks the current phase as complete and advances.
func (f *flowControllerAdapter) CompleteCurrentPhase() {
	f.controller.CompleteCurrentPhase()
}

// IsComplete returns true if the onboarding flow is complete.
func (f *flowControllerAdapter) IsComplete() bool {
	return f.controller.IsComplete()
}

// phaseAdapter adapts flow.Phase to onboardingPhase interface.
type phaseAdapter struct {
	phase flow.Phase
}

// String returns the string representation of the onboarding phase.
func (p phaseAdapter) String() string {
	return p.phase.String()
}

// newFlowControllerImpl creates a flow.Controller with adapted callbacks.
// This is called from murmur.go's newFlowController function.
func newFlowControllerImpl(callbacks flowCallbacks) onboardingFlowController {
	// Adapt the int-based callbacks to flow.Phase-based callbacks.
	flowCallbacks := flow.Callbacks{
		OnPhaseStart: func(phase flow.Phase) {
			if callbacks.onPhaseStart != nil {
				callbacks.onPhaseStart(int(phase))
			}
		},
		OnPhaseComplete: func(phase flow.Phase) {
			if callbacks.onPhaseComplete != nil {
				callbacks.onPhaseComplete(int(phase))
			}
		},
		OnFlowComplete: func(totalTime time.Duration) {
			if callbacks.onFlowComplete != nil {
				callbacks.onFlowComplete(totalTime)
			}
		},
		OnError: func(phase flow.Phase, err error) {
			if callbacks.onError != nil {
				callbacks.onError(int(phase), err)
			}
		},
	}

	controller := flow.NewController(flowCallbacks)
	return &flowControllerAdapter{controller: controller}
}
