// Package modes implements network separation based on privacy mode.
// Per SHADOW_GRADIENT.md §Network Separation, users in different modes
// should have distinct network access patterns.
package modes

import (
	"errors"
	"sync"
)

// Network separation errors.
var (
	ErrTopicNotAllowed = errors.New("topic not allowed in current mode")
	ErrOperationDenied = errors.New("operation denied in current mode")
)

// Topic categories for network separation.
const (
	// Surface Layer topics - require Surface identity.
	TopicCategorySurface = "surface"
	// Anonymous Layer topics - require Specter identity.
	TopicCategoryAnonymous = "anonymous"
	// Shared topics - available to all modes.
	TopicCategoryShared = "shared"
)

// TopicInfo describes a gossip topic and its category.
type TopicInfo struct {
	Name     string
	Category string
}

// NetworkSeparator enforces topic access based on privacy mode.
// Per SHADOW_GRADIENT.md, network traffic should be separated:
// - Open mode: Surface topics only
// - Hybrid mode: Both Surface and Anonymous topics
// - Guarded mode: Both Surface and Anonymous topics (with padding)
// - Fortress mode: Anonymous topics only (routed through Shroud)
type NetworkSeparator struct {
	mu            sync.RWMutex
	modeManager   *Manager
	allowedTopics map[string]bool
	topicRegistry map[string]TopicInfo
}

// NewNetworkSeparator creates a new NetworkSeparator linked to a mode manager.
func NewNetworkSeparator(manager *Manager) *NetworkSeparator {
	ns := &NetworkSeparator{
		modeManager:   manager,
		allowedTopics: make(map[string]bool),
		topicRegistry: make(map[string]TopicInfo),
	}

	// Register mode change listener to update allowed topics.
	manager.OnTransition(func(old, new Mode) {
		ns.updateAllowedTopics(new)
	})

	// Initialize allowed topics for current mode.
	ns.updateAllowedTopics(manager.Current())

	return ns
}

// RegisterTopic registers a topic with its category.
func (ns *NetworkSeparator) RegisterTopic(name, category string) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.topicRegistry[name] = TopicInfo{Name: name, Category: category}
	ns.recalculateAllowedTopics()
}

// IsTopicAllowed checks if a topic is allowed in the current mode.
func (ns *NetworkSeparator) IsTopicAllowed(topic string) bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.allowedTopics[topic]
}

// AllowedTopics returns the list of topics allowed in the current mode.
func (ns *NetworkSeparator) AllowedTopics() []string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	var topics []string
	for topic, allowed := range ns.allowedTopics {
		if allowed {
			topics = append(topics, topic)
		}
	}
	return topics
}

// FilterTopics returns only the allowed topics from the input list.
func (ns *NetworkSeparator) FilterTopics(topics []string) []string {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	var filtered []string
	for _, topic := range topics {
		if ns.allowedTopics[topic] {
			filtered = append(filtered, topic)
		}
	}
	return filtered
}

// updateAllowedTopics recalculates allowed topics when mode changes.
func (ns *NetworkSeparator) updateAllowedTopics(mode Mode) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.allowedTopics = make(map[string]bool)
	ns.recalculateAllowedTopics()
}

// recalculateAllowedTopics updates the allowed topics map based on current mode.
// Must be called with lock held.
func (ns *NetworkSeparator) recalculateAllowedTopics() {
	mode := ns.modeManager.Current()

	for topic, info := range ns.topicRegistry {
		allowed := ns.isTopicAllowedForMode(info.Category, mode)
		ns.allowedTopics[topic] = allowed
	}
}

// isTopicAllowedForMode checks if a topic category is allowed for a mode.
func (ns *NetworkSeparator) isTopicAllowedForMode(category string, mode Mode) bool {
	switch category {
	case TopicCategorySurface:
		// Surface topics require Surface identity.
		return mode.AllowsSurface()
	case TopicCategoryAnonymous:
		// Anonymous topics require Specter identity.
		return mode.AllowsSpecter()
	case TopicCategoryShared:
		// Shared topics are always allowed.
		return true
	default:
		return false
	}
}

// CanPublish checks if publishing to a topic is allowed.
func (ns *NetworkSeparator) CanPublish(topic string) error {
	if !ns.IsTopicAllowed(topic) {
		return ErrTopicNotAllowed
	}
	return nil
}

// CanSubscribe checks if subscribing to a topic is allowed.
func (ns *NetworkSeparator) CanSubscribe(topic string) error {
	if !ns.IsTopicAllowed(topic) {
		return ErrTopicNotAllowed
	}
	return nil
}

// NetworkOperation represents a network operation type.
type NetworkOperation int

const (
	// OperationPublishSurface is a Surface identity publication.
	OperationPublishSurface NetworkOperation = iota
	// OperationPublishAnonymous is an anonymous (Specter) publication.
	OperationPublishAnonymous
	// OperationConnectSurface creates a Surface connection.
	OperationConnectSurface
	// OperationConnectAnonymous creates an anonymous connection.
	OperationConnectAnonymous
	// OperationShroudRelay relays traffic through Shroud.
	OperationShroudRelay
)

// CanPerformOperation checks if an operation is allowed in the current mode.
func (ns *NetworkSeparator) CanPerformOperation(op NetworkOperation) error {
	mode := ns.modeManager.Current()

	switch op {
	case OperationPublishSurface, OperationConnectSurface:
		if !mode.AllowsSurface() {
			return ErrOperationDenied
		}
	case OperationPublishAnonymous, OperationConnectAnonymous:
		if !mode.AllowsSpecter() {
			return ErrOperationDenied
		}
	case OperationShroudRelay:
		// Shroud relay is allowed for all Specter-enabled modes.
		if !mode.AllowsSpecter() {
			return ErrOperationDenied
		}
	}

	return nil
}

// SeparationPolicy describes the network separation policy for a mode.
type SeparationPolicy struct {
	Mode                Mode
	AllowSurfaceTopics  bool
	AllowAnonymousTopics bool
	RequireShroud       bool
	RequireTrafficPadding bool
}

// GetPolicy returns the separation policy for the current mode.
func (ns *NetworkSeparator) GetPolicy() SeparationPolicy {
	mode := ns.modeManager.Current()
	caps := mode.Capabilities()

	return SeparationPolicy{
		Mode:                mode,
		AllowSurfaceTopics:  caps.SurfaceAllowed,
		AllowAnonymousTopics: caps.SpecterAllowed,
		RequireShroud:       caps.ShroudRequired,
		RequireTrafficPadding: caps.TrafficPaddingActive,
	}
}
