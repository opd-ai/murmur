package mechanics

import (
	"testing"
	"time"
)

func TestOutcomeVerifierBasic(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 2
	verifier := NewOutcomeVerifier(config)

	var poolID [32]byte
	copy(poolID[:], []byte("test-pool-001"))

	params := MetricParams{
		Topic:     "/murmur/waves/1",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
	}

	// Start verification.
	err := verifier.StartVerification(poolID, MetricGossipVolume, params)
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	state, found := verifier.GetVerificationState(poolID)
	if !found {
		t.Fatal("Verification state not found")
	}
	if state != VerificationCollect {
		t.Errorf("Expected state Collect, got %v", VerificationStateString(state))
	}
}

func TestOutcomeVerifierObservations(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 3
	config.MaxValueDelta = 0.05
	verifier := NewOutcomeVerifier(config)

	var poolID [32]byte
	copy(poolID[:], []byte("test-pool-002"))

	err := verifier.StartVerification(poolID, MetricNodeCount, MetricParams{})
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	// Submit observations with similar values.
	observers := [][32]byte{
		{1}, {2}, {3}, {4},
	}

	values := []float64{100.0, 101.0, 99.0, 100.5}

	for i, obs := range observers {
		observation := &MetricObservation{
			ObserverKey:  obs,
			PoolID:       poolID,
			MetricType:   MetricNodeCount,
			Value:        values[i],
			Timestamp:    time.Now(),
			ObserverHash: computeObservationHash(obs, poolID, values[i]),
		}

		err := verifier.SubmitObservation(observation)
		if err != nil {
			t.Fatalf("SubmitObservation %d failed: %v", i, err)
		}
	}

	// Should have reached consensus.
	state, _ := verifier.GetVerificationState(poolID)
	if state != VerificationConfirmed {
		t.Errorf("Expected Confirmed, got %v", VerificationStateString(state))
	}

	// Finalize and check result.
	result, err := verifier.FinalizeVerification(poolID)
	if err != nil {
		t.Fatalf("FinalizeVerification failed: %v", err)
	}

	if result.State != VerificationConfirmed {
		t.Errorf("Expected confirmed state, got %v", VerificationStateString(result.State))
	}

	// Median of [99, 100, 100.5, 101] = 100.25
	if result.ConfirmedOutcome < 99 || result.ConfirmedOutcome > 102 {
		t.Errorf("Unexpected outcome: %v", result.ConfirmedOutcome)
	}
}

func TestOutcomeVerifierBooleanConsensus(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 3
	verifier := NewOutcomeVerifier(config)

	var poolID [32]byte
	copy(poolID[:], []byte("test-pool-bool"))

	err := verifier.StartVerification(poolID, MetricHuntSuccess, MetricParams{})
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	// Submit boolean observations (majority true).
	observations := []struct {
		key   [32]byte
		value float64
	}{
		{[32]byte{1}, 1.0}, // true
		{[32]byte{2}, 1.0}, // true
		{[32]byte{3}, 0.0}, // false
		{[32]byte{4}, 1.0}, // true
	}

	for _, o := range observations {
		obs := &MetricObservation{
			ObserverKey:  o.key,
			PoolID:       poolID,
			MetricType:   MetricHuntSuccess,
			Value:        o.value,
			Timestamp:    time.Now(),
			ObserverHash: computeObservationHash(o.key, poolID, o.value),
		}
		err := verifier.SubmitObservation(obs)
		if err != nil {
			t.Fatalf("SubmitObservation failed: %v", err)
		}
	}

	result, err := verifier.FinalizeVerification(poolID)
	if err != nil {
		t.Fatalf("FinalizeVerification failed: %v", err)
	}

	if result.ConfirmedOutcome != 1.0 {
		t.Errorf("Expected true (1.0), got %v", result.ConfirmedOutcome)
	}
}

func TestOutcomeVerifierDispute(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 3
	config.MaxValueDelta = 0.01 // Very tight tolerance.
	verifier := NewOutcomeVerifier(config)

	var poolID [32]byte
	copy(poolID[:], []byte("test-pool-dispute"))

	err := verifier.StartVerification(poolID, MetricWaveCount, MetricParams{})
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	// Submit widely varying observations.
	observations := []struct {
		key   [32]byte
		value float64
	}{
		{[32]byte{1}, 100.0},
		{[32]byte{2}, 500.0},
		{[32]byte{3}, 1000.0},
	}

	for _, o := range observations {
		obs := &MetricObservation{
			ObserverKey:  o.key,
			PoolID:       poolID,
			MetricType:   MetricWaveCount,
			Value:        o.value,
			Timestamp:    time.Now(),
			ObserverHash: computeObservationHash(o.key, poolID, o.value),
		}
		_ = verifier.SubmitObservation(obs)
	}

	state, _ := verifier.GetVerificationState(poolID)
	if state != VerificationDisputed {
		t.Errorf("Expected Disputed, got %v", VerificationStateString(state))
	}
}

func TestOutcomeVerifierNotStarted(t *testing.T) {
	verifier := NewOutcomeVerifier(DefaultVerificationConfig())

	var poolID [32]byte
	copy(poolID[:], []byte("nonexistent"))

	obs := &MetricObservation{
		PoolID: poolID,
	}

	err := verifier.SubmitObservation(obs)
	if err != ErrVerificationNotStarted {
		t.Errorf("Expected ErrVerificationNotStarted, got %v", err)
	}
}

func TestOutcomeVerifierDuplicateStart(t *testing.T) {
	verifier := NewOutcomeVerifier(DefaultVerificationConfig())

	var poolID [32]byte
	copy(poolID[:], []byte("duplicate"))

	// First start.
	err := verifier.StartVerification(poolID, MetricNodeCount, MetricParams{})
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Duplicate start should succeed (idempotent).
	err = verifier.StartVerification(poolID, MetricNodeCount, MetricParams{})
	if err != nil {
		t.Errorf("Duplicate start should not error, got: %v", err)
	}
}

func TestGossipVolumeObserver(t *testing.T) {
	observer := &GossipVolumeObserver{
		GetVolume: func(topic string, start, end time.Time) (int64, error) {
			if topic == "/murmur/waves/1" {
				return 5000, nil
			}
			return 0, nil
		},
	}

	if observer.MetricType() != MetricGossipVolume {
		t.Error("Wrong metric type")
	}

	params := MetricParams{
		Topic:     "/murmur/waves/1",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
	}

	value, err := observer.Observe(params)
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 5000 {
		t.Errorf("Expected 5000, got %v", value)
	}
}

func TestTerritoryCountObserver(t *testing.T) {
	observer := &TerritoryCountObserver{
		GetTerritoryCount: func(region string) (int, error) {
			return 15, nil
		},
	}

	if observer.MetricType() != MetricTerritoryCount {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{Region: "east"})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 15 {
		t.Errorf("Expected 15, got %v", value)
	}
}

func TestNodeCountObserver(t *testing.T) {
	observer := &NodeCountObserver{
		GetNodeCount: func() (int, error) {
			return 250, nil
		},
	}

	if observer.MetricType() != MetricNodeCount {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 250 {
		t.Errorf("Expected 250, got %v", value)
	}
}

func TestWaveCountObserver(t *testing.T) {
	observer := &WaveCountObserver{
		GetWaveCount: func(start, end time.Time) (int64, error) {
			return 7500, nil
		},
	}

	if observer.MetricType() != MetricWaveCount {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 7500 {
		t.Errorf("Expected 7500, got %v", value)
	}
}

func TestSpecterCountObserver(t *testing.T) {
	observer := &SpecterCountObserver{
		GetSpecterCount: func() (int, error) {
			return 120, nil
		},
	}

	if observer.MetricType() != MetricSpecterCount {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 120 {
		t.Errorf("Expected 120, got %v", value)
	}
}

func TestEventParticipationObserver(t *testing.T) {
	observer := &EventParticipationObserver{
		GetEventParticipation: func(start, end time.Time) (int, error) {
			return 75, nil
		},
	}

	if observer.MetricType() != MetricEventParticipation {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 75 {
		t.Errorf("Expected 75, got %v", value)
	}
}

func TestHuntSuccessObserver(t *testing.T) {
	observer := &HuntSuccessObserver{
		GetHuntSuccessRate: func(start, end time.Time) (float64, error) {
			return 0.65, nil
		},
	}

	if observer.MetricType() != MetricHuntSuccess {
		t.Error("Wrong metric type")
	}

	value, err := observer.Observe(MetricParams{})
	if err != nil {
		t.Fatalf("Observe failed: %v", err)
	}
	if value != 0.65 {
		t.Errorf("Expected 0.65, got %v", value)
	}
}

func TestObserverMissingCallback(t *testing.T) {
	observers := []MetricObserver{
		&GossipVolumeObserver{},
		&TerritoryCountObserver{},
		&NodeCountObserver{},
		&WaveCountObserver{},
		&SpecterCountObserver{},
		&EventParticipationObserver{},
		&HuntSuccessObserver{},
	}

	for _, obs := range observers {
		_, err := obs.Observe(MetricParams{})
		if err != ErrMissingObservableData {
			t.Errorf("Expected ErrMissingObservableData for %s, got %v",
				MetricTypeString(obs.MetricType()), err)
		}
	}
}

func TestMetricTypeString(t *testing.T) {
	tests := []struct {
		mt       ObservableMetricType
		expected string
	}{
		{MetricGossipVolume, "GossipVolume"},
		{MetricTerritoryCount, "TerritoryCount"},
		{MetricEventParticipation, "EventParticipation"},
		{MetricNodeCount, "NodeCount"},
		{MetricWaveCount, "WaveCount"},
		{MetricSpecterCount, "SpecterCount"},
		{MetricHuntSuccess, "HuntSuccess"},
		{MetricCustom, "Custom"},
		{ObservableMetricType(99), "Unknown"},
	}

	for _, tt := range tests {
		result := MetricTypeString(tt.mt)
		if result != tt.expected {
			t.Errorf("MetricTypeString(%d) = %s, want %s", tt.mt, result, tt.expected)
		}
	}
}

func TestVerificationStateString(t *testing.T) {
	tests := []struct {
		vs       VerificationState
		expected string
	}{
		{VerificationPending, "Pending"},
		{VerificationCollect, "Collecting"},
		{VerificationConsensus, "Consensus"},
		{VerificationConfirmed, "Confirmed"},
		{VerificationDisputed, "Disputed"},
		{VerificationFailed, "Failed"},
		{VerificationState(99), "Unknown"},
	}

	for _, tt := range tests {
		result := VerificationStateString(tt.vs)
		if result != tt.expected {
			t.Errorf("VerificationStateString(%d) = %s, want %s", tt.vs, result, tt.expected)
		}
	}
}

func TestComputeMedian(t *testing.T) {
	tests := []struct {
		values   []float64
		expected float64
	}{
		{[]float64{}, 0},
		{[]float64{5}, 5},
		{[]float64{1, 3}, 2},
		{[]float64{1, 2, 3}, 2},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{5, 1, 3, 2, 4}, 3},
	}

	for _, tt := range tests {
		result := computeMedian(tt.values)
		if result != tt.expected {
			t.Errorf("computeMedian(%v) = %v, want %v", tt.values, result, tt.expected)
		}
	}
}

func TestFindConsensusValue(t *testing.T) {
	tests := []struct {
		values   []float64
		delta    float64
		expected float64
		found    bool
	}{
		// Boolean values.
		{[]float64{1, 1, 1, 0}, 0.01, 1, true},
		{[]float64{0, 0, 0, 1}, 0.01, 0, true},
		{[]float64{1, 0}, 0.01, 0, true}, // Tie goes to 0 (ones=1 is not > len/2=1).

		// Numeric values with agreement.
		{[]float64{100, 101, 99}, 0.05, 100, true},

		// Numeric values without agreement.
		{[]float64{100, 500, 1000}, 0.01, 0, false},
	}

	for i, tt := range tests {
		result, found := findConsensusValue(tt.values, tt.delta)
		if found != tt.found {
			t.Errorf("Test %d: findConsensusValue found=%v, want %v", i, found, tt.found)
		}
		if found && result != tt.expected {
			t.Errorf("Test %d: findConsensusValue=%v, want %v", i, result, tt.expected)
		}
	}
}

func TestCountConfirmations(t *testing.T) {
	values := []float64{100, 101, 99, 500}
	consensus := 100.0
	delta := 0.05

	count := countConfirmations(values, consensus, delta)
	if count != 3 { // 100, 101, 99 are within 5% of 100; 500 is not.
		t.Errorf("Expected 3 confirmations, got %d", count)
	}
}

func TestVerifierWithRegisteredObserver(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 1
	verifier := NewOutcomeVerifier(config)

	// Register an observer.
	observer := &NodeCountObserver{
		GetNodeCount: func() (int, error) {
			return 500, nil
		},
	}
	verifier.RegisterObserver(observer)

	var poolID [32]byte
	copy(poolID[:], []byte("observer-test"))

	err := verifier.StartVerification(poolID, MetricNodeCount, MetricParams{})
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	var observerKey [32]byte
	copy(observerKey[:], []byte("observer-1"))

	sign := func(data []byte) [64]byte {
		var sig [64]byte
		copy(sig[:], data) // Mock signature.
		return sig
	}

	err = verifier.ObserveAndSubmit(poolID, observerKey, sign)
	if err != nil {
		t.Fatalf("ObserveAndSubmit failed: %v", err)
	}

	result, err := verifier.FinalizeVerification(poolID)
	if err != nil {
		t.Fatalf("FinalizeVerification failed: %v", err)
	}

	if result.ConfirmedOutcome != 500 {
		t.Errorf("Expected 500, got %v", result.ConfirmedOutcome)
	}
}

func TestVerificationResult(t *testing.T) {
	config := DefaultVerificationConfig()
	config.MinConfirmations = 2
	verifier := NewOutcomeVerifier(config)

	var poolID [32]byte
	copy(poolID[:], []byte("result-test"))

	err := verifier.StartVerification(poolID, MetricNodeCount, MetricParams{})
	if err != nil {
		t.Fatalf("StartVerification failed: %v", err)
	}

	// Submit observations.
	for i := 0; i < 3; i++ {
		var key [32]byte
		key[0] = byte(i + 1)
		obs := &MetricObservation{
			ObserverKey:  key,
			PoolID:       poolID,
			MetricType:   MetricNodeCount,
			Value:        200.0,
			Timestamp:    time.Now(),
			ObserverHash: computeObservationHash(key, poolID, 200.0),
		}
		_ = verifier.SubmitObservation(obs)
	}

	// Get result.
	result, found := verifier.GetResult(poolID)
	if found {
		t.Error("Result should not be found before finalization")
	}

	result, err = verifier.FinalizeVerification(poolID)
	if err != nil {
		t.Fatalf("FinalizeVerification failed: %v", err)
	}

	result2, found := verifier.GetResult(poolID)
	if !found {
		t.Error("Result should be found after finalization")
	}

	if result2.ConfirmedOutcome != result.ConfirmedOutcome {
		t.Error("Results don't match")
	}
}
