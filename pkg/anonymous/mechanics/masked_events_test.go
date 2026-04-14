package mechanics

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestMaskedEventStateString(t *testing.T) {
	tests := []struct {
		state MaskedEventState
		want  string
	}{
		{MaskedEventPending, "Pending"},
		{MaskedEventActive, "Active"},
		{MaskedEventEnded, "Ended"},
		{MaskedEventState(99), "Unknown"},
	}

	for _, tc := range tests {
		got := MaskedEventStateString(tc.state)
		if got != tc.want {
			t.Errorf("MaskedEventStateString(%d) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestIsValidMaskedEventDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     bool
	}{
		{30 * time.Minute, true},
		{60 * time.Minute, true},
		{120 * time.Minute, true},
		{240 * time.Minute, true},
		{45 * time.Minute, false},
		{15 * time.Minute, false},
		{300 * time.Minute, false},
	}

	for _, tc := range tests {
		got := IsValidMaskedEventDuration(tc.duration)
		if got != tc.want {
			t.Errorf("IsValidMaskedEventDuration(%v) = %v, want %v", tc.duration, got, tc.want)
		}
	}
}

func TestNewMaskedEvent(t *testing.T) {
	creator := [32]byte{1, 2, 3}
	startTime := time.Now().Add(1 * time.Hour)

	event, err := NewMaskedEvent(creator, "Test Event", startTime, 30*time.Minute, 10)
	if err != nil {
		t.Fatalf("NewMaskedEvent failed: %v", err)
	}

	if event.Topic != "Test Event" {
		t.Errorf("Topic = %q, want %q", event.Topic, "Test Event")
	}
	if event.CreatorSpecterKey != creator {
		t.Error("CreatorSpecterKey mismatch")
	}
	if event.Duration != 30*time.Minute {
		t.Errorf("Duration = %v, want %v", event.Duration, 30*time.Minute)
	}
	if event.MaxParticipants != 10 {
		t.Errorf("MaxParticipants = %d, want 10", event.MaxParticipants)
	}
	if event.State != MaskedEventPending {
		t.Errorf("Initial state = %v, want MaskedEventPending", event.State)
	}
}

func TestNewMaskedEvent_TopicTooLong(t *testing.T) {
	creator := [32]byte{1}
	longTopic := make([]byte, 257)
	for i := range longTopic {
		longTopic[i] = 'x'
	}

	_, err := NewMaskedEvent(creator, string(longTopic), time.Now(), 30*time.Minute, 10)
	if err != ErrMaskedEventTopicTooLong {
		t.Errorf("Expected ErrMaskedEventTopicTooLong, got %v", err)
	}
}

func TestNewMaskedEvent_InvalidDuration(t *testing.T) {
	creator := [32]byte{1}
	_, err := NewMaskedEvent(creator, "Test", time.Now(), 45*time.Minute, 10)
	if err != ErrMaskedEventInvalidDuration {
		t.Errorf("Expected ErrMaskedEventInvalidDuration, got %v", err)
	}
}

func TestNewMaskedEvent_InvalidParticipants(t *testing.T) {
	creator := [32]byte{1}

	// Too few (not 0, but less than 5).
	_, err := NewMaskedEvent(creator, "Test", time.Now(), 30*time.Minute, 3)
	if err != ErrMaskedEventInvalidParticipants {
		t.Errorf("Expected ErrMaskedEventInvalidParticipants for 3, got %v", err)
	}

	// Too many.
	_, err = NewMaskedEvent(creator, "Test", time.Now(), 30*time.Minute, 101)
	if err != ErrMaskedEventInvalidParticipants {
		t.Errorf("Expected ErrMaskedEventInvalidParticipants for 101, got %v", err)
	}

	// 0 (unlimited) is valid.
	_, err = NewMaskedEvent(creator, "Test", time.Now(), 30*time.Minute, 0)
	if err != nil {
		t.Errorf("0 participants should be valid (unlimited), got %v", err)
	}
}

func TestMaskedEvent_GossipTopic(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now(), 30*time.Minute, 10)

	topic := event.GossipTopic()
	if len(topic) == 0 {
		t.Error("GossipTopic returned empty string")
	}
	// Should start with /murmur/event/
	expected := "/murmur/event/"
	if topic[:len(expected)] != expected {
		t.Errorf("GossipTopic = %q, should start with %q", topic, expected)
	}
}

func TestMaskedEvent_Join(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2, 3, 4}
	keypair, err := event.Join(specter)
	if err != nil {
		t.Fatalf("Join failed: %v", err)
	}

	if keypair == nil {
		t.Fatal("Join returned nil keypair")
	}
	if len(keypair.Pseudonym) == 0 {
		t.Error("Keypair has empty pseudonym")
	}
	if event.ParticipantCount() != 1 {
		t.Errorf("ParticipantCount = %d, want 1", event.ParticipantCount())
	}
}

func TestMaskedEvent_Join_AlreadyJoined(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	event.Join(specter)

	// Try to join again.
	_, err := event.Join(specter)
	if err != ErrMaskedEventAlreadyJoined {
		t.Errorf("Expected ErrMaskedEventAlreadyJoined, got %v", err)
	}
}

func TestMaskedEvent_Join_Full(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 5)

	// Fill the event.
	for i := 0; i < 5; i++ {
		specter := [32]byte{byte(i + 10)}
		_, err := event.Join(specter)
		if err != nil {
			t.Fatalf("Join %d failed: %v", i, err)
		}
	}

	// Try to join when full.
	_, err := event.Join([32]byte{99})
	if err != ErrMaskedEventFull {
		t.Errorf("Expected ErrMaskedEventFull, got %v", err)
	}
}

func TestMaskedEvent_Join_Ended(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)
	event.End()

	_, err := event.Join([32]byte{2})
	if err != ErrMaskedEventEnded {
		t.Errorf("Expected ErrMaskedEventEnded, got %v", err)
	}
}

func TestMaskedEvent_Start(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	err := event.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !event.IsActive() {
		t.Error("Event should be active after Start")
	}
}

func TestMaskedEvent_Start_AlreadyStarted(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)
	event.Start()

	err := event.Start()
	if err != ErrMaskedEventAlreadyStarted {
		t.Errorf("Expected ErrMaskedEventAlreadyStarted, got %v", err)
	}
}

func TestMaskedEvent_End(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)
	event.Start()
	event.End()

	if !event.IsEnded() {
		t.Error("Event should be ended after End")
	}
}

func TestMaskedEvent_Update(t *testing.T) {
	creator := [32]byte{1}
	// Set start time in the past.
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(-1*time.Second), 30*time.Minute, 10)

	event.Update()

	if !event.IsActive() {
		t.Error("Event should become active after Update past start time")
	}
}

func TestMaskedEvent_OnStateChange(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	var changes int32
	event.OnStateChange(func(old, new MaskedEventState) {
		atomic.AddInt32(&changes, 1)
	})

	event.Start()
	event.End()

	if atomic.LoadInt32(&changes) != 2 {
		t.Errorf("OnStateChange called %d times, want 2", atomic.LoadInt32(&changes))
	}
}

func TestMaskedEvent_HasParticipant(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	if !event.HasParticipant(keypair.PublicKey) {
		t.Error("HasParticipant should return true for joined participant")
	}

	if event.HasParticipant([32]byte{99}) {
		t.Error("HasParticipant should return false for non-participant")
	}
}

func TestMaskedEvent_RecordWave(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	err := event.RecordWave(keypair.PublicKey)
	if err != nil {
		t.Fatalf("RecordWave failed: %v", err)
	}

	p := event.GetParticipant(keypair.PublicKey)
	if p.WaveCount != 1 {
		t.Errorf("WaveCount = %d, want 1", p.WaveCount)
	}
}

func TestMaskedEvent_RecordWave_NotJoined(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	err := event.RecordWave([32]byte{99})
	if err != ErrMaskedEventNotJoined {
		t.Errorf("Expected ErrMaskedEventNotJoined, got %v", err)
	}
}

func TestMaskedEvent_RecordAmplification(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	err := event.RecordAmplification(keypair.PublicKey)
	if err != nil {
		t.Fatalf("RecordAmplification failed: %v", err)
	}

	p := event.GetParticipant(keypair.PublicKey)
	if p.AmplificationsReceived != 1 {
		t.Errorf("AmplificationsReceived = %d, want 1", p.AmplificationsReceived)
	}
}

func TestMaskedEvent_TimeRemaining(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	remaining := event.TimeRemaining()
	if remaining <= 0 || remaining > 1*time.Hour {
		t.Errorf("TimeRemaining (pending) = %v, expected in (0, 1h]", remaining)
	}

	event.Start()
	remaining = event.TimeRemaining()
	if remaining <= 0 || remaining > 30*time.Minute {
		t.Errorf("TimeRemaining (active) = %v, expected in (0, 30m]", remaining)
	}

	event.End()
	remaining = event.TimeRemaining()
	if remaining != 0 {
		t.Errorf("TimeRemaining (ended) = %v, want 0", remaining)
	}
}

func TestMaskedEvent_RegisterMaskedKey(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	maskedKey := [32]byte{10, 20, 30}
	err := event.RegisterMaskedKey(maskedKey, "Test Pseudonym")
	if err != nil {
		t.Fatalf("RegisterMaskedKey failed: %v", err)
	}

	if !event.HasParticipant(maskedKey) {
		t.Error("Registered key should be a participant")
	}
}

func TestMaskedEvent_GenerateSummary(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test Topic", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	// Add participants with activity.
	specter1 := [32]byte{2}
	specter2 := [32]byte{3}
	kp1, _ := event.Join(specter1)
	kp2, _ := event.Join(specter2)

	event.RecordWave(kp1.PublicKey)
	event.RecordWave(kp1.PublicKey)
	event.RecordWave(kp2.PublicKey)
	event.RecordAmplification(kp1.PublicKey)
	event.RecordAmplification(kp1.PublicKey)
	event.RecordAmplification(kp2.PublicKey)

	event.End()

	summary := event.GenerateSummary()

	if summary.Topic != "Test Topic" {
		t.Errorf("Summary topic = %q, want %q", summary.Topic, "Test Topic")
	}
	if summary.ParticipantCount != 2 {
		t.Errorf("ParticipantCount = %d, want 2", summary.ParticipantCount)
	}
	if summary.TotalWaves != 3 {
		t.Errorf("TotalWaves = %d, want 3", summary.TotalWaves)
	}
	if summary.TotalAmplifications != 3 {
		t.Errorf("TotalAmplifications = %d, want 3", summary.TotalAmplifications)
	}
	if len(summary.Leaderboard) != 2 {
		t.Errorf("Leaderboard len = %d, want 2", len(summary.Leaderboard))
	}
	// First should have more amplifications.
	if summary.Leaderboard[0].AmplificationsReceived < summary.Leaderboard[1].AmplificationsReceived {
		t.Error("Leaderboard should be sorted by amplifications descending")
	}
}

// MaskedKeypair tests.

func TestMaskedKeypair_Sign(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	data := []byte("test message")
	sig, err := keypair.Sign(data)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if len(sig) != 64 {
		t.Errorf("Signature length = %d, want 64", len(sig))
	}
}

func TestMaskedKeypair_Sign_Destroyed(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)
	keypair.Destroy()

	_, err := keypair.Sign([]byte("test"))
	if err != ErrMaskedEventKeyDestroyed {
		t.Errorf("Expected ErrMaskedEventKeyDestroyed, got %v", err)
	}
}

func TestMaskedKeypair_GetX25519PublicKey(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	pub, err := keypair.GetX25519PublicKey()
	if err != nil {
		t.Fatalf("GetX25519PublicKey failed: %v", err)
	}
	if pub == [32]byte{} {
		t.Error("X25519 public key should not be zero")
	}
}

func TestMaskedKeypair_ComputeSharedSecret(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	// Create two keypairs.
	kp1, _ := event.Join([32]byte{2})
	kp2, _ := event.Join([32]byte{3})

	pub1, _ := kp1.GetX25519PublicKey()
	pub2, _ := kp2.GetX25519PublicKey()

	shared1, err := kp1.ComputeSharedSecret(pub2)
	if err != nil {
		t.Fatalf("ComputeSharedSecret 1 failed: %v", err)
	}

	shared2, err := kp2.ComputeSharedSecret(pub1)
	if err != nil {
		t.Fatalf("ComputeSharedSecret 2 failed: %v", err)
	}

	// Shared secrets should be equal.
	if shared1 != shared2 {
		t.Error("Shared secrets should match")
	}
}

func TestMaskedKeypair_Destroy(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	specter := [32]byte{2}
	keypair, _ := event.Join(specter)

	keypair.Destroy()

	if !keypair.IsDestroyed() {
		t.Error("IsDestroyed should return true after Destroy")
	}

	// Sign should fail.
	_, err := keypair.Sign([]byte("test"))
	if err != ErrMaskedEventKeyDestroyed {
		t.Errorf("Sign after Destroy should fail with ErrMaskedEventKeyDestroyed, got %v", err)
	}
}

func TestGenerateMaskedPseudonym(t *testing.T) {
	pubKey := [32]byte{1, 2, 3, 4, 5}
	pseudonym := GenerateMaskedPseudonym(pubKey)

	if len(pseudonym) == 0 {
		t.Error("GenerateMaskedPseudonym returned empty string")
	}

	// Should be two words separated by space.
	words := 0
	for _, c := range pseudonym {
		if c == ' ' {
			words++
		}
	}
	if words != 1 {
		t.Errorf("Pseudonym %q should have exactly one space", pseudonym)
	}

	// Same key should produce same pseudonym.
	pseudonym2 := GenerateMaskedPseudonym(pubKey)
	if pseudonym != pseudonym2 {
		t.Error("Same key should produce same pseudonym")
	}
}

func TestCalculateResonanceBurst(t *testing.T) {
	tests := []struct {
		amplifications int
		wantPositive   bool
	}{
		{0, false},
		{-1, false},
		{1, true},
		{10, true},
		{100, true},
	}

	for _, tc := range tests {
		burst := CalculateResonanceBurst(tc.amplifications)
		if tc.wantPositive && burst <= 0 {
			t.Errorf("CalculateResonanceBurst(%d) = %f, want positive", tc.amplifications, burst)
		}
		if !tc.wantPositive && burst > 0 {
			t.Errorf("CalculateResonanceBurst(%d) = %f, want 0", tc.amplifications, burst)
		}
	}

	// More amplifications should give higher burst.
	burst10 := CalculateResonanceBurst(10)
	burst100 := CalculateResonanceBurst(100)
	if burst100 <= burst10 {
		t.Errorf("Burst for 100 (%f) should be > burst for 10 (%f)", burst100, burst10)
	}
}

func TestNaturalLog(t *testing.T) {
	tests := []struct {
		x        float64
		expected float64
		epsilon  float64
	}{
		{1.0, 0.0, 0.0001},
		{2.718281828, 1.0, 0.01},
		{10.0, 2.302585, 0.01},
	}

	for _, tc := range tests {
		got := naturalLog(tc.x)
		diff := got - tc.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tc.epsilon {
			t.Errorf("naturalLog(%f) = %f, want ~%f", tc.x, got, tc.expected)
		}
	}
}

func TestNewMaskedEventGated(t *testing.T) {
	creator := [32]byte{1}
	gate := newMockGate(creator, 150)

	event, err := NewMaskedEventGated(creator, "Test", time.Now(), 30*time.Minute, 10, gate)
	if err != nil {
		t.Fatalf("NewMaskedEventGated failed: %v", err)
	}
	if event == nil {
		t.Fatal("NewMaskedEventGated returned nil")
	}
}

func TestNewMaskedEventGated_InsufficientResonance(t *testing.T) {
	creator := [32]byte{1}
	gate := newMockGate(creator, 50)

	_, err := NewMaskedEventGated(creator, "Test", time.Now(), 30*time.Minute, 10, gate)
	if err != ErrMaskedEventInsufficientResonance {
		t.Errorf("Expected ErrMaskedEventInsufficientResonance, got %v", err)
	}
}

func TestMaskedEvent_GetParticipants(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	event.Join([32]byte{2})
	event.Join([32]byte{3})

	participants := event.GetParticipants()
	if len(participants) != 2 {
		t.Errorf("GetParticipants returned %d participants, want 2", len(participants))
	}
}

func TestMaskedEvent_IsFull(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 5)

	if event.IsFull() {
		t.Error("New event should not be full")
	}

	for i := 0; i < 5; i++ {
		event.Join([32]byte{byte(i + 10)})
	}

	if !event.IsFull() {
		t.Error("Event should be full after 5 joins")
	}
}

func TestMaskedEvent_UnlimitedParticipants(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 0)

	// Should never be full.
	if event.IsFull() {
		t.Error("Unlimited event should never be full")
	}

	// Add many participants.
	for i := 0; i < 50; i++ {
		_, err := event.Join([32]byte{byte(i)})
		if err != nil {
			t.Fatalf("Join %d failed: %v", i, err)
		}
	}

	if event.IsFull() {
		t.Error("Unlimited event should still not be full")
	}
}

func TestMaskedKeypair_GetPublicKey(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	keypair, _ := event.Join([32]byte{2})

	pubKey := keypair.GetPublicKey()
	if pubKey == [32]byte{} {
		t.Error("GetPublicKey should return non-zero key")
	}
}

func TestMaskedKeypair_GetPseudonym(t *testing.T) {
	creator := [32]byte{1}
	event, _ := NewMaskedEvent(creator, "Test", time.Now().Add(1*time.Hour), 30*time.Minute, 10)

	keypair, _ := event.Join([32]byte{2})

	pseudonym := keypair.GetPseudonym()
	if len(pseudonym) == 0 {
		t.Error("GetPseudonym should return non-empty string")
	}
}
