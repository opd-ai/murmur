// Package modes implements behavioral separation guidance.
// Per SECURITY_PRIVACY.md, behavioral correlation is a user-level risk that
// the protocol cannot fully address. This module provides guidance and metrics
// to help users understand and manage their activity patterns.
package modes

import (
	"sync"
	"time"
)

// BehavioralGuidance provides recommendations for activity pattern differentiation.
// Per SECURITY_PRIVACY.md, users should vary their behavior between layers
// (different topics, different writing style, different activity schedule)
// if they want strong unlinkability.
type BehavioralGuidance struct {
	mu              sync.RWMutex
	surfaceStats    ActivityStats
	specterStats    ActivityStats
	recommendations []Recommendation
}

// ActivityStats tracks activity patterns for correlation risk assessment.
type ActivityStats struct {
	// Time-based patterns.
	LastActivity time.Time
	ActiveHours  [24]int // Activity count per hour of day
	ActiveDays   [7]int  // Activity count per day of week

	// Content patterns.
	TopicCounts   map[string]int
	AvgMessageLen float64
	TotalMessages int

	// Timing patterns.
	AvgResponseTime time.Duration
	AvgPostInterval time.Duration
}

// CorrelationRisk represents the estimated risk of identity correlation.
type CorrelationRisk int

const (
	RiskLow CorrelationRisk = iota
	RiskMedium
	RiskHigh
)

func (r CorrelationRisk) String() string {
	switch r {
	case RiskLow:
		return "Low"
	case RiskMedium:
		return "Medium"
	case RiskHigh:
		return "High"
	default:
		return "Unknown"
	}
}

// Recommendation is a behavioral guidance recommendation.
type Recommendation struct {
	Category    string
	Title       string
	Description string
	Priority    int // 1=high, 2=medium, 3=low
}

// NewBehavioralGuidance creates a new behavioral guidance instance.
func NewBehavioralGuidance() *BehavioralGuidance {
	return &BehavioralGuidance{
		surfaceStats: ActivityStats{TopicCounts: make(map[string]int)},
		specterStats: ActivityStats{TopicCounts: make(map[string]int)},
	}
}

// RecordSurfaceActivity records Surface identity activity.
func (bg *BehavioralGuidance) RecordSurfaceActivity(topic string, messageLen int) {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	bg.recordActivity(&bg.surfaceStats, topic, messageLen)
}

// RecordSpecterActivity records Specter identity activity.
func (bg *BehavioralGuidance) RecordSpecterActivity(topic string, messageLen int) {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	bg.recordActivity(&bg.specterStats, topic, messageLen)
}

// recordActivity updates activity stats.
func (bg *BehavioralGuidance) recordActivity(stats *ActivityStats, topic string, messageLen int) {
	now := time.Now()

	// Update time-based patterns.
	stats.ActiveHours[now.Hour()]++
	stats.ActiveDays[now.Weekday()]++

	// Update content patterns.
	stats.TopicCounts[topic]++
	stats.TotalMessages++

	// Update average message length.
	total := stats.AvgMessageLen * float64(stats.TotalMessages-1)
	stats.AvgMessageLen = (total + float64(messageLen)) / float64(stats.TotalMessages)

	// Update timing patterns.
	if !stats.LastActivity.IsZero() {
		interval := now.Sub(stats.LastActivity)
		avgTotal := stats.AvgPostInterval * time.Duration(stats.TotalMessages-1)
		stats.AvgPostInterval = (avgTotal + interval) / time.Duration(stats.TotalMessages)
	}

	stats.LastActivity = now
}

// AssessCorrelationRisk evaluates the risk of identity correlation.
// Per SECURITY_PRIVACY.md, behavioral correlation risks include:
// - Similar activity timing
// - Topic overlap
// - Message style similarity
func (bg *BehavioralGuidance) AssessCorrelationRisk() CorrelationRisk {
	bg.mu.RLock()
	defer bg.mu.RUnlock()

	risk := 0

	// Check timing correlation.
	risk += bg.assessTimingCorrelation()

	// Check topic overlap.
	risk += bg.assessTopicOverlap()

	// Check message style similarity.
	risk += bg.assessStyleSimilarity()

	if risk >= 6 {
		return RiskHigh
	} else if risk >= 3 {
		return RiskMedium
	}
	return RiskLow
}

// assessTimingCorrelation checks for similar activity timing patterns.
func (bg *BehavioralGuidance) assessTimingCorrelation() int {
	risk := 0

	// Compare active hours.
	surfacePeak := bg.findPeakHours(bg.surfaceStats.ActiveHours[:])
	specterPeak := bg.findPeakHours(bg.specterStats.ActiveHours[:])

	overlap := bg.countOverlap(surfacePeak, specterPeak)
	if overlap > 2 {
		risk += 3 // High timing correlation
	} else if overlap > 0 {
		risk += 1 // Some timing correlation
	}

	return risk
}

// findPeakHours finds the top 3 active hours.
func (bg *BehavioralGuidance) findPeakHours(hours []int) []int {
	type hourCount struct {
		hour  int
		count int
	}

	var hc []hourCount
	for i, c := range hours {
		if c > 0 {
			hc = append(hc, hourCount{i, c})
		}
	}

	// Simple sort (bubble for small array).
	for i := 0; i < len(hc); i++ {
		for j := i + 1; j < len(hc); j++ {
			if hc[j].count > hc[i].count {
				hc[i], hc[j] = hc[j], hc[i]
			}
		}
	}

	// Take top 3.
	var peaks []int
	for i := 0; i < len(hc) && i < 3; i++ {
		peaks = append(peaks, hc[i].hour)
	}
	return peaks
}

// countOverlap counts overlapping elements between two slices.
func (bg *BehavioralGuidance) countOverlap(a, b []int) int {
	aSet := make(map[int]bool)
	for _, v := range a {
		aSet[v] = true
	}

	count := 0
	for _, v := range b {
		if aSet[v] {
			count++
		}
	}
	return count
}

// assessTopicOverlap checks for overlapping topics between identities.
func (bg *BehavioralGuidance) assessTopicOverlap() int {
	surfaceTopics := make(map[string]bool)
	for topic := range bg.surfaceStats.TopicCounts {
		surfaceTopics[topic] = true
	}

	overlap := 0
	for topic := range bg.specterStats.TopicCounts {
		if surfaceTopics[topic] {
			overlap++
		}
	}

	totalTopics := len(surfaceTopics) + len(bg.specterStats.TopicCounts) - overlap
	if totalTopics == 0 {
		return 0
	}

	overlapRatio := float64(overlap) / float64(totalTopics)
	if overlapRatio > 0.5 {
		return 3 // High topic overlap
	} else if overlapRatio > 0.2 {
		return 1 // Some topic overlap
	}
	return 0
}

// assessStyleSimilarity checks for similar message styles.
func (bg *BehavioralGuidance) assessStyleSimilarity() int {
	if bg.surfaceStats.TotalMessages < 5 || bg.specterStats.TotalMessages < 5 {
		return 0 // Not enough data
	}

	// Compare average message length.
	lenDiff := bg.surfaceStats.AvgMessageLen - bg.specterStats.AvgMessageLen
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}

	avgLen := (bg.surfaceStats.AvgMessageLen + bg.specterStats.AvgMessageLen) / 2
	if avgLen == 0 {
		return 0
	}

	lenSimilarity := 1.0 - (lenDiff / avgLen)
	if lenSimilarity > 0.8 {
		return 2 // High style similarity
	} else if lenSimilarity > 0.5 {
		return 1 // Some style similarity
	}
	return 0
}

// GetRecommendations returns behavioral separation recommendations.
func (bg *BehavioralGuidance) GetRecommendations() []Recommendation {
	bg.mu.RLock()
	defer bg.mu.RUnlock()

	var recs []Recommendation

	// Always provide base recommendations per SECURITY_PRIVACY.md.
	recs = append(recs, Recommendation{
		Category:    "Timing",
		Title:       "Vary your activity schedule",
		Description: "Use your Surface and Specter identities at different times of day to reduce timing correlation. Avoid posting to both identities within short time windows.",
		Priority:    1,
	})

	recs = append(recs, Recommendation{
		Category:    "Topics",
		Title:       "Diversify your topics",
		Description: "Discuss different subjects on each identity. If you have professional discussions on Surface, use Specter for unrelated hobbies or interests.",
		Priority:    2,
	})

	recs = append(recs, Recommendation{
		Category:    "Style",
		Title:       "Vary your writing style",
		Description: "Use different vocabulary, sentence structures, and punctuation patterns between identities. Consider using different languages if you're multilingual.",
		Priority:    2,
	})

	recs = append(recs, Recommendation{
		Category:    "Connections",
		Title:       "Separate social graphs",
		Description: "Avoid connecting with the same people on both identities. Your Specter connections should be distinct from your Surface connections.",
		Priority:    1,
	})

	recs = append(recs, Recommendation{
		Category:    "Devices",
		Title:       "Consider device separation",
		Description: "For maximum unlinkability, use different devices or network connections for each identity. A shared device can leak correlation through network timing.",
		Priority:    3,
	})

	// Add context-specific recommendations based on current risk.
	risk := bg.assessTimingCorrelation()
	if risk > 1 {
		recs = append(recs, Recommendation{
			Category:    "Warning",
			Title:       "High timing correlation detected",
			Description: "Your Surface and Specter activities occur at similar times. Consider adding delays between switching identities.",
			Priority:    1,
		})
	}

	if bg.assessTopicOverlap() > 1 {
		recs = append(recs, Recommendation{
			Category:    "Warning",
			Title:       "Topic overlap detected",
			Description: "You're discussing similar topics on both identities. This increases correlation risk.",
			Priority:    1,
		})
	}

	return recs
}

// GetActivitySummary returns a summary of activity patterns.
func (bg *BehavioralGuidance) GetActivitySummary() (surface, specter ActivityStats) {
	bg.mu.RLock()
	defer bg.mu.RUnlock()

	return bg.surfaceStats, bg.specterStats
}

// Reset clears all activity statistics.
func (bg *BehavioralGuidance) Reset() {
	bg.mu.Lock()
	defer bg.mu.Unlock()

	bg.surfaceStats = ActivityStats{TopicCounts: make(map[string]int)}
	bg.specterStats = ActivityStats{TopicCounts: make(map[string]int)}
}

// GuidanceMessages returns general behavioral guidance messages.
// Per SECURITY_PRIVACY.md, these messages advise users on maintaining unlinkability.
func GuidanceMessages() []string {
	return []string{
		"Behavioral correlation is the primary residual risk when using both Surface and Specter identities.",
		"Vary your activity patterns - different times, different topics, different writing styles.",
		"Do not discuss information on the Anonymous Layer that uniquely identifies you on the Surface Layer.",
		"The Masked Event mechanic provides temporary escape from behavioral correlation using single-use identities.",
		"For strong long-term anonymity, combine Shroud routing with external anonymity networks (Tor, VPN).",
		"Avoid long-term behavioral patterns that aid correlation analysis.",
	}
}
