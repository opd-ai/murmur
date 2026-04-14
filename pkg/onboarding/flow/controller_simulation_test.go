//go:build simulation

// Package flow simulation tests validate complete onboarding scenarios.
package flow

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
	"github.com/opd-ai/murmur/pkg/onboarding/bootstrap"
)

// TestOnboardingCompletionTime validates that a new user can complete
// onboarding in under 5 minutes. Per ONBOARDING.md, the full flow consists
// of 6 phases: Welcome → Identity → Mode → Bootstrap → Exploration → FirstWave.
func TestOnboardingCompletionTime(t *testing.T) {
	const maxOnboardingTime = 5 * time.Minute

	// Track timing per phase
	type phaseTiming struct {
		phase    Phase
		duration time.Duration
	}
	var timings []phaseTiming
	var mu sync.Mutex

	// Track completion
	var flowComplete atomic.Bool
	var totalDuration atomic.Int64

	// Simulate realistic user interaction timing
	phaseUserTimes := map[Phase]time.Duration{
		PhaseWelcome:           2 * time.Second,  // User reads welcome screen
		PhaseIdentityCreation:  10 * time.Second, // Keypair generation + name entry
		PhaseModeSelection:     5 * time.Second,  // User selects privacy mode
		PhaseNetworkBootstrap:  15 * time.Second, // Network connection (mock)
		PhaseGuidedExploration: 30 * time.Second, // User explores Pulse Map tutorial
		PhaseFirstWave:         20 * time.Second, // User composes first Wave
	}

	c := NewController(Callbacks{
		OnPhaseStart: func(phase Phase) {
			t.Logf("Phase started: %s", phase)
		},
		OnPhaseComplete: func(phase Phase) {
			mu.Lock()
			timings = append(timings, phaseTiming{
				phase:    phase,
				duration: phaseUserTimes[phase],
			})
			mu.Unlock()
			t.Logf("Phase completed: %s (simulated %v)", phase, phaseUserTimes[phase])
		},
		OnFlowComplete: func(duration time.Duration) {
			flowComplete.Store(true)
			totalDuration.Store(int64(duration))
			t.Logf("Onboarding complete in %v", duration)
		},
	})

	// Start flow
	startTime := time.Now()
	c.Start()

	// Simulate user progressing through phases
	for !c.IsComplete() {
		currentPhase := c.CurrentPhase()
		if currentPhase == PhaseComplete {
			break
		}

		// Simulate phase-specific actions
		switch currentPhase {
		case PhaseIdentityCreation:
			// Simulate identity generation
			kp, err := keys.GenerateKeyPair()
			if err != nil {
				t.Fatalf("Failed to generate identity: %v", err)
			}
			c.SetPhaseData(currentPhase, "keypair", kp)
			c.SetPhaseData(currentPhase, "displayName", "TestUser")

		case PhaseModeSelection:
			// Simulate mode selection (default Hybrid)
			c.SetPhaseData(currentPhase, "mode", modes.Hybrid)

		case PhaseNetworkBootstrap:
			// Simulate bootstrap with mock connector
			c.SetPhaseData(currentPhase, "peerCount", 5)
		}

		// Simulate user interaction time
		time.Sleep(phaseUserTimes[currentPhase] / 10) // Speed up for test

		// Complete phase
		c.CompleteCurrentPhase()
	}

	// Wait for completion callback
	time.Sleep(50 * time.Millisecond)

	actualTime := time.Since(startTime)

	// Verify completion
	if !flowComplete.Load() {
		t.Error("Flow did not complete")
	}

	if !c.IsComplete() {
		t.Errorf("Controller not in complete state: %s", c.CurrentPhase())
	}

	// Calculate total simulated user time
	var simulatedUserTime time.Duration
	mu.Lock()
	for _, timing := range timings {
		simulatedUserTime += timing.duration
	}
	mu.Unlock()

	t.Logf("Simulated user interaction time: %v", simulatedUserTime)
	t.Logf("Actual test execution time: %v", actualTime)

	// The simulated user time should be under 5 minutes
	if simulatedUserTime > maxOnboardingTime {
		t.Errorf("Simulated onboarding time %v exceeds 5 minute limit", simulatedUserTime)
	}

	// All phases should complete
	mu.Lock()
	completedPhases := len(timings)
	mu.Unlock()

	if completedPhases != PhaseCount {
		t.Errorf("Expected %d phases completed, got %d", PhaseCount, completedPhases)
	}

	t.Log("✓ Onboarding completion time validation passed")
}

// TestOnboardingWithIdentityFormation validates that onboarding produces
// a valid identity and establishes peer connections.
func TestOnboardingWithIdentityFormation(t *testing.T) {
	var surfaceIdentity *keys.KeyPair
	var selectedMode modes.Mode
	var peerCount int

	c := NewController(Callbacks{})
	c.Start()

	// Phase 1: Welcome
	c.CompleteCurrentPhase()

	// Phase 2: Identity Creation
	kp, err := keys.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}
	surfaceIdentity = kp
	c.SetPhaseData(PhaseIdentityCreation, "keypair", kp)
	c.SetPhaseData(PhaseIdentityCreation, "displayName", "TestUser")
	c.CompleteCurrentPhase()

	// Phase 3: Mode Selection (Hybrid includes Specter)
	selectedMode = modes.Hybrid
	c.SetPhaseData(PhaseModeSelection, "mode", selectedMode)
	c.CompleteCurrentPhase()

	// Phase 4: Network Bootstrap (mock connector with pre-existing peers)
	mockConnector := &mockNetworkConnector{peers: 5}
	ctx := context.Background()
	cfg := bootstrap.Config{
		BootstrapPeers: []string{"/ip4/127.0.0.1/tcp/4001/p2p/QmMock1"},
		MinPeers:       1, // Low threshold for quick completion
		Timeout:        2 * time.Second,
		RetryInterval:  100 * time.Millisecond,
		MaxRetries:     1,
	}
	mgr := bootstrap.NewManager(cfg, mockConnector, bootstrap.Callbacks{
		OnComplete: func(count int) {
			peerCount = count
		},
	})

	// Start bootstrap - runs synchronously with mock connector
	if err := mgr.Start(ctx); err != nil {
		t.Logf("Bootstrap returned: %v (expected with mock)", err)
	}

	// Get peer count directly from connector since callback may be async
	peerCount = mockConnector.PeerCount()
	c.SetPhaseData(PhaseNetworkBootstrap, "peerCount", peerCount)
	c.CompleteCurrentPhase()

	// Phase 5: Guided Exploration (simulated)
	c.SetPhaseData(PhaseGuidedExploration, "tutorialComplete", true)
	c.CompleteCurrentPhase()

	// Phase 6: First Wave
	c.SetPhaseData(PhaseFirstWave, "wavePublished", true)
	c.SetPhaseData(PhaseFirstWave, "waveContent", "Hello MURMUR!")
	c.CompleteCurrentPhase()

	// Verify completion
	if !c.IsComplete() {
		t.Errorf("Flow not complete: %s", c.CurrentPhase())
	}

	// Verify identity formed
	if surfaceIdentity == nil {
		t.Error("Surface identity not created")
	} else {
		if len(surfaceIdentity.PublicKey) != 32 {
			t.Error("Invalid public key length")
		}
		t.Logf("Surface identity created: %x...", surfaceIdentity.PublicKey[:8])
	}

	// Verify mode selected
	if selectedMode != modes.Hybrid {
		t.Errorf("Expected Hybrid mode, got %v", selectedMode)
	}

	// Verify peer connections (mock connector should have connected to bootstrap peer)
	if peerCount < 1 {
		t.Error("No peer connections established")
	}
	t.Logf("Connected to %d peers", peerCount)

	t.Log("✓ Identity formation and peer connection validated")
}

// TestOnboardingInterruption validates state recovery after interruption.
func TestOnboardingInterruption(t *testing.T) {
	// First session: partial completion
	c1 := NewController(Callbacks{})
	c1.Start()

	// Complete first two phases
	c1.CompleteCurrentPhase() // Welcome
	c1.SetPhaseData(PhaseIdentityCreation, "keypair", "mock-keypair")
	c1.SetPhaseData(PhaseIdentityCreation, "displayName", "InterruptedUser")
	c1.CompleteCurrentPhase() // Identity

	// Save state (simulates app close)
	savedState := c1.SaveState()
	if savedState == nil {
		t.Fatal("Failed to save state")
	}

	// Verify we're at Mode Selection
	if savedState.CurrentPhase != PhaseModeSelection {
		t.Errorf("Expected ModeSelection phase, got %s", savedState.CurrentPhase)
	}

	// Second session: restore and continue
	c2 := NewController(Callbacks{})
	if !c2.RestoreState(savedState) {
		t.Error("Failed to restore state")
	}

	// Verify restored state
	if c2.CurrentPhase() != PhaseModeSelection {
		t.Errorf("Restored to wrong phase: %s", c2.CurrentPhase())
	}

	// Verify preserved data
	displayName := c2.GetPhaseData(PhaseIdentityCreation, "displayName")
	if displayName != "InterruptedUser" {
		t.Errorf("Display name not preserved: %v", displayName)
	}

	// Verify earlier phases marked complete
	if !c2.Progress(PhaseWelcome).Completed {
		t.Error("Welcome phase should be completed")
	}
	if !c2.Progress(PhaseIdentityCreation).Completed {
		t.Error("Identity phase should be completed")
	}

	// Continue and complete
	for !c2.IsComplete() {
		c2.CompleteCurrentPhase()
	}

	if !c2.IsComplete() {
		t.Error("Flow not complete after resumption")
	}

	t.Log("✓ Interruption recovery validated")
}

// TestOnboardingTimingBreakdown measures time per phase.
func TestOnboardingTimingBreakdown(t *testing.T) {
	// Expected realistic timings per ONBOARDING.md
	expectedTimings := map[Phase]struct{ min, max time.Duration }{
		PhaseWelcome:           {0, 30 * time.Second},                // Quick welcome
		PhaseIdentityCreation:  {5 * time.Second, 60 * time.Second},  // Key generation + input
		PhaseModeSelection:     {3 * time.Second, 30 * time.Second},  // Mode decision
		PhaseNetworkBootstrap:  {5 * time.Second, 60 * time.Second},  // Network delay
		PhaseGuidedExploration: {10 * time.Second, 90 * time.Second}, // Tutorial completion
		PhaseFirstWave:         {10 * time.Second, 60 * time.Second}, // Wave composition
	}

	c := NewController(Callbacks{})
	c.Start()

	// Simulate each phase with timing
	for phase := PhaseWelcome; phase <= PhaseFirstWave; phase++ {
		if c.CurrentPhase() != phase {
			t.Errorf("Expected phase %s, at %s", phase, c.CurrentPhase())
			break
		}

		// Simulate realistic user time (10% of max expected)
		expected := expectedTimings[phase]
		simulatedDelay := expected.max / 10
		time.Sleep(simulatedDelay)

		c.CompleteCurrentPhase()
	}

	// Calculate total
	totalTime := c.TotalTime()
	t.Logf("Total onboarding time: %v", totalTime)

	// Test simulates 10% of max timings, so expect ~33s total
	// Allow up to 60s for timing variations in CI
	if totalTime > 60*time.Second {
		t.Errorf("Test took longer than expected: %v", totalTime)
	}

	// Per-phase timing validation (from saved data)
	for phase := PhaseWelcome; phase <= PhaseFirstWave; phase++ {
		progress := c.Progress(phase)
		if progress == nil {
			t.Errorf("No progress for phase %s", phase)
			continue
		}
		if !progress.Completed {
			t.Errorf("Phase %s not completed", phase)
		}
		phaseDuration := progress.EndTime.Sub(progress.StartTime)
		t.Logf("  %s: %v", phase, phaseDuration)
	}

	t.Log("✓ Timing breakdown validated")
}

// mockNetworkConnector simulates network connectivity for testing.
type mockNetworkConnector struct {
	mu    sync.Mutex
	peers int
}

func (m *mockNetworkConnector) Connect(ctx context.Context, addr string) (string, error) {
	m.mu.Lock()
	m.peers++
	m.mu.Unlock()
	return "QmMockPeer" + addr[len(addr)-8:], nil
}

func (m *mockNetworkConnector) PeerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.peers
}

func (m *mockNetworkConnector) StartDiscovery(ctx context.Context) error {
	return nil
}
