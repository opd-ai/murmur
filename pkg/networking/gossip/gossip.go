// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, topics include /murmur/waves/1,
// /murmur/identity/1, /murmur/shroud/1, and /murmur/pulse/1.
package gossip

import (
	"context"
	"fmt"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Topic names per TECHNICAL_IMPLEMENTATION.md §3.1.
const (
	TopicWaves    = "/murmur/waves/1"
	TopicIdentity = "/murmur/identity/1"
	TopicShroud   = "/murmur/shroud/1"
	TopicPulse    = "/murmur/pulse/1"
)

// HeartbeatInterval is the interval for sending heartbeat pings.
// Per DESIGN_DOCUMENT.md Part II §6.
const HeartbeatInterval = 30 * time.Second

// PubSub wraps libp2p pubsub with MURMUR-specific topic management.
type PubSub struct {
	ps     *pubsub.PubSub
	h      host.Host
	topics map[string]*pubsub.Topic
	subs   map[string]*pubsub.Subscription
	mu     sync.RWMutex
}

// MessageHandler is called for each received message on a topic.
type MessageHandler func(ctx context.Context, msg *pubsub.Message)

// New creates a new PubSub instance with GossipSub and peer scoring.
// Per DESIGN_DOCUMENT.md Part II §7, peer scoring penalizes invalid signatures,
// failed PoW, expired TTL, and applies IP colocation penalty for Sybil resistance.
func New(ctx context.Context, h host.Host) (*PubSub, error) {
	peerScoreParams := buildPeerScoreParams()
	thresholds := buildScoreThresholds()

	ps, err := pubsub.NewGossipSub(ctx, h,
		pubsub.WithPeerScore(peerScoreParams, thresholds),
		pubsub.WithFloodPublish(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GossipSub: %w", err)
	}

	return &PubSub{
		ps:     ps,
		h:      h,
		topics: make(map[string]*pubsub.Topic),
		subs:   make(map[string]*pubsub.Subscription),
	}, nil
}

// buildPeerScoreParams constructs GossipSub peer scoring parameters per DESIGN_DOCUMENT.md.
func buildPeerScoreParams() *pubsub.PeerScoreParams {
	topicParams := buildDefaultTopicParams()

	return &pubsub.PeerScoreParams{
		Topics:                      map[string]*pubsub.TopicScoreParams{TopicWaves: topicParams, TopicIdentity: topicParams, TopicShroud: topicParams, TopicPulse: topicParams},
		AppSpecificScore:            func(p peer.ID) float64 { return 0 },
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    -10,
		IPColocationFactorThreshold: 3,
		BehaviourPenaltyWeight:      -1,
		BehaviourPenaltyDecay:       0.9,
		DecayInterval:               1 * time.Minute,
		DecayToZero:                 0.01,
	}
}

// buildDefaultTopicParams returns topic-specific scoring parameters.
func buildDefaultTopicParams() *pubsub.TopicScoreParams {
	return &pubsub.TopicScoreParams{
		TopicWeight:                    1,
		TimeInMeshWeight:               0.01,
		TimeInMeshQuantum:              time.Second,
		TimeInMeshCap:                  100,
		FirstMessageDeliveriesWeight:   1,
		FirstMessageDeliveriesDecay:    0.9,
		FirstMessageDeliveriesCap:      100,
		InvalidMessageDeliveriesWeight: -10,
		InvalidMessageDeliveriesDecay:  0.9,
	}
}

// buildScoreThresholds returns peer score thresholds for GossipSub.
func buildScoreThresholds() *pubsub.PeerScoreThresholds {
	return &pubsub.PeerScoreThresholds{
		GossipThreshold:             -100,
		PublishThreshold:            -1000,
		GraylistThreshold:           -10000,
		AcceptPXThreshold:           0,
		OpportunisticGraftThreshold: 5,
	}
}

// Join joins a topic and returns the topic handle.
func (p *PubSub) Join(topicName string) (*pubsub.Topic, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if topic, ok := p.topics[topicName]; ok {
		return topic, nil
	}

	topic, err := p.ps.Join(topicName)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic %s: %w", topicName, err)
	}

	p.topics[topicName] = topic
	return topic, nil
}

// Subscribe subscribes to a topic and starts receiving messages.
// The handler is called for each received message.
func (p *PubSub) Subscribe(ctx context.Context, topicName string, handler MessageHandler) error {
	topic, err := p.Join(topicName)
	if err != nil {
		return err
	}

	sub, err := p.registerSubscription(topicName, topic)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Already subscribed
	}

	p.startMessageHandler(ctx, sub, handler)
	return nil
}

// registerSubscription creates a subscription if one doesn't exist.
// Returns nil subscription if already subscribed.
func (p *PubSub) registerSubscription(topicName string, topic *pubsub.Topic) (*pubsub.Subscription, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.subs[topicName]; ok {
		return nil, nil // Already subscribed
	}

	sub, err := topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topicName, err)
	}
	p.subs[topicName] = sub
	return sub, nil
}

// startMessageHandler launches a goroutine to process incoming messages.
func (p *PubSub) startMessageHandler(ctx context.Context, sub *pubsub.Subscription, handler MessageHandler) {
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				return // Context cancelled or subscription closed
			}
			handler(ctx, msg)
		}
	}()
}

// Publish publishes a message to a topic.
func (p *PubSub) Publish(ctx context.Context, topicName string, data []byte) error {
	topic, err := p.Join(topicName)
	if err != nil {
		return err
	}
	return topic.Publish(ctx, data)
}

// Topics returns the list of joined topic names.
func (p *PubSub) Topics() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.topics))
	for name := range p.topics {
		names = append(names, name)
	}
	return names
}

// TopicPeers returns the list of peers subscribed to a topic.
func (p *PubSub) TopicPeers(topicName string) []peer.ID {
	p.mu.RLock()
	topic, ok := p.topics[topicName]
	p.mu.RUnlock()

	if !ok {
		return nil
	}
	return topic.ListPeers()
}

// Close closes all subscriptions and topics.
func (p *PubSub) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sub := range p.subs {
		sub.Cancel()
	}
	p.subs = make(map[string]*pubsub.Subscription)

	for _, topic := range p.topics {
		if err := topic.Close(); err != nil {
			return fmt.Errorf("failed to close topic: %w", err)
		}
	}
	p.topics = make(map[string]*pubsub.Topic)

	return nil
}
