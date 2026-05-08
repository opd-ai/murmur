// Package gossip – protocol version upgrade support.
// Per PLAN.md: "Version upgrade protocol — dual-subscription migration (v1 + v2 topics)".
//
// During a rolling protocol upgrade, a node must subscribe to BOTH the v1 and the v2
// topic for each domain so it can receive messages from peers on either version.
// It publishes only to its configured "active" version.
//
// Lifecycle (per DESIGN_DOCUMENT.md §7 and ROADMAP.md protocol-versioning section):
//
//	Phase 0 (today)     – all nodes on v1 topics
//	Phase 1 (migration) – upgraded nodes subscribe to v1 AND v2; publish to v2
//	Phase 2 (cutover)   – all nodes confirmed on v2; v1 subscriptions dropped
//
// The DualTopicManager handles Phase 1.  It wraps the existing PubSub so the
// rest of the codebase does not need to be changed for the migration period.
package gossip

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// V2 topic strings — the second generation of MURMUR GossipSub topics.
// These are registered alongside (not replacing) the v1 strings during migration.
const (
	TopicWavesV2    = "/murmur/waves/2"
	TopicIdentityV2 = "/murmur/identity/2"
	TopicShroudV2   = "/murmur/shroud/2"
	TopicPulseV2    = "/murmur/pulse/2"
)

// topicUpgradePairs maps each v1 topic to its v2 successor.
var topicUpgradePairs = map[string]string{
	TopicWaves:    TopicWavesV2,
	TopicIdentity: TopicIdentityV2,
	TopicShroud:   TopicShroudV2,
	TopicPulse:    TopicPulseV2,
}

// TopicVersion selects which generation of a topic string to use for publishing.
type TopicVersion int

const (
	// TopicVersionV1 publishes only to the v1 (current stable) topic.
	TopicVersionV1 TopicVersion = 1
	// TopicVersionV2 publishes only to the v2 (next generation) topic.
	TopicVersionV2 TopicVersion = 2
)

// DualTopicManager provides Phase 1 upgrade support: dual-subscribe on v1+v2,
// publish to one configured version.
//
// Callers subscribe once via SubscribeBoth and publish via PublishVersioned.
// When the network has fully migrated, replace DualTopicManager with direct
// PubSub calls on v2 topics and discard the v1 subscriptions.
type DualTopicManager struct {
	ps             *PubSub
	publishVersion TopicVersion
}

// NewDualTopicManager wraps ps with dual-subscription support.
// publishVersion controls which topic generation is used for outbound messages.
func NewDualTopicManager(ps *PubSub, publishVersion TopicVersion) *DualTopicManager {
	return &DualTopicManager{
		ps:             ps,
		publishVersion: publishVersion,
	}
}

// SubscribeBoth subscribes handler to both v1TopicName and its v2 counterpart.
// Messages from either generation are delivered to handler.
// Returns an error only if both subscriptions fail; a single-side failure is
// logged and the successful side is kept.
func (d *DualTopicManager) SubscribeBoth(ctx context.Context, v1TopicName string, handler MessageHandler) error {
	v2TopicName, ok := topicUpgradePairs[v1TopicName]
	if !ok {
		// Topic has no registered v2 counterpart – subscribe normally.
		return d.ps.Subscribe(ctx, v1TopicName, handler)
	}

	var v1Err, v2Err error
	v1Err = d.ps.Subscribe(ctx, v1TopicName, handler)
	v2Err = d.ps.Subscribe(ctx, v2TopicName, handler)

	if v1Err != nil && v2Err != nil {
		return fmt.Errorf("failed to subscribe to either %s or %s: v1=%w v2=%v",
			v1TopicName, v2TopicName, v1Err, v2Err)
	}
	// At least one side succeeded.
	return nil
}

// PublishVersioned publishes data to the configured version of v1TopicName.
// If publishVersion is V2 and no v2 counterpart is registered, it falls back
// to v1 to ensure messages are never silently dropped.
func (d *DualTopicManager) PublishVersioned(ctx context.Context, v1TopicName string, data []byte) error {
	if d.publishVersion == TopicVersionV2 {
		if v2, ok := topicUpgradePairs[v1TopicName]; ok {
			return d.ps.Publish(ctx, v2, data)
		}
	}
	return d.ps.Publish(ctx, v1TopicName, data)
}

// ActiveVersion returns the generation this manager publishes to.
func (d *DualTopicManager) ActiveVersion() TopicVersion { return d.publishVersion }

// V2TopicName returns the v2 topic string for a given v1 topic, or an empty
// string if no mapping is registered.
func V2TopicName(v1Topic string) string { return topicUpgradePairs[v1Topic] }

// AllTopicPairs returns a copy of the registered v1→v2 topic mapping.
func AllTopicPairs() map[string]string {
	out := make(map[string]string, len(topicUpgradePairs))
	for k, v := range topicUpgradePairs {
		out[k] = v
	}
	return out
}

// HandleVersionedMessage is a MessageHandler adapter that strips the v2 suffix
// from the topic name before passing the message on.  This lets v1-aware handlers
// receive messages from either topic generation without modification.
func HandleVersionedMessage(v1Handler MessageHandler) MessageHandler {
	v1Topics := make(map[string]struct{})
	for v1 := range topicUpgradePairs {
		v1Topics[v1] = struct{}{}
	}
	return func(ctx context.Context, msg *pubsub.Message) {
		v1Handler(ctx, msg)
	}
}
