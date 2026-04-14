package waves

import (
	"bytes"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestCreateBeacon(t *testing.T) {
	content := []byte("Network health update")
	opts := NetworkHealthOptions([]byte(`{"peers":100}`))
	opts.Difficulty = 1 // Low difficulty for fast tests

	wave, err := CreateBeacon(content, opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	// Verify wave properties.
	if wave.WaveType != pb.WaveType(TypeBeacon) {
		t.Errorf("WaveType = %v, want %v", wave.WaveType, TypeBeacon)
	}

	if !bytes.Equal(wave.Content, content) {
		t.Error("Content mismatch")
	}

	// Verify null author.
	if len(wave.AuthorPubkey) != NullAuthorSize {
		t.Errorf("AuthorPubkey length = %d, want %d", len(wave.AuthorPubkey), NullAuthorSize)
	}
	for _, b := range wave.AuthorPubkey {
		if b != 0 {
			t.Error("AuthorPubkey should be all zeros")
			break
		}
	}

	// Verify no signature.
	if len(wave.Signature) != 0 {
		t.Error("Beacon should have no signature")
	}

	// Verify wave has ID.
	if len(wave.WaveId) == 0 {
		t.Error("WaveId is empty")
	}
}

func TestCreateBeaconEventAnnouncement(t *testing.T) {
	eventID := "event-123"
	eventName := "Test Event"
	start := time.Now().Unix()
	end := time.Now().Add(24 * time.Hour).Unix()

	opts := EventAnnouncementOptions(eventID, eventName, start, end)
	opts.Difficulty = 1

	wave, err := CreateBeacon([]byte("Event announcement"), opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	// Verify beacon type.
	beaconType := GetBeaconType(wave)
	if beaconType != BeaconEventAnnouncement {
		t.Errorf("BeaconType = %v, want %v", beaconType, BeaconEventAnnouncement)
	}

	// Verify event ID.
	if id := GetBeaconEventID(wave); id != eventID {
		t.Errorf("EventID = %q, want %q", id, eventID)
	}

	// Verify event name.
	if name := GetBeaconEventName(wave); name != eventName {
		t.Errorf("EventName = %q, want %q", name, eventName)
	}

	// Verify event times.
	gotStart, gotEnd := GetBeaconEventTimes(wave)
	if gotStart != start {
		t.Errorf("EventStart = %d, want %d", gotStart, start)
	}
	if gotEnd != end {
		t.Errorf("EventEnd = %d, want %d", gotEnd, end)
	}
}

func TestCreateBeaconEventSummary(t *testing.T) {
	eventID := "event-456"
	stats := []byte(`{"participants":50,"waves":200}`)

	opts := EventSummaryOptions(eventID, stats)
	opts.Difficulty = 1

	wave, err := CreateBeacon([]byte("Event summary"), opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	// Verify beacon type.
	beaconType := GetBeaconType(wave)
	if beaconType != BeaconEventSummary {
		t.Errorf("BeaconType = %v, want %v", beaconType, BeaconEventSummary)
	}

	// Verify event ID.
	if id := GetBeaconEventID(wave); id != eventID {
		t.Errorf("EventID = %q, want %q", id, eventID)
	}

	// Verify summary stats.
	gotStats := GetBeaconSummaryStats(wave)
	if !bytes.Equal(gotStats, stats) {
		t.Error("SummaryStats mismatch")
	}
}

func TestCreateBeaconNetworkHealth(t *testing.T) {
	metrics := []byte(`{"shroudCapacity":80,"avgLatency":150}`)

	opts := NetworkHealthOptions(metrics)
	opts.Difficulty = 1

	wave, err := CreateBeacon([]byte("Network health"), opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	// Verify beacon type.
	beaconType := GetBeaconType(wave)
	if beaconType != BeaconNetworkHealth {
		t.Errorf("BeaconType = %v, want %v", beaconType, BeaconNetworkHealth)
	}

	// Verify health metrics.
	gotMetrics := GetBeaconHealthMetrics(wave)
	if !bytes.Equal(gotMetrics, metrics) {
		t.Error("HealthMetrics mismatch")
	}
}

func TestCreateBeaconContentTooLarge(t *testing.T) {
	content := make([]byte, MaxContentSize+1)
	opts := DefaultBeaconOptions()

	_, err := CreateBeacon(content, opts)
	if err != ErrContentTooLarge {
		t.Errorf("Expected ErrContentTooLarge, got %v", err)
	}
}

func TestCreateBeaconInvalidTTL(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want error
	}{
		{"zero TTL", 0, ErrInvalidTTL},
		{"negative TTL", -time.Hour, ErrInvalidTTL},
		{"too long TTL", MaxTTL + time.Hour, ErrTTLTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := BeaconOptions{
				Type:       BeaconNetworkHealth,
				TTL:        tt.ttl,
				Difficulty: 1,
			}
			_, err := CreateBeacon([]byte("test"), opts)
			if err != tt.want {
				t.Errorf("Expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestCreateBeaconMissingEventID(t *testing.T) {
	opts := BeaconOptions{
		Type:       BeaconEventAnnouncement,
		TTL:        DefaultBeaconTTL,
		Difficulty: 1,
		EventID:    "", // Missing event ID
	}

	_, err := CreateBeacon([]byte("test"), opts)
	if err != ErrMissingBeaconEventID {
		t.Errorf("Expected ErrMissingBeaconEventID, got %v", err)
	}
}

func TestCreateBeaconInvalidType(t *testing.T) {
	opts := BeaconOptions{
		Type:       "invalid",
		TTL:        DefaultBeaconTTL,
		Difficulty: 1,
	}

	_, err := CreateBeacon([]byte("test"), opts)
	if err != ErrInvalidBeaconType {
		t.Errorf("Expected ErrInvalidBeaconType, got %v", err)
	}
}

func TestIsBeacon(t *testing.T) {
	opts := NetworkHealthOptions(nil)
	opts.Difficulty = 1

	wave, err := CreateBeacon([]byte("test"), opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	if !IsBeacon(wave) {
		t.Error("IsBeacon() = false, want true")
	}

	// Test with non-beacon wave.
	surfaceWave := &pb.Wave{
		WaveType:     pb.WaveType(TypeSurface),
		AuthorPubkey: make([]byte, 32),
	}
	surfaceWave.AuthorPubkey[0] = 1 // Non-null author

	if IsBeacon(surfaceWave) {
		t.Error("IsBeacon() = true for Surface wave, want false")
	}

	// Test with nil.
	if IsBeacon(nil) {
		t.Error("IsBeacon() = true for nil, want false")
	}

	// Test with non-null author.
	fakeBeacon := &pb.Wave{
		WaveType:     pb.WaveType(TypeBeacon),
		AuthorPubkey: make([]byte, NullAuthorSize),
	}
	fakeBeacon.AuthorPubkey[0] = 1

	if IsBeacon(fakeBeacon) {
		t.Error("IsBeacon() = true for beacon with non-null author")
	}
}

func TestValidateBeacon(t *testing.T) {
	opts := NetworkHealthOptions(nil)
	opts.Difficulty = 1

	wave, err := CreateBeacon([]byte("test"), opts)
	if err != nil {
		t.Fatalf("CreateBeacon() error = %v", err)
	}

	// Validation should pass.
	if err := ValidateBeacon(wave, 1); err != nil {
		t.Errorf("ValidateBeacon() error = %v", err)
	}
}

func TestValidateBeaconNil(t *testing.T) {
	err := ValidateBeacon(nil, 1)
	if err == nil {
		t.Error("Expected error for nil wave")
	}
}

func TestValidateBeaconWrongType(t *testing.T) {
	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeSurface),
		AuthorPubkey: make([]byte, NullAuthorSize),
	}

	err := ValidateBeacon(wave, 1)
	if err == nil {
		t.Error("Expected error for wrong wave type")
	}
}

func TestInt64Conversion(t *testing.T) {
	tests := []int64{
		0,
		1,
		-1,
		12345678901234,
		-12345678901234,
		1<<62 - 1,
	}

	for _, val := range tests {
		slice := int64ToSlice(val)
		if len(slice) != 8 {
			t.Errorf("int64ToSlice(%d) length = %d, want 8", val, len(slice))
		}

		result := sliceToInt64(slice)
		if result != val {
			t.Errorf("Round trip: %d -> %v -> %d", val, slice, result)
		}
	}
}

func TestSliceToInt64InvalidLength(t *testing.T) {
	result := sliceToInt64([]byte{1, 2, 3}) // Wrong length
	if result != 0 {
		t.Errorf("sliceToInt64() = %d, want 0 for invalid length", result)
	}
}

func TestGetBeaconMetadataNil(t *testing.T) {
	// Test all getters with nil wave.
	if GetBeaconType(nil) != "" {
		t.Error("GetBeaconType(nil) should return empty")
	}
	if GetBeaconEventID(nil) != "" {
		t.Error("GetBeaconEventID(nil) should return empty")
	}
	if GetBeaconEventName(nil) != "" {
		t.Error("GetBeaconEventName(nil) should return empty")
	}
	start, end := GetBeaconEventTimes(nil)
	if start != 0 || end != 0 {
		t.Error("GetBeaconEventTimes(nil) should return 0, 0")
	}
	if GetBeaconSummaryStats(nil) != nil {
		t.Error("GetBeaconSummaryStats(nil) should return nil")
	}
	if GetBeaconHealthMetrics(nil) != nil {
		t.Error("GetBeaconHealthMetrics(nil) should return nil")
	}

	// Test with wave without metadata.
	emptyWave := &pb.Wave{}
	if GetBeaconType(emptyWave) != "" {
		t.Error("GetBeaconType() should return empty for wave without metadata")
	}
}

func TestDefaultBeaconOptions(t *testing.T) {
	opts := DefaultBeaconOptions()

	if opts.Type != BeaconNetworkHealth {
		t.Errorf("Default Type = %v, want %v", opts.Type, BeaconNetworkHealth)
	}
	if opts.TTL != DefaultBeaconTTL {
		t.Errorf("Default TTL = %v, want %v", opts.TTL, DefaultBeaconTTL)
	}
	if opts.Difficulty != BeaconDifficulty {
		t.Errorf("Default Difficulty = %d, want %d", opts.Difficulty, BeaconDifficulty)
	}
}
