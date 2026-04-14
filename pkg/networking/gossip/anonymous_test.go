package gossip

import (
	"context"
	"testing"
	"time"
)

func TestQuantizeTimestamp(t *testing.T) {
	// Test that timestamps are quantized to 5-minute buckets
	testCases := []struct {
		input    time.Time
		expected time.Time
	}{
		{
			input:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 12, 2, 30, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 12, 4, 59, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 12, 7, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 5, 0, 0, time.UTC),
		},
		{
			input:    time.Date(2024, 1, 1, 12, 10, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 1, 12, 10, 0, 0, time.UTC),
		},
	}

	for _, tc := range testCases {
		result := QuantizeTimestamp(tc.input)
		if !result.Equal(tc.expected) {
			t.Errorf("QuantizeTimestamp(%v) = %v, expected %v",
				tc.input, result, tc.expected)
		}
	}
}

func TestAnonymousTopicConstants(t *testing.T) {
	// Verify topic strings match spec
	if TopicAnonymousWaves != "/murmur/anonymous/waves/1.0" {
		t.Errorf("TopicAnonymousWaves = %s, expected /murmur/anonymous/waves/1.0",
			TopicAnonymousWaves)
	}
	if TopicAnonymousMechanics != "/murmur/anonymous/mechanics/1.0" {
		t.Errorf("TopicAnonymousMechanics = %s, expected /murmur/anonymous/mechanics/1.0",
			TopicAnonymousMechanics)
	}
	if TopicAnonymousBeacons != "/murmur/anonymous/beacons/1.0" {
		t.Errorf("TopicAnonymousBeacons = %s, expected /murmur/anonymous/beacons/1.0",
			TopicAnonymousBeacons)
	}
}

func TestTimestampQuantum(t *testing.T) {
	// Verify quantum is 5 minutes
	expected := 5 * time.Minute
	if TimestampQuantum != expected {
		t.Errorf("TimestampQuantum = %v, expected %v", TimestampQuantum, expected)
	}
}

func TestAnonymousPoWDifficulty(t *testing.T) {
	// Per WAVE_PROPAGATION.md, anonymous PoW should be higher than surface (20)
	if AnonymousPoWDifficulty < 20 {
		t.Errorf("AnonymousPoWDifficulty = %d, should be >= 20", AnonymousPoWDifficulty)
	}
	if BeaconPoWDifficulty <= AnonymousPoWDifficulty {
		t.Errorf("BeaconPoWDifficulty (%d) should be > AnonymousPoWDifficulty (%d)",
			BeaconPoWDifficulty, AnonymousPoWDifficulty)
	}
}

func TestEventTopic(t *testing.T) {
	eventID := "abc123"
	topic := EventTopic(eventID)
	expected := "/murmur/event/abc123/1.0"
	if topic != expected {
		t.Errorf("EventTopic(%s) = %s, expected %s", eventID, topic, expected)
	}
}

func TestCouncilTopic(t *testing.T) {
	councilID := "council-xyz"
	topic := CouncilTopic(councilID)
	expected := "/murmur/council/council-xyz/1.0"
	if topic != expected {
		t.Errorf("CouncilTopic(%s) = %s, expected %s", councilID, topic, expected)
	}
}

func TestNewAnonymousTopicHandlers(t *testing.T) {
	tracker := NewPeerScoreTracker()
	handlers := NewAnonymousTopicHandlers(tracker)

	if handlers == nil {
		t.Fatal("expected non-nil handlers")
	}
	if handlers.dedup == nil {
		t.Error("dedup should be initialized")
	}
	if handlers.scoreTracker != tracker {
		t.Error("scoreTracker should be set")
	}
}

func TestAnonymousTopicHandlers_SetHandlers(t *testing.T) {
	handlers := NewAnonymousTopicHandlers(nil)

	// Test setting wave handler
	handlers.SetWaveHandler(&mockAnonymousWaveHandler{})
	if handlers.waveHandler == nil {
		t.Error("wave handler should be set")
	}

	// Test setting mechanics handler
	handlers.SetMechanicsHandler(&mockAnonymousMechanicsHandler{})
	if handlers.mechanicsHandler == nil {
		t.Error("mechanics handler should be set")
	}

	// Test setting beacon handler
	handlers.SetBeaconHandler(&mockBeaconWaveHandler{})
	if handlers.beaconHandler == nil {
		t.Error("beacon handler should be set")
	}
}

func TestAnonymousTopicHandlers_CreateAnonymousTopicHandler(t *testing.T) {
	handlers := NewAnonymousTopicHandlers(nil)
	handler := handlers.CreateAnonymousTopicHandler(TopicAnonymousWaves)
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestVerifyAnonymousPoW(t *testing.T) {
	// Test difficulty requirements
	if VerifyAnonymousPoW([]byte{}, []byte{}, AnonymousPoWDifficulty-1) {
		t.Error("should reject difficulty below AnonymousPoWDifficulty")
	}
	if !VerifyAnonymousPoW([]byte{}, []byte{}, AnonymousPoWDifficulty) {
		t.Error("should accept difficulty at AnonymousPoWDifficulty")
	}
	if !VerifyAnonymousPoW([]byte{}, []byte{}, AnonymousPoWDifficulty+1) {
		t.Error("should accept difficulty above AnonymousPoWDifficulty")
	}
}

func TestVerifyBeaconPoW(t *testing.T) {
	// Test Beacon difficulty requirements
	if VerifyBeaconPoW([]byte{}, []byte{}, BeaconPoWDifficulty-1) {
		t.Error("should reject difficulty below BeaconPoWDifficulty")
	}
	if !VerifyBeaconPoW([]byte{}, []byte{}, BeaconPoWDifficulty) {
		t.Error("should accept difficulty at BeaconPoWDifficulty")
	}
	if !VerifyBeaconPoW([]byte{}, []byte{}, BeaconPoWDifficulty+1) {
		t.Error("should accept difficulty above BeaconPoWDifficulty")
	}
}

// Mock implementations for testing

type mockAnonymousWaveHandler struct{}

func (m *mockAnonymousWaveHandler) HandleSpecterWave(ctx context.Context, env *Envelope) error {
	return nil
}

func (m *mockAnonymousWaveHandler) HandleMaskedWave(ctx context.Context, env *Envelope) error {
	return nil
}

type mockAnonymousMechanicsHandler struct{}

func (m *mockAnonymousMechanicsHandler) HandlePhantomGift(ctx context.Context, env *Envelope) error {
	return nil
}

func (m *mockAnonymousMechanicsHandler) HandleSpecterMark(ctx context.Context, env *Envelope) error {
	return nil
}

func (m *mockAnonymousMechanicsHandler) HandleMiniGameEvent(ctx context.Context, env *Envelope) error {
	return nil
}

func (m *mockAnonymousMechanicsHandler) HandleCouncilMessage(ctx context.Context, env *Envelope) error {
	return nil
}

type mockBeaconWaveHandler struct{}

func (m *mockBeaconWaveHandler) HandleBeaconWave(ctx context.Context, env *Envelope) error {
	return nil
}
