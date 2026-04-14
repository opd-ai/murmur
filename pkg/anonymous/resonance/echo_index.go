// Package resonance provides local reputation computation and rank thresholds.
// This file implements Echo Index computation — a per-cluster metric measuring
// the ideological diversity of amplification within a cluster.
// Per RESONANCE_SYSTEM.md, Echo Index = intra_cluster_amplifications / total_amplifications.
package resonance

import (
	"math"
	"sync"
	"time"
)

// EchoIndexThresholds define the display categories.
const (
	EchoIndexEchoChamber = 0.7 // Above this = echo chamber (warm colors)
	EchoIndexOutwardOpen = 0.4 // Below this = outward-looking (cool colors)
)

// EchoCategory represents the diversity category of a cluster.
type EchoCategory int

const (
	EchoCategoryNeutral EchoCategory = iota // 0.4 - 0.7
	EchoCategoryInsular                     // > 0.7 (echo chamber)
	EchoCategoryOpen                        // < 0.4 (outward-looking)
)

// String returns a human-readable category name.
func (c EchoCategory) String() string {
	switch c {
	case EchoCategoryInsular:
		return "Insular"
	case EchoCategoryOpen:
		return "Open"
	default:
		return "Neutral"
	}
}

// AmplificationRecord represents a single amplification event.
type AmplificationRecord struct {
	AmplifierID      string    // ID of the node that amplified
	AuthorID         string    // ID of the original Wave author
	WaveID           string    // ID of the amplified Wave
	Timestamp        time.Time // When the amplification occurred
	AmplifierCluster string    // Cluster ID of the amplifier at time of event
	AuthorCluster    string    // Cluster ID of the author at time of event
}

// ClusterEchoIndex represents the Echo Index for a single cluster.
type ClusterEchoIndex struct {
	ClusterID        string
	EchoIndex        float64 // 0.0 to 1.0
	Category         EchoCategory
	IntraClusterAmps int // Amplifications of content from same cluster
	ExtraClusterAmps int // Amplifications of content from other clusters
	TotalAmps        int // Total amplifications from this cluster
	LastComputed     time.Time
	SampleWindowDays int // Window size (typically 30)
}

// CategoryFromEchoIndex returns the category for a given Echo Index value.
func CategoryFromEchoIndex(index float64) EchoCategory {
	switch {
	case index >= EchoIndexEchoChamber:
		return EchoCategoryInsular
	case index <= EchoIndexOutwardOpen:
		return EchoCategoryOpen
	default:
		return EchoCategoryNeutral
	}
}

// EchoIndexComputer computes Echo Index for clusters in a topology.
type EchoIndexComputer struct {
	mu sync.RWMutex

	// Amplification records (trailing window).
	records []AmplificationRecord

	// Configuration.
	windowDays int

	// Computed indices.
	clusterIndices map[string]*ClusterEchoIndex

	// Last computation time.
	lastComputed time.Time
}

// EchoIndexConfig configures the Echo Index computer.
type EchoIndexConfig struct {
	WindowDays int // Trailing window for amplification data (default: 30)
}

// DefaultEchoIndexConfig returns the standard configuration.
func DefaultEchoIndexConfig() EchoIndexConfig {
	return EchoIndexConfig{
		WindowDays: 30,
	}
}

// NewEchoIndexComputer creates a new Echo Index computer.
func NewEchoIndexComputer() *EchoIndexComputer {
	return &EchoIndexComputer{
		windowDays:     30,
		records:        make([]AmplificationRecord, 0),
		clusterIndices: make(map[string]*ClusterEchoIndex),
	}
}

// NewEchoIndexComputerWithConfig creates a computer with custom configuration.
func NewEchoIndexComputerWithConfig(cfg EchoIndexConfig) *EchoIndexComputer {
	windowDays := cfg.WindowDays
	if windowDays <= 0 {
		windowDays = 30
	}
	return &EchoIndexComputer{
		windowDays:     windowDays,
		records:        make([]AmplificationRecord, 0),
		clusterIndices: make(map[string]*ClusterEchoIndex),
	}
}

// RecordAmplification records an amplification event.
func (c *EchoIndexComputer) RecordAmplification(record AmplificationRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	c.records = append(c.records, record)
}

// RecordAmplificationSimple records an amplification with minimal data.
func (c *EchoIndexComputer) RecordAmplificationSimple(
	amplifierID, authorID, waveID string,
	amplifierCluster, authorCluster string,
) {
	c.RecordAmplification(AmplificationRecord{
		AmplifierID:      amplifierID,
		AuthorID:         authorID,
		WaveID:           waveID,
		Timestamp:        time.Now(),
		AmplifierCluster: amplifierCluster,
		AuthorCluster:    authorCluster,
	})
}

// Compute recomputes the Echo Index for all clusters.
// Per spec, this should be called every 24 hours.
func (c *EchoIndexComputer) Compute() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Prune old records first.
	c.pruneOldRecords()

	// Clear previous indices.
	c.clusterIndices = make(map[string]*ClusterEchoIndex)

	// Group amplifications by amplifier cluster.
	clusterAmps := make(map[string]struct {
		intra int
		extra int
	})

	for _, record := range c.records {
		if record.AmplifierCluster == "" {
			continue
		}

		data := clusterAmps[record.AmplifierCluster]
		if record.AmplifierCluster == record.AuthorCluster {
			data.intra++
		} else {
			data.extra++
		}
		clusterAmps[record.AmplifierCluster] = data
	}

	// Compute Echo Index for each cluster.
	now := time.Now()
	for clusterID, data := range clusterAmps {
		total := data.intra + data.extra
		var echoIndex float64
		if total > 0 {
			echoIndex = float64(data.intra) / float64(total)
		}

		c.clusterIndices[clusterID] = &ClusterEchoIndex{
			ClusterID:        clusterID,
			EchoIndex:        echoIndex,
			Category:         CategoryFromEchoIndex(echoIndex),
			IntraClusterAmps: data.intra,
			ExtraClusterAmps: data.extra,
			TotalAmps:        total,
			LastComputed:     now,
			SampleWindowDays: c.windowDays,
		}
	}

	c.lastComputed = now
}

// pruneOldRecords removes records outside the trailing window.
func (c *EchoIndexComputer) pruneOldRecords() {
	cutoff := time.Now().Add(-time.Duration(c.windowDays) * 24 * time.Hour)

	// Find the first record within the window.
	keepFrom := 0
	for i, record := range c.records {
		if record.Timestamp.After(cutoff) {
			keepFrom = i
			break
		}
		keepFrom = len(c.records) // All records are old
	}

	if keepFrom > 0 && keepFrom < len(c.records) {
		c.records = c.records[keepFrom:]
	} else if keepFrom >= len(c.records) {
		c.records = c.records[:0]
	}
}

// GetClusterIndex returns the Echo Index for a specific cluster.
func (c *EchoIndexComputer) GetClusterIndex(clusterID string) *ClusterEchoIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if idx, ok := c.clusterIndices[clusterID]; ok {
		return idx
	}
	return nil
}

// GetAllIndices returns Echo Index for all tracked clusters.
func (c *EchoIndexComputer) GetAllIndices() map[string]*ClusterEchoIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*ClusterEchoIndex, len(c.clusterIndices))
	for k, v := range c.clusterIndices {
		result[k] = v
	}
	return result
}

// GetInsularClusters returns clusters with Echo Index above the echo chamber threshold.
func (c *EchoIndexComputer) GetInsularClusters() []*ClusterEchoIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var insular []*ClusterEchoIndex
	for _, idx := range c.clusterIndices {
		if idx.Category == EchoCategoryInsular {
			insular = append(insular, idx)
		}
	}
	return insular
}

// GetOpenClusters returns clusters with Echo Index below the open threshold.
func (c *EchoIndexComputer) GetOpenClusters() []*ClusterEchoIndex {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var open []*ClusterEchoIndex
	for _, idx := range c.clusterIndices {
		if idx.Category == EchoCategoryOpen {
			open = append(open, idx)
		}
	}
	return open
}

// LastComputedTime returns when indices were last computed.
func (c *EchoIndexComputer) LastComputedTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastComputed
}

// RecordCount returns the number of amplification records in the window.
func (c *EchoIndexComputer) RecordCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.records)
}

// ClusterCount returns the number of tracked clusters.
func (c *EchoIndexComputer) ClusterCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.clusterIndices)
}

// GetNodeClusterIndex returns the Echo Index for the cluster containing a node.
// This is the value displayed on profile cards.
func (c *EchoIndexComputer) GetNodeClusterIndex(nodeID, clusterID string) *ClusterEchoIndex {
	return c.GetClusterIndex(clusterID)
}

// Clear removes all records and computed indices.
func (c *EchoIndexComputer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.records = c.records[:0]
	c.clusterIndices = make(map[string]*ClusterEchoIndex)
	c.lastComputed = time.Time{}
}

// EchoShadow represents the Anonymous Layer equivalent of Echo Index.
// It uses the same computation but applied to Specter clusters.
type EchoShadow struct {
	*EchoIndexComputer
}

// NewEchoShadow creates a new Echo Shadow computer for the Anonymous Layer.
func NewEchoShadow() *EchoShadow {
	return &EchoShadow{
		EchoIndexComputer: NewEchoIndexComputer(),
	}
}

// NewEchoShadowWithConfig creates an Echo Shadow with custom configuration.
func NewEchoShadowWithConfig(cfg EchoIndexConfig) *EchoShadow {
	return &EchoShadow{
		EchoIndexComputer: NewEchoIndexComputerWithConfig(cfg),
	}
}

// RecordSpecterAmplification records an anonymous amplification event.
func (s *EchoShadow) RecordSpecterAmplification(
	specterAmplifierID, specterAuthorID, waveID string,
	amplifierCluster, authorCluster string,
) {
	s.RecordAmplificationSimple(
		specterAmplifierID, specterAuthorID, waveID,
		amplifierCluster, authorCluster,
	)
}

// GetSpecterClusterShadow returns the Echo Shadow for a Specter cluster.
func (s *EchoShadow) GetSpecterClusterShadow(clusterID string) *ClusterEchoIndex {
	return s.GetClusterIndex(clusterID)
}

// ColorForEchoIndex returns an RGB color value for visualization.
// Insular (high) = warm colors (255, 165, 0 = orange to 255, 0, 0 = red)
// Open (low) = cool colors (0, 0, 255 = blue to 0, 255, 0 = green)
// Neutral = gray (128, 128, 128)
func ColorForEchoIndex(index float64) (r, g, b uint8) {
	// Clamp to valid range.
	index = math.Max(0, math.Min(1, index))

	switch {
	case index >= EchoIndexEchoChamber:
		// Insular: orange to red based on how far above threshold.
		intensity := (index - EchoIndexEchoChamber) / (1.0 - EchoIndexEchoChamber)
		return 255, uint8(165 * (1 - intensity)), 0

	case index <= EchoIndexOutwardOpen:
		// Open: blue to green based on how far below threshold.
		intensity := (EchoIndexOutwardOpen - index) / EchoIndexOutwardOpen
		return 0, uint8(255 * intensity), uint8(255 * (1 - intensity))

	default:
		// Neutral: gray.
		return 128, 128, 128
	}
}
