package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestMetricsInitialization verifies all metrics are registered and accessible.
func TestMetricsInitialization(t *testing.T) {
	// Test that connection gauge can be set
	ConnectionsGauge.WithLabelValues("identity").Set(5)
	ConnectionsGauge.WithLabelValues("gossip").Set(10)
	ConnectionsGauge.WithLabelValues("random").Set(2)

	// Test that counters can be incremented
	WavesReceivedTotal.Inc()
	WavesPublishedTotal.Inc()

	// Test ResonanceScoreGauge
	ResonanceScoreGauge.WithLabelValues("surface").Set(42)
	ResonanceScoreGauge.WithLabelValues("specter").Set(75)

	// Test GossipSub metrics
	GossipMessagesReceivedTotal.WithLabelValues("/murmur/waves/1").Inc()
	GossipMessagesPublishedTotal.WithLabelValues("/murmur/waves/1").Inc()

	// Test anonymous metrics
	AnonymousEventsReceivedTotal.WithLabelValues("gift").Inc()
	AnonymousEventsPublishedTotal.WithLabelValues("gift").Inc()

	// Test Shroud metrics
	ShroudCircuitsActiveGauge.Set(3)

	// Test DHT metrics
	DHTBootstrapAttemptsTotal.Inc()
	DHTBootstrapSuccessesTotal.Inc()

	// Test memory metrics
	MemoryAllocatedBytesGauge.Set(256 * 1024 * 1024) // 256 MiB

	// Test cache metrics
	WaveCacheEntriesGauge.Set(1000)

	// Test peer score metrics
	PeerScoreGauge.WithLabelValues("12D3KooW").Set(10.5)

	// Test rate limiting metrics
	RateLimitDropsTotal.WithLabelValues("12D3KooW").Inc()

	// Test deduplication metrics
	DeduplicationDropsTotal.Inc()

	// Verify Waves received counter has value >= 1 (may be higher if running with -count > 1)
	if count := testutil.ToFloat64(WavesReceivedTotal); count < 1 {
		t.Errorf("WavesReceivedTotal = %f, want >= 1", count)
	}

	// Verify deduplication drops counter has value >= 1 (may be higher if running with -count > 1)
	if count := testutil.ToFloat64(DeduplicationDropsTotal); count < 1 {
		t.Errorf("DeduplicationDropsTotal = %f, want >= 1", count)
	}
}

// TestMetricsLabels verifies that labeled metrics work correctly.
func TestMetricsLabels(t *testing.T) {
	// Set different values for different connection types
	ConnectionsGauge.WithLabelValues("identity").Set(3)
	ConnectionsGauge.WithLabelValues("gossip").Set(7)

	// Set different anonymous event types
	AnonymousEventsReceivedTotal.WithLabelValues("gift").Add(5)
	AnonymousEventsReceivedTotal.WithLabelValues("mark").Add(3)
	AnonymousEventsReceivedTotal.WithLabelValues("puzzle").Add(2)

	// No assertions - if labels are invalid, promauto will panic
	// This test ensures label names and values are valid
}
