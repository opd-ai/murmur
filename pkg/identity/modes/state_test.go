package modes

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{Open, "Open"},
		{Hybrid, "Hybrid"},
		{Guarded, "Guarded"},
		{Fortress, "Fortress"},
		{Mode(99), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.mode.String(); got != tc.want {
			t.Errorf("%d.String() = %s, want %s", tc.mode, got, tc.want)
		}
	}
}

func TestModeAllowsSurface(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{Open, true},
		{Hybrid, true},
		{Guarded, true},
		{Fortress, false},
	}

	for _, tc := range tests {
		if got := tc.mode.AllowsSurface(); got != tc.want {
			t.Errorf("%s.AllowsSurface() = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestModeAllowsSpecter(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{Open, false},
		{Hybrid, true},
		{Guarded, true},
		{Fortress, true},
	}

	for _, tc := range tests {
		if got := tc.mode.AllowsSpecter(); got != tc.want {
			t.Errorf("%s.AllowsSpecter() = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestModeRequiresShroud(t *testing.T) {
	tests := []struct {
		mode Mode
		want bool
	}{
		{Open, false},
		{Hybrid, false},
		{Guarded, false},
		{Fortress, true},
	}

	for _, tc := range tests {
		if got := tc.mode.RequiresShroud(); got != tc.want {
			t.Errorf("%s.RequiresShroud() = %v, want %v", tc.mode, got, tc.want)
		}
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager()

	if m == nil {
		t.Fatal("manager is nil")
	}

	if m.Current() != Open {
		t.Errorf("initial mode = %s, want Open", m.Current())
	}
}

func TestNewManagerWithConfig(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, time.Millisecond)

	if m.Current() != Hybrid {
		t.Errorf("initial mode = %s, want Hybrid", m.Current())
	}
}

func TestTransitionAdjacent(t *testing.T) {
	m := NewManagerWithConfig(Open, 0)
	m.SetSpecterAvailable(true)

	// Open -> Hybrid should work.
	if err := m.Transition(Hybrid); err != nil {
		t.Errorf("Open -> Hybrid failed: %v", err)
	}

	// Hybrid -> Guarded should work.
	if err := m.Transition(Guarded); err != nil {
		t.Errorf("Hybrid -> Guarded failed: %v", err)
	}

	m.SetShroudAvailable(true)

	// Guarded -> Fortress should work.
	if err := m.Transition(Fortress); err != nil {
		t.Errorf("Guarded -> Fortress failed: %v", err)
	}

	// Fortress -> Guarded should work.
	if err := m.Transition(Guarded); err != nil {
		t.Errorf("Fortress -> Guarded failed: %v", err)
	}
}

func TestTransitionSkipNotAllowed(t *testing.T) {
	m := NewManagerWithConfig(Open, 0)
	m.SetSpecterAvailable(true)
	m.SetShroudAvailable(true)

	// Open -> Guarded (skipping Hybrid) should fail.
	err := m.Transition(Guarded)
	if err != ErrInvalidTransition {
		t.Errorf("Open -> Guarded should fail: got %v", err)
	}
}

func TestTransitionToOpen(t *testing.T) {
	m := NewManagerWithConfig(Fortress, 0)
	m.SetSpecterAvailable(true)
	m.SetShroudAvailable(true)

	// Fortress -> Open should work (emergency exit).
	if err := m.Transition(Open); err != nil {
		t.Errorf("Fortress -> Open failed: %v", err)
	}

	if m.Current() != Open {
		t.Errorf("mode = %s, want Open", m.Current())
	}
}

func TestTransitionDestroysSpecter(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, 0)
	m.SetSpecterAvailable(true)

	// Track if destroyer was called.
	destroyed := false
	m.SetSpecterDestroyer(func() error {
		destroyed = true
		return nil
	})

	// Transition from Hybrid (Specter-enabled) to Open (no Specter).
	if err := m.Transition(Open); err != nil {
		t.Fatalf("Hybrid -> Open failed: %v", err)
	}

	if !destroyed {
		t.Error("Specter destroyer was not called on Hybrid -> Open transition")
	}

	// hasSpecter should be set to false.
	m.SetSpecterAvailable(true) // Try to set it true.
	// But transition should have cleared it, so we can't verify directly.
	// The API doesn't expose hasSpecter, but we can check if we can transition
	// back to Hybrid (which requires Specter).
}

func TestTransitionToOpenPreservesSpecterIfDestroyed(t *testing.T) {
	m := NewManagerWithConfig(Guarded, 0)
	m.SetSpecterAvailable(true)

	// Set destroyer that clears Specter.
	m.SetSpecterDestroyer(func() error {
		return nil
	})

	// Transition to Open.
	if err := m.Transition(Open); err != nil {
		t.Fatalf("Guarded -> Open failed: %v", err)
	}

	// After transition, hasSpecter should be false, so we can't transition
	// back to Hybrid without re-setting it.
	err := m.Transition(Hybrid)
	if err != ErrMissingRequirement {
		t.Errorf("expected ErrMissingRequirement (no Specter), got %v", err)
	}
}

func TestTransitionSpecterDestroyerFails(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, 0)
	m.SetSpecterAvailable(true)

	// Destroyer that fails.
	m.SetSpecterDestroyer(func() error {
		return errors.New("destruction failed")
	})

	// Transition should fail if destroyer fails.
	err := m.Transition(Open)
	if err != ErrSpecterDestructionFailed {
		t.Errorf("expected ErrSpecterDestructionFailed, got %v", err)
	}

	// Mode should remain unchanged.
	if m.Current() != Hybrid {
		t.Errorf("mode = %s, want Hybrid (unchanged after failed destruction)", m.Current())
	}
}

func TestTransitionNoDestroyerWhenSpecterStaysEnabled(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, 0)
	m.SetSpecterAvailable(true)

	// Track if destroyer was called.
	destroyed := false
	m.SetSpecterDestroyer(func() error {
		destroyed = true
		return nil
	})

	// Transition from Hybrid to Guarded (both Specter-enabled).
	if err := m.Transition(Guarded); err != nil {
		t.Fatalf("Hybrid -> Guarded failed: %v", err)
	}

	if destroyed {
		t.Error("Specter destroyer should NOT be called when staying in Specter-enabled mode")
	}
}

func TestTransitionCooldown(t *testing.T) {
	m := NewManagerWithConfig(Open, 50*time.Millisecond)
	m.SetSpecterAvailable(true)

	// First transition should work.
	if err := m.Transition(Hybrid); err != nil {
		t.Fatalf("first transition failed: %v", err)
	}

	// Immediate second transition should fail.
	err := m.Transition(Guarded)
	if err != ErrCooldownActive {
		t.Errorf("expected ErrCooldownActive, got %v", err)
	}

	// Wait for cooldown.
	time.Sleep(60 * time.Millisecond)

	// Should work now.
	if err := m.Transition(Guarded); err != nil {
		t.Errorf("transition after cooldown failed: %v", err)
	}
}

func TestTransitionMissingSpecter(t *testing.T) {
	m := NewManagerWithConfig(Open, 0)
	// hasSpecter is false by default.

	err := m.Transition(Hybrid)
	if err != ErrMissingRequirement {
		t.Errorf("expected ErrMissingRequirement, got %v", err)
	}
}

func TestTransitionMissingShroud(t *testing.T) {
	m := NewManagerWithConfig(Guarded, 0)
	m.SetSpecterAvailable(true)
	// hasShroud is false by default.

	err := m.Transition(Fortress)
	if err != ErrMissingRequirement {
		t.Errorf("expected ErrMissingRequirement, got %v", err)
	}
}

func TestForceTransition(t *testing.T) {
	m := NewManager()

	// Force skip validation.
	m.ForceTransition(Fortress)

	if m.Current() != Fortress {
		t.Errorf("mode = %s, want Fortress", m.Current())
	}
}

func TestOnTransition(t *testing.T) {
	m := NewManagerWithConfig(Open, 0)
	m.SetSpecterAvailable(true)

	var received struct {
		mu  sync.Mutex
		old Mode
		new Mode
	}

	m.OnTransition(func(old, new Mode) {
		received.mu.Lock()
		defer received.mu.Unlock()
		received.old = old
		received.new = new
	})

	m.Transition(Hybrid)

	// Give goroutine time to execute.
	time.Sleep(10 * time.Millisecond)

	received.mu.Lock()
	if received.old != Open || received.new != Hybrid {
		t.Errorf("callback received (%s, %s), want (Open, Hybrid)",
			received.old, received.new)
	}
	received.mu.Unlock()
}

func TestCooldownRemaining(t *testing.T) {
	cooldown := 100 * time.Millisecond
	m := NewManagerWithConfig(Open, cooldown)
	m.SetSpecterAvailable(true)

	m.Transition(Hybrid)

	remaining := m.CooldownRemaining()
	if remaining <= 0 || remaining > cooldown {
		t.Errorf("remaining = %v, want (0, %v]", remaining, cooldown)
	}

	time.Sleep(cooldown + 10*time.Millisecond)

	if m.CooldownRemaining() != 0 {
		t.Errorf("remaining should be 0 after cooldown")
	}
}

func TestAvailableModes(t *testing.T) {
	m := NewManagerWithConfig(Open, 0)

	// Without Specter, only Open is available.
	available := m.AvailableModes()
	if len(available) != 1 || available[0] != Open {
		t.Errorf("expected [Open], got %v", available)
	}

	// With Specter, Hybrid becomes available.
	m.SetSpecterAvailable(true)
	available = m.AvailableModes()
	if len(available) != 2 {
		t.Errorf("expected 2 modes, got %d", len(available))
	}
}

func TestModeCapabilities(t *testing.T) {
	caps := Fortress.Capabilities()

	if caps.SurfaceAllowed {
		t.Error("Fortress should not allow Surface")
	}
	if !caps.SpecterAllowed {
		t.Error("Fortress should allow Specter")
	}
	if !caps.ShroudRequired {
		t.Error("Fortress should require Shroud")
	}
	if caps.AnonymityLevel != 3 {
		t.Errorf("Fortress anonymity = %d, want 3", caps.AnonymityLevel)
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from  Mode
		to    Mode
		valid bool
	}{
		{Open, Open, true},
		{Open, Hybrid, true},
		{Open, Guarded, false}, // skip
		{Open, Fortress, false},
		{Hybrid, Open, true},
		{Hybrid, Guarded, true},
		{Hybrid, Fortress, false}, // skip
		{Fortress, Open, true},    // emergency exit
		{Fortress, Hybrid, true},
		{Fortress, Guarded, true},
	}

	for _, tc := range tests {
		got := isValidTransition(tc.from, tc.to)
		if got != tc.valid {
			t.Errorf("isValidTransition(%s, %s) = %v, want %v",
				tc.from, tc.to, got, tc.valid)
		}
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManagerWithConfig(Open, time.Millisecond)
	m.SetSpecterAvailable(true)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Current()
			m.AvailableModes()
			m.CooldownRemaining()
		}()
	}
	wg.Wait()
}

func TestRequiresTrafficPadding(t *testing.T) {
	tests := []struct {
		mode     Mode
		requires bool
	}{
		{Open, false},
		{Hybrid, false},
		{Guarded, true},
		{Fortress, true},
	}

	for _, tc := range tests {
		if got := tc.mode.RequiresTrafficPadding(); got != tc.requires {
			t.Errorf("%s.RequiresTrafficPadding() = %v, want %v", tc.mode, got, tc.requires)
		}
	}
}

func TestModeCapabilitiesTrafficPadding(t *testing.T) {
	tests := []struct {
		mode    Mode
		padding bool
	}{
		{Open, false},
		{Hybrid, false},
		{Guarded, true},
		{Fortress, true},
	}

	for _, tc := range tests {
		caps := tc.mode.Capabilities()
		if caps.TrafficPaddingActive != tc.padding {
			t.Errorf("%s.Capabilities().TrafficPaddingActive = %v, want %v",
				tc.mode, caps.TrafficPaddingActive, tc.padding)
		}
	}
}

func TestTrafficPaddingStartsOnGuardedTransition(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, 0)
	m.SetSpecterAvailable(true)

	started := false
	m.SetTrafficPaddingCallbacks(
		func() error { started = true; return nil },
		func() error { return nil },
	)

	// Transition to Guarded should start traffic padding.
	if err := m.Transition(Guarded); err != nil {
		t.Fatalf("Hybrid -> Guarded failed: %v", err)
	}

	if !started {
		t.Error("Traffic padding was not started on Hybrid -> Guarded transition")
	}

	if !m.IsTrafficPaddingEnabled() {
		t.Error("IsTrafficPaddingEnabled should be true after entering Guarded")
	}
}

func TestTrafficPaddingStartsOnFortressTransition(t *testing.T) {
	m := NewManagerWithConfig(Guarded, 0)
	m.SetSpecterAvailable(true)
	m.SetShroudAvailable(true)

	started := false
	m.SetTrafficPaddingCallbacks(
		func() error { started = true; return nil },
		func() error { return nil },
	)

	// Transition from Guarded to Fortress - padding already enabled, shouldn't restart.
	m.paddingEnabled = true // Simulating existing padding state
	if err := m.Transition(Fortress); err != nil {
		t.Fatalf("Guarded -> Fortress failed: %v", err)
	}

	// Since both modes require padding, starter should NOT be called again.
	if started {
		t.Error("Traffic padding starter should NOT be called when staying in padding-enabled modes")
	}
}

func TestTrafficPaddingStopsOnOpenTransition(t *testing.T) {
	m := NewManagerWithConfig(Guarded, 0)
	m.SetSpecterAvailable(true)
	m.paddingEnabled = true

	stopped := false
	m.SetTrafficPaddingCallbacks(
		func() error { return nil },
		func() error { stopped = true; return nil },
	)

	// Transition to Hybrid should stop traffic padding.
	if err := m.Transition(Hybrid); err != nil {
		t.Fatalf("Guarded -> Hybrid failed: %v", err)
	}

	if !stopped {
		t.Error("Traffic padding was not stopped on Guarded -> Hybrid transition")
	}

	if m.IsTrafficPaddingEnabled() {
		t.Error("IsTrafficPaddingEnabled should be false after leaving Guarded")
	}
}

func TestTrafficPaddingStarterError(t *testing.T) {
	m := NewManagerWithConfig(Hybrid, 0)
	m.SetSpecterAvailable(true)

	m.SetTrafficPaddingCallbacks(
		func() error { return errors.New("start failed") },
		func() error { return nil },
	)

	// Transition should fail if padding starter fails.
	err := m.Transition(Guarded)
	if err != ErrTrafficPaddingStartErr {
		t.Errorf("expected ErrTrafficPaddingStartErr, got %v", err)
	}

	// Mode should remain unchanged.
	if m.Current() != Hybrid {
		t.Errorf("mode = %s, want Hybrid (unchanged after failed padding start)", m.Current())
	}
}

func TestTrafficPaddingStopperError(t *testing.T) {
	m := NewManagerWithConfig(Guarded, 0)
	m.SetSpecterAvailable(true)
	m.paddingEnabled = true

	m.SetTrafficPaddingCallbacks(
		func() error { return nil },
		func() error { return errors.New("stop failed") },
	)

	// Transition should fail if padding stopper fails.
	err := m.Transition(Hybrid)
	if err != ErrTrafficPaddingStopErr {
		t.Errorf("expected ErrTrafficPaddingStopErr, got %v", err)
	}

	// Mode should remain unchanged.
	if m.Current() != Guarded {
		t.Errorf("mode = %s, want Guarded (unchanged after failed padding stop)", m.Current())
	}
}
