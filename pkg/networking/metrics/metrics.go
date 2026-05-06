// Package metrics provides Prometheus metrics integration for MURMUR.
// Per AUDIT.md MEDIUM finding, operators need visibility into connection counts,
// message rates, and Resonance distribution.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ConnectionsGauge tracks current peer connections by type.
	// Labels: type (identity, gossip, random)
	ConnectionsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "murmur_connections",
			Help: "Current number of peer connections by type",
		},
		[]string{"type"},
	)

	// WavesReceivedTotal counts total Waves received.
	WavesReceivedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "murmur_waves_received_total",
			Help: "Total number of Waves received from the network",
		},
	)

	// WavesPublishedTotal counts total Waves published.
	WavesPublishedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "murmur_waves_published_total",
			Help: "Total number of Waves published to the network",
		},
	)

	// ResonanceScoreGauge tracks current Resonance scores.
	// Labels: layer (surface, specter)
	ResonanceScoreGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "murmur_resonance_score",
			Help: "Current Resonance score by layer",
		},
		[]string{"layer"},
	)

	// GossipMessagesReceivedTotal counts total GossipSub messages by topic.
	// Labels: topic
	GossipMessagesReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_gossip_messages_received_total",
			Help: "Total number of GossipSub messages received by topic",
		},
		[]string{"topic"},
	)

	// GossipMessagesPublishedTotal counts total GossipSub messages by topic.
	// Labels: topic
	GossipMessagesPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_gossip_messages_published_total",
			Help: "Total number of GossipSub messages published by topic",
		},
		[]string{"topic"},
	)

	// AnonymousEventsReceivedTotal counts anonymous mechanic events received.
	// Labels: type (gift, mark, puzzle, hunt, etc.)
	AnonymousEventsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_anonymous_events_received_total",
			Help: "Total number of anonymous mechanic events received by type",
		},
		[]string{"type"},
	)

	// AnonymousEventsPublishedTotal counts anonymous mechanic events published.
	// Labels: type (gift, mark, puzzle, hunt, etc.)
	AnonymousEventsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_anonymous_events_published_total",
			Help: "Total number of anonymous mechanic events published by type",
		},
		[]string{"type"},
	)

	// ShroudCircuitsActiveGauge tracks active Shroud circuits.
	ShroudCircuitsActiveGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "murmur_shroud_circuits_active",
			Help: "Current number of active Shroud onion circuits",
		},
	)

	// ShroudCircuitBuildDurationSeconds tracks circuit construction latency.
	// Per AUDIT.md M4, this histogram instruments circuit build time.
	ShroudCircuitBuildDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "murmur_shroud_circuit_build_duration_seconds",
			Help:    "Time taken to build a Shroud circuit (key exchange only)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
	)

	// DHTBootstrapAttemptsTotal counts DHT bootstrap attempts.
	DHTBootstrapAttemptsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "murmur_dht_bootstrap_attempts_total",
			Help: "Total number of DHT bootstrap attempts",
		},
	)

	// DHTBootstrapSuccessesTotal counts successful DHT bootstraps.
	DHTBootstrapSuccessesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "murmur_dht_bootstrap_successes_total",
			Help: "Total number of successful DHT bootstraps",
		},
	)

	// MemoryAllocatedBytesGauge tracks current memory allocation.
	MemoryAllocatedBytesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "murmur_memory_allocated_bytes",
			Help: "Current memory allocated in bytes",
		},
	)

	// WaveCacheEntriesGauge tracks number of Waves in memory cache.
	WaveCacheEntriesGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "murmur_wave_cache_entries",
			Help: "Current number of Waves in memory cache",
		},
	)

	// PeerScoreGauge tracks peer scores.
	// Labels: peer_id (truncated to first 8 chars for cardinality control)
	PeerScoreGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "murmur_peer_score",
			Help: "Current peer score by peer ID",
		},
		[]string{"peer_id"},
	)

	// RateLimitDropsTotal counts messages dropped due to rate limiting.
	// Labels: peer_id (truncated)
	RateLimitDropsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_rate_limit_drops_total",
			Help: "Total number of messages dropped due to rate limiting by peer",
		},
		[]string{"peer_id"},
	)

	// DeduplicationDropsTotal counts messages dropped due to deduplication.
	DeduplicationDropsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "murmur_deduplication_drops_total",
			Help: "Total number of messages dropped due to deduplication (already seen)",
		},
	)

	// EventBusDropsTotal counts events dropped by the event bus.
	// Events are dropped when the inbound buffer or subscriber channels are full.
	// Labels: reason (inbound_full, subscriber_full)
	EventBusDropsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_eventbus_drops_total",
			Help: "Total number of events dropped by the event bus due to full buffers",
		},
		[]string{"reason"},
	)

	// EventBusEventsTotal counts total events dispatched by type.
	// Labels: event_type (WaveReceived, PeerConnected, etc.)
	EventBusEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "murmur_eventbus_events_total",
			Help: "Total number of events dispatched by the event bus by type",
		},
		[]string{"event_type"},
	)
)
