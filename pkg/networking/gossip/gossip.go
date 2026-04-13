// Package gossip provides GossipSub configuration, topic management, and peer scoring.
// Per TECHNICAL_IMPLEMENTATION.md §3.1, topics include /murmur/waves/1,
// /murmur/identity/1, /murmur/shroud/1, and /murmur/pulse/1.
package gossip

// Topic names per TECHNICAL_IMPLEMENTATION.md §3.1.
const (
	TopicWaves    = "/murmur/waves/1"
	TopicIdentity = "/murmur/identity/1"
	TopicShroud   = "/murmur/shroud/1"
	TopicPulse    = "/murmur/pulse/1"
)

// TODO: Implement GossipSub setup per PLAN.md Step 9.
