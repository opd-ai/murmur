// Package waves provides Wave creation, signing, and validation.
// This file implements Beacon Wave — system-generated high-visibility broadcasts
// used for event announcements, summaries, and network health updates.
// Per WAVES.md, Beacon Waves (type 0x08) have null author, no signature,
// and higher PoW difficulty (24 bits).
package waves

import (
	"errors"
	"time"

	"github.com/opd-ai/murmur/pkg/content/pow"
	pb "github.com/opd-ai/murmur/proto"
)

// Beacon Wave constants.
const (
	// BeaconDifficulty is the elevated PoW difficulty for Beacon Waves.
	// Per WAVES.md, 24 leading zero bits to prevent spam.
	BeaconDifficulty = 24

	// BeaconTypeKey is the metadata key for beacon type.
	BeaconTypeKey = "beacon_type"

	// BeaconTypeEventAnnounce indicates an event announcement beacon.
	BeaconTypeEventAnnounce = "event_announce"

	// BeaconTypeEventSummary indicates an event summary beacon.
	BeaconTypeEventSummary = "event_summary"

	// BeaconTypeNetworkHealth indicates a network health beacon.
	BeaconTypeNetworkHealth = "network_health"

	// BeaconEventIDKey is the metadata key for event identifier.
	BeaconEventIDKey = "event_id"

	// BeaconEventNameKey is the metadata key for event name.
	BeaconEventNameKey = "event_name"

	// BeaconEventStartKey is the metadata key for event start time.
	BeaconEventStartKey = "event_start"

	// BeaconEventEndKey is the metadata key for event end time.
	BeaconEventEndKey = "event_end"

	// BeaconSummaryStatsKey is the metadata key for summary statistics.
	BeaconSummaryStatsKey = "summary_stats"

	// BeaconHealthMetricsKey is the metadata key for health metrics.
	BeaconHealthMetricsKey = "health_metrics"

	// NullAuthorSize is the size of the null author public key (32 zero bytes).
	NullAuthorSize = 32

	// DefaultBeaconTTL is the default TTL for Beacon Waves.
	DefaultBeaconTTL = 24 * time.Hour
)

// Beacon Wave errors.
var (
	ErrInvalidBeaconType    = errors.New("invalid beacon type")
	ErrMissingBeaconEventID = errors.New("missing event ID for event beacon")
)

// BeaconType represents the type of Beacon Wave.
type BeaconType string

const (
	BeaconEventAnnouncement BeaconType = BeaconTypeEventAnnounce
	BeaconEventSummary      BeaconType = BeaconTypeEventSummary
	BeaconNetworkHealth     BeaconType = BeaconTypeNetworkHealth
)

// BeaconOptions configures Beacon Wave creation.
type BeaconOptions struct {
	// Type is the beacon type.
	Type BeaconType

	// TTL is the time-to-live for the beacon.
	TTL time.Duration

	// Difficulty is the PoW difficulty (default: 24).
	Difficulty uint8

	// EventID is the event identifier (for event beacons).
	EventID string

	// EventName is the human-readable event name.
	EventName string

	// EventStart is the event start time (Unix timestamp).
	EventStart int64

	// EventEnd is the event end time (Unix timestamp).
	EventEnd int64

	// SummaryStats is JSON-encoded summary statistics.
	SummaryStats []byte

	// HealthMetrics is JSON-encoded health metrics.
	HealthMetrics []byte
}

// DefaultBeaconOptions returns default options for Beacon Wave creation.
func DefaultBeaconOptions() BeaconOptions {
	return BeaconOptions{
		Type:       BeaconNetworkHealth,
		TTL:        DefaultBeaconTTL,
		Difficulty: BeaconDifficulty,
	}
}

// EventAnnouncementOptions returns options for an event announcement beacon.
func EventAnnouncementOptions(eventID, eventName string, start, end int64) BeaconOptions {
	return BeaconOptions{
		Type:       BeaconEventAnnouncement,
		TTL:        DefaultBeaconTTL,
		Difficulty: BeaconDifficulty,
		EventID:    eventID,
		EventName:  eventName,
		EventStart: start,
		EventEnd:   end,
	}
}

// EventSummaryOptions returns options for an event summary beacon.
func EventSummaryOptions(eventID string, stats []byte) BeaconOptions {
	return BeaconOptions{
		Type:         BeaconEventSummary,
		TTL:          DefaultBeaconTTL,
		Difficulty:   BeaconDifficulty,
		EventID:      eventID,
		SummaryStats: stats,
	}
}

// NetworkHealthOptions returns options for a network health beacon.
func NetworkHealthOptions(metrics []byte) BeaconOptions {
	return BeaconOptions{
		Type:          BeaconNetworkHealth,
		TTL:           DefaultBeaconTTL,
		Difficulty:    BeaconDifficulty,
		HealthMetrics: metrics,
	}
}

// CreateBeacon creates a Beacon Wave.
// Beacon Waves have null author (32 zero bytes), no signature,
// and are verified by content structure and PoW only.
func CreateBeacon(content []byte, opts BeaconOptions) (*pb.Wave, error) {
	if len(content) > MaxContentSize {
		return nil, ErrContentTooLarge
	}
	if opts.TTL <= 0 {
		return nil, ErrInvalidTTL
	}
	if opts.TTL > MaxTTL {
		return nil, ErrTTLTooLong
	}

	// Validate beacon type.
	if err := validateBeaconOptions(opts); err != nil {
		return nil, err
	}

	// Build the Beacon Wave.
	wave := buildBeaconWave(content, opts)

	// Compute PoW (no signature for Beacon Waves).
	if err := computeBeaconPoW(wave, opts.Difficulty); err != nil {
		return nil, err
	}

	return wave, nil
}

// validateBeaconOptions validates beacon-specific options.
func validateBeaconOptions(opts BeaconOptions) error {
	switch opts.Type {
	case BeaconEventAnnouncement, BeaconEventSummary:
		if opts.EventID == "" {
			return ErrMissingBeaconEventID
		}
	case BeaconNetworkHealth:
		// No special requirements
	default:
		return ErrInvalidBeaconType
	}
	return nil
}

// buildBeaconWave constructs the Beacon Wave structure.
func buildBeaconWave(content []byte, opts BeaconOptions) *pb.Wave {
	now := time.Now()

	// Null author (32 zero bytes).
	nullAuthor := make([]byte, NullAuthorSize)

	// Build metadata based on beacon type.
	metadata := buildBeaconMetadata(opts)

	wave := &pb.Wave{
		WaveType:     pb.WaveType(TypeBeacon),
		Content:      content,
		AuthorPubkey: nullAuthor,
		Signature:    nil, // No signature for Beacon Waves
		CreatedAt:    now.Unix(),
		TtlSeconds:   int64(opts.TTL.Seconds()),
		HopCount:     0,
		Metadata:     metadata,
	}

	wave.WaveId = computeWaveID(wave)
	return wave
}

// buildBeaconMetadata builds metadata for the beacon type.
func buildBeaconMetadata(opts BeaconOptions) map[string][]byte {
	metadata := map[string][]byte{
		BeaconTypeKey: []byte(opts.Type),
	}

	switch opts.Type {
	case BeaconEventAnnouncement:
		metadata[BeaconEventIDKey] = []byte(opts.EventID)
		if opts.EventName != "" {
			metadata[BeaconEventNameKey] = []byte(opts.EventName)
		}
		if opts.EventStart > 0 {
			metadata[BeaconEventStartKey] = int64ToSlice(opts.EventStart)
		}
		if opts.EventEnd > 0 {
			metadata[BeaconEventEndKey] = int64ToSlice(opts.EventEnd)
		}

	case BeaconEventSummary:
		metadata[BeaconEventIDKey] = []byte(opts.EventID)
		if len(opts.SummaryStats) > 0 {
			metadata[BeaconSummaryStatsKey] = opts.SummaryStats
		}

	case BeaconNetworkHealth:
		if len(opts.HealthMetrics) > 0 {
			metadata[BeaconHealthMetricsKey] = opts.HealthMetrics
		}
	}

	return metadata
}

// int64ToSlice converts an int64 to a byte slice.
func int64ToSlice(v int64) []byte {
	b := make([]byte, 8)
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}

// sliceToInt64 converts a byte slice to int64.
func sliceToInt64(b []byte) int64 {
	if len(b) != 8 {
		return 0
	}
	return int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
}

// computeBeaconPoW computes PoW for a Beacon Wave (no signature).
func computeBeaconPoW(wave *pb.Wave, difficulty uint8) error {
	powInput := beaconPowData(wave)
	work, err := pow.Compute(powInput, difficulty)
	if err != nil {
		return err
	}
	wave.PowNonce = work.Nonce
	return nil
}

// beaconPowData returns the data used for Beacon PoW computation.
func beaconPowData(wave *pb.Wave) []byte {
	// For Beacon Waves, PoW is over WaveID only (no signature).
	return wave.WaveId
}

// IsBeacon checks if a Wave is a Beacon Wave.
func IsBeacon(wave *pb.Wave) bool {
	if wave == nil {
		return false
	}
	if wave.WaveType != pb.WaveType(TypeBeacon) {
		return false
	}
	// Verify null author.
	if len(wave.AuthorPubkey) != NullAuthorSize {
		return false
	}
	for _, b := range wave.AuthorPubkey {
		if b != 0 {
			return false
		}
	}
	return true
}

// GetBeaconType returns the beacon type from metadata.
func GetBeaconType(wave *pb.Wave) BeaconType {
	if wave == nil || wave.Metadata == nil {
		return ""
	}
	return BeaconType(wave.Metadata[BeaconTypeKey])
}

// GetBeaconEventID returns the event ID from beacon metadata.
func GetBeaconEventID(wave *pb.Wave) string {
	if wave == nil || wave.Metadata == nil {
		return ""
	}
	return string(wave.Metadata[BeaconEventIDKey])
}

// GetBeaconEventName returns the event name from beacon metadata.
func GetBeaconEventName(wave *pb.Wave) string {
	if wave == nil || wave.Metadata == nil {
		return ""
	}
	return string(wave.Metadata[BeaconEventNameKey])
}

// GetBeaconEventTimes returns event start and end times.
func GetBeaconEventTimes(wave *pb.Wave) (start, end int64) {
	if wave == nil || wave.Metadata == nil {
		return 0, 0
	}
	start = sliceToInt64(wave.Metadata[BeaconEventStartKey])
	end = sliceToInt64(wave.Metadata[BeaconEventEndKey])
	return start, end
}

// GetBeaconSummaryStats returns summary statistics from beacon metadata.
func GetBeaconSummaryStats(wave *pb.Wave) []byte {
	if wave == nil || wave.Metadata == nil {
		return nil
	}
	return wave.Metadata[BeaconSummaryStatsKey]
}

// GetBeaconHealthMetrics returns health metrics from beacon metadata.
func GetBeaconHealthMetrics(wave *pb.Wave) []byte {
	if wave == nil || wave.Metadata == nil {
		return nil
	}
	return wave.Metadata[BeaconHealthMetricsKey]
}

// ValidateBeacon validates a Beacon Wave.
// Beacon Waves are verified by content structure and PoW only, not signature.
func ValidateBeacon(wave *pb.Wave, difficulty uint8) error {
	if wave == nil {
		return errors.New("wave is nil")
	}
	if !IsBeacon(wave) {
		return errors.New("not a Beacon Wave")
	}

	// Check content size.
	if len(wave.Content) > MaxContentSize {
		return ErrContentTooLarge
	}

	// Check expiration.
	if IsExpired(wave) {
		return ErrExpired
	}

	// Verify beacon type is present.
	beaconType := GetBeaconType(wave)
	if beaconType == "" {
		return ErrInvalidBeaconType
	}

	// Verify PoW (using beacon-specific PoW data).
	powInput := beaconPowData(wave)
	if !pow.Verify(powInput, wave.PowNonce, difficulty) {
		return ErrInvalidPoW
	}

	return nil
}
