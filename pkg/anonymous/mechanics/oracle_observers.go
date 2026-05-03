// Package mechanics - Oracle Pool metric observers.
// This file implements concrete MetricObserver implementations for Oracle Pools.
package mechanics

import (
	"time"
)

type GossipVolumeObserver struct {
	// GetVolume queries the network for message count.
	// Injected dependency to allow testing without actual network.
	GetVolume func(topic string, start, end time.Time) (int64, error)
}

// Observe returns the gossip volume for the given parameters.
func (o *GossipVolumeObserver) Observe(params MetricParams) (float64, error) {
	if o.GetVolume == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetVolume(params.Topic, params.StartTime, params.EndTime)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricGossipVolume.
func (o *GossipVolumeObserver) MetricType() ObservableMetricType {
	return MetricGossipVolume
}

// TerritoryCountObserver observes territory counts.
type TerritoryCountObserver struct {
	// GetTerritoryCount queries for territory count in a region.
	GetTerritoryCount func(region string) (int, error)
}

// Observe returns the territory count.
func (o *TerritoryCountObserver) Observe(params MetricParams) (float64, error) {
	if o.GetTerritoryCount == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetTerritoryCount(params.Region)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricTerritoryCount.
func (o *TerritoryCountObserver) MetricType() ObservableMetricType {
	return MetricTerritoryCount
}

// NodeCountObserver observes active node counts.
type NodeCountObserver struct {
	// GetNodeCount queries for active node count.
	GetNodeCount func() (int, error)
}

// Observe returns the node count.
func (o *NodeCountObserver) Observe(_ MetricParams) (float64, error) {
	if o.GetNodeCount == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetNodeCount()
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricNodeCount.
func (o *NodeCountObserver) MetricType() ObservableMetricType {
	return MetricNodeCount
}

// WaveCountObserver observes Wave publication counts.
type WaveCountObserver struct {
	// GetWaveCount queries for Waves published in a period.
	GetWaveCount func(start, end time.Time) (int64, error)
}

// Observe returns the Wave count.
func (o *WaveCountObserver) Observe(params MetricParams) (float64, error) {
	if o.GetWaveCount == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetWaveCount(params.StartTime, params.EndTime)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricWaveCount.
func (o *WaveCountObserver) MetricType() ObservableMetricType {
	return MetricWaveCount
}

// SpecterCountObserver observes active Specter counts.
type SpecterCountObserver struct {
	// GetSpecterCount queries for active Specter count.
	GetSpecterCount func() (int, error)
}

// Observe returns the Specter count.
func (o *SpecterCountObserver) Observe(_ MetricParams) (float64, error) {
	if o.GetSpecterCount == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetSpecterCount()
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricSpecterCount.
func (o *SpecterCountObserver) MetricType() ObservableMetricType {
	return MetricSpecterCount
}

// EventParticipationObserver observes Masked Event participation.
type EventParticipationObserver struct {
	// GetEventParticipation queries for event participation.
	GetEventParticipation func(start, end time.Time) (int, error)
}

// Observe returns event participation count.
func (o *EventParticipationObserver) Observe(params MetricParams) (float64, error) {
	if o.GetEventParticipation == nil {
		return 0, ErrMissingObservableData
	}
	count, err := o.GetEventParticipation(params.StartTime, params.EndTime)
	if err != nil {
		return 0, err
	}
	return float64(count), nil
}

// MetricType returns MetricEventParticipation.
func (o *EventParticipationObserver) MetricType() ObservableMetricType {
	return MetricEventParticipation
}

// HuntSuccessObserver observes Specter Hunt success rates.
type HuntSuccessObserver struct {
	// GetHuntSuccessRate queries for hunt success rate (0.0-1.0).
	GetHuntSuccessRate func(start, end time.Time) (float64, error)
}

// Observe returns hunt success rate.
func (o *HuntSuccessObserver) Observe(params MetricParams) (float64, error) {
	if o.GetHuntSuccessRate == nil {
		return 0, ErrMissingObservableData
	}
	return o.GetHuntSuccessRate(params.StartTime, params.EndTime)
}

// MetricType returns MetricHuntSuccess.
func (o *HuntSuccessObserver) MetricType() ObservableMetricType {
	return MetricHuntSuccess
}

// MetricTypeString returns a human-readable name for a metric type.
func MetricTypeString(mt ObservableMetricType) string {
	switch mt {
	case MetricGossipVolume:
		return "GossipVolume"
	case MetricTerritoryCount:
		return "TerritoryCount"
	case MetricEventParticipation:
		return "EventParticipation"
	case MetricNodeCount:
		return "NodeCount"
	case MetricWaveCount:
		return "WaveCount"
	case MetricSpecterCount:
		return "SpecterCount"
	case MetricHuntSuccess:
		return "HuntSuccess"
	case MetricCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// VerificationStateString returns a human-readable name for a verification state.
func VerificationStateString(vs VerificationState) string {
	switch vs {
	case VerificationPending:
		return "Pending"
	case VerificationCollect:
		return "Collecting"
	case VerificationConsensus:
		return "Consensus"
	case VerificationConfirmed:
		return "Confirmed"
	case VerificationDisputed:
		return "Disputed"
	case VerificationFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}
