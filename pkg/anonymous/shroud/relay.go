// Package shroud provides three-hop onion circuit construction and relay forwarding.
// Per SECURITY_PRIVACY.md §Class 2, Shroud Nodes forward encrypted traffic with
// traffic mixing (random delay injection) and dummy traffic generation.
package shroud

import (
	"context"
	"crypto/rand"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// Relay configuration constants per SHADOW_GRADIENT.md.
const (
	// DummyPacketRate is the default rate for constant-rate padding (2 packets/sec).
	// Per ROADMAP: constant-rate dummy packets (2 per second) on active circuits.
	DummyPacketRate = 500 * time.Millisecond

	// MixDelayMean is the mean delay for exponential distribution (200ms).
	// Per ROADMAP: random delay (exponential distribution, mean 200ms).
	MixDelayMean = 200 * time.Millisecond

	// MinMixDelay is the minimum random delay added to forwarded packets.
	MinMixDelay = 50 * time.Millisecond

	// MaxMixDelay is the maximum random delay added to forwarded packets.
	// Cap at 3x mean to prevent excessive delays.
	MaxMixDelay = 600 * time.Millisecond

	// RelayBufferSize is the packet buffer size for the relay.
	RelayBufferSize = 256
)

// Relay errors.
var (
	ErrRelayNotEnabled = errors.New("relay not enabled")
	ErrRelayShutdown   = errors.New("relay is shutting down")
	ErrBufferFull      = errors.New("relay buffer full")
)

// PacketHandler processes a forwarded packet and returns the next hop and data.
type PacketHandler func(packet []byte) (nextHop string, data []byte, err error)

// PacketSender sends a packet to a peer.
type PacketSender func(peerID string, data []byte) error

// RelayStats tracks relay performance metrics.
type RelayStats struct {
	PacketsForwarded uint64
	PacketsDropped   uint64
	DummyPacketsSent uint64
	BytesForwarded   uint64
	AvgDelayMs       float64
}

// Relay implements Shroud Node message forwarding with traffic mixing.
type Relay struct {
	mu       sync.RWMutex
	beacon   *Beacon
	enabled  atomic.Bool
	shutdown atomic.Bool

	// Packet processing.
	handler PacketHandler
	sender  PacketSender

	// Traffic mixing.
	inbound chan relayPacket
	delayMu sync.Mutex
	delays  []time.Duration

	// Statistics.
	stats RelayStats

	// Cancellation.
	ctx    context.Context
	cancel context.CancelFunc
}

// relayPacket is an internal packet with metadata.
type relayPacket struct {
	data      []byte
	nextHop   string
	arriveAt  time.Time
	scheduled time.Time
}

// NewRelay creates a new Shroud relay.
func NewRelay(beacon *Beacon, handler PacketHandler, sender PacketSender) *Relay {
	ctx, cancel := context.WithCancel(context.Background())

	return &Relay{
		beacon:  beacon,
		handler: handler,
		sender:  sender,
		inbound: make(chan relayPacket, RelayBufferSize),
		delays:  make([]time.Duration, 0, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Enable enables the relay for forwarding.
func (r *Relay) Enable() {
	r.enabled.Store(true)
}

// Disable disables the relay.
func (r *Relay) Disable() {
	r.enabled.Store(false)
}

// IsEnabled returns true if the relay is enabled.
func (r *Relay) IsEnabled() bool {
	return r.enabled.Load()
}

// Start begins the relay processing and dummy traffic generation.
func (r *Relay) Start(ctx context.Context) {
	go r.processLoop(ctx)
	go r.dummyTrafficLoop(ctx)
}

// Stop stops the relay.
func (r *Relay) Stop() {
	r.shutdown.Store(true)
	r.cancel()
}

// Forward queues a packet for forwarding with traffic mixing.
func (r *Relay) Forward(packet []byte) error {
	if err := r.validateRelayState(); err != nil {
		return err
	}

	nextHop, data, err := r.handler(packet)
	if err != nil {
		atomic.AddUint64(&r.stats.PacketsDropped, 1)
		return err
	}

	return r.enqueuePacket(nextHop, data)
}

// validateRelayState checks if the relay is enabled and not shutting down.
func (r *Relay) validateRelayState() error {
	if r.shutdown.Load() {
		return ErrRelayShutdown
	}
	if !r.enabled.Load() {
		return ErrRelayNotEnabled
	}
	return nil
}

// enqueuePacket creates and queues a packet with random delay for traffic mixing.
func (r *Relay) enqueuePacket(nextHop string, data []byte) error {
	rp := relayPacket{
		data:      data,
		nextHop:   nextHop,
		arriveAt:  time.Now(),
		scheduled: time.Now().Add(r.randomDelay()),
	}

	select {
	case r.inbound <- rp:
		return nil
	default:
		atomic.AddUint64(&r.stats.PacketsDropped, 1)
		return ErrBufferFull
	}
}

// processLoop handles packet forwarding with delay.
func (r *Relay) processLoop(ctx context.Context) {
	pending := make([]relayPacket, 0, RelayBufferSize)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if r.processLoopIteration(ctx, &pending, ticker) {
			return
		}
	}
}

// processLoopIteration handles one iteration of the process loop.
// Returns true if the loop should exit.
func (r *Relay) processLoopIteration(ctx context.Context, pending *[]relayPacket, ticker *time.Ticker) bool {
	select {
	case <-ctx.Done():
		return true
	case <-r.ctx.Done():
		return true
	case pkt := <-r.inbound:
		*pending = append(*pending, pkt)
	case <-ticker.C:
		*pending = r.processPendingPackets(*pending)
	}
	return false
}

// processPendingPackets sends ready packets and returns remaining.
func (r *Relay) processPendingPackets(pending []relayPacket) []relayPacket {
	now := time.Now()
	remaining := make([]relayPacket, 0, len(pending))

	for _, pkt := range pending {
		if r.isPacketReady(pkt, now) {
			r.sendPacket(pkt)
		} else {
			remaining = append(remaining, pkt)
		}
	}
	return remaining
}

// isPacketReady checks if a packet's scheduled time has arrived.
func (r *Relay) isPacketReady(pkt relayPacket, now time.Time) bool {
	return now.After(pkt.scheduled) || now.Equal(pkt.scheduled)
}

// sendPacket sends a packet to its next hop.
func (r *Relay) sendPacket(pkt relayPacket) {
	if err := r.sender(pkt.nextHop, pkt.data); err != nil {
		atomic.AddUint64(&r.stats.PacketsDropped, 1)
		return
	}

	atomic.AddUint64(&r.stats.PacketsForwarded, 1)
	atomic.AddUint64(&r.stats.BytesForwarded, uint64(len(pkt.data)))

	// Track delay for statistics.
	delay := time.Since(pkt.arriveAt)
	r.recordDelay(delay)
}

// recordDelay records a delay value for statistics.
func (r *Relay) recordDelay(d time.Duration) {
	r.delayMu.Lock()
	defer r.delayMu.Unlock()

	r.delays = append(r.delays, d)
	if len(r.delays) > 100 {
		r.delays = r.delays[1:]
	}

	// Update average.
	var total time.Duration
	for _, delay := range r.delays {
		total += delay
	}
	r.mu.Lock()
	r.stats.AvgDelayMs = float64(total.Milliseconds()) / float64(len(r.delays))
	r.mu.Unlock()
}

// dummyTrafficLoop generates constant-rate padding per SHADOW_GRADIENT.md §Traffic Padding.
func (r *Relay) dummyTrafficLoop(ctx context.Context) {
	ticker := time.NewTicker(DummyPacketRate)
	defer ticker.Stop()

	for {
		if r.dummyTrafficIteration(ctx, ticker) {
			return
		}
	}
}

// dummyTrafficIteration handles one iteration of the dummy traffic loop.
// Returns true if the loop should exit.
func (r *Relay) dummyTrafficIteration(ctx context.Context, ticker *time.Ticker) bool {
	select {
	case <-ctx.Done():
		return true
	case <-r.ctx.Done():
		return true
	case <-ticker.C:
		r.maybeSendDummyPacket()
	}
	return false
}

// maybeSendDummyPacket sends a dummy packet if conditions are met.
func (r *Relay) maybeSendDummyPacket() {
	if r.enabled.Load() && !r.shutdown.Load() {
		r.sendDummyPacket()
	}
}

// sendDummyPacket sends a dummy packet to a random peer.
func (r *Relay) sendDummyPacket() {
	relays := r.beacon.ListRelays()
	if len(relays) == 0 {
		return
	}

	// Select random relay.
	var randomBytes [1]byte
	rand.Read(randomBytes[:])
	idx := int(randomBytes[0]) % len(relays)
	target := relays[idx]

	// Generate dummy packet (fixed size, random content).
	dummy := make([]byte, FixedPacketSize)
	rand.Read(dummy)

	// Mark as dummy packet (first byte = 0x00).
	dummy[0] = 0x00

	if err := r.sender(target.PeerID, dummy); err == nil {
		atomic.AddUint64(&r.stats.DummyPacketsSent, 1)
	}
}

// randomDelay returns a random delay using exponential distribution.
// Per ROADMAP: random delay (exponential distribution, mean 200ms).
// Exponential distribution is ideal for mix networks as it provides:
// 1. Memoryless property - no timing correlation between packets
// 2. Heavy right tail - occasional long delays disrupt traffic analysis
func (r *Relay) randomDelay() time.Duration {
	// Generate uniform random value in [0, 1).
	var randomBytes [8]byte
	rand.Read(randomBytes[:])

	// Convert to uniform [0, 1).
	u := float64(uint64(randomBytes[0])<<56|
		uint64(randomBytes[1])<<48|
		uint64(randomBytes[2])<<40|
		uint64(randomBytes[3])<<32|
		uint64(randomBytes[4])<<24|
		uint64(randomBytes[5])<<16|
		uint64(randomBytes[6])<<8|
		uint64(randomBytes[7])) / float64(1<<64)

	// Avoid log(0) which is -infinity.
	if u == 0 {
		u = 1e-10
	}

	// Exponential distribution: -mean * ln(u).
	delay := -float64(MixDelayMean) * math.Log(u)

	// Clamp to [MinMixDelay, MaxMixDelay].
	if delay < float64(MinMixDelay) {
		delay = float64(MinMixDelay)
	}
	if delay > float64(MaxMixDelay) {
		delay = float64(MaxMixDelay)
	}

	return time.Duration(delay)
}

// RandomExponentialDelay generates an exponential random delay with the given mean.
// This is exported for use by other components needing mix network delays.
func RandomExponentialDelay(mean time.Duration, min, max time.Duration) time.Duration {
	var randomBytes [8]byte
	rand.Read(randomBytes[:])

	u := float64(uint64(randomBytes[0])<<56|
		uint64(randomBytes[1])<<48|
		uint64(randomBytes[2])<<40|
		uint64(randomBytes[3])<<32|
		uint64(randomBytes[4])<<24|
		uint64(randomBytes[5])<<16|
		uint64(randomBytes[6])<<8|
		uint64(randomBytes[7])) / float64(1<<64)

	if u == 0 {
		u = 1e-10
	}

	delay := -float64(mean) * math.Log(u)

	if delay < float64(min) {
		delay = float64(min)
	}
	if delay > float64(max) {
		delay = float64(max)
	}

	return time.Duration(delay)
}

// Stats returns current relay statistics.
func (r *Relay) Stats() RelayStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return RelayStats{
		PacketsForwarded: atomic.LoadUint64(&r.stats.PacketsForwarded),
		PacketsDropped:   atomic.LoadUint64(&r.stats.PacketsDropped),
		DummyPacketsSent: atomic.LoadUint64(&r.stats.DummyPacketsSent),
		BytesForwarded:   atomic.LoadUint64(&r.stats.BytesForwarded),
		AvgDelayMs:       r.stats.AvgDelayMs,
	}
}

// IsDummyPacket returns true if the packet is a dummy padding packet.
func IsDummyPacket(data []byte) bool {
	return len(data) == FixedPacketSize && data[0] == 0x00
}

// ShroudNode is a Shroud relay for Fortress-mode users.
// Per SHADOW_GRADIENT.md, Fortress-mode users contribute to the Shroud Network
// by serving as relay nodes, strengthening anonymity for all users.
type ShroudNode struct {
	mu     sync.RWMutex
	beacon *Beacon
	relay  *Relay

	// Node configuration.
	config ShroudNodeConfig

	// State.
	running   atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time

	// Capacity and metrics tracking.
	currentBandwidth  uint64 // Current bandwidth usage in bytes/sec.
	circuitCount      uint32 // Number of active circuits through this node.
	totalBytesRelayed uint64 // Total bytes relayed lifetime.
	version           string // Node version string.
}

// ShroudNodeConfig configures a Shroud relay node.
type ShroudNodeConfig struct {
	// PeerID is this node's libp2p peer ID.
	PeerID string

	// MaxBandwidth is the maximum bandwidth to dedicate (bytes/sec).
	// Default: 1 MB/sec (8 Mbps).
	MaxBandwidth uint64

	// MaxCircuits is the maximum number of concurrent circuits.
	// Default: 100.
	MaxCircuits uint32

	// AdvertiseInterval is how often to advertise as a relay.
	// Default: BeaconInterval (5 minutes).
	AdvertiseInterval time.Duration

	// EnableMixing enables traffic mixing delays.
	// Default: true.
	EnableMixing bool

	// EnableDummyTraffic enables cover traffic generation.
	// Default: true.
	EnableDummyTraffic bool
}

// DefaultShroudNodeConfig returns the default configuration for Fortress-mode.
func DefaultShroudNodeConfig() ShroudNodeConfig {
	return ShroudNodeConfig{
		MaxBandwidth:       1_000_000,   // 1 MB/sec
		MaxCircuits:        100,         // 100 concurrent circuits
		AdvertiseInterval:  BeaconInterval,
		EnableMixing:       true,
		EnableDummyTraffic: true,
	}
}

// NewShroudNode creates a new Shroud relay node for Fortress-mode operation.
// The handler processes incoming packets and the sender transmits to peers.
func NewShroudNode(beacon *Beacon, handler PacketHandler, sender PacketSender, config ShroudNodeConfig) *ShroudNode {
	ctx, cancel := context.WithCancel(context.Background())

	return &ShroudNode{
		beacon: beacon,
		relay:  NewRelay(beacon, handler, sender),
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins Shroud Node operation.
// This enables the node as a relay, starts packet processing, and begins
// advertising availability to other nodes.
func (n *ShroudNode) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.running.Load() {
		n.mu.Unlock()
		return errors.New("shroud node already running")
	}

	// Enable relay on the beacon.
	n.beacon.EnableRelay(n.config.PeerID, n.config.MaxBandwidth)

	// Enable and start the relay.
	n.relay.Enable()
	n.relay.Start(ctx)

	n.running.Store(true)
	n.startTime = time.Now()
	n.mu.Unlock()

	// Start advertisement loop.
	if n.config.AdvertiseInterval > 0 {
		go n.advertiseLoop(ctx)
	}

	return nil
}

// Stop shuts down the Shroud Node.
func (n *ShroudNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.running.Load() {
		return
	}

	n.running.Store(false)
	n.relay.Stop()
	n.relay.Disable()
	n.beacon.DisableRelay()
	n.cancel()
}

// IsRunning returns true if the Shroud Node is operational.
func (n *ShroudNode) IsRunning() bool {
	return n.running.Load()
}

// advertiseLoop periodically advertises this node as an available relay.
func (n *ShroudNode) advertiseLoop(ctx context.Context) {
	ticker := time.NewTicker(n.config.AdvertiseInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			// Would publish beacon advertisement here.
			// For now, the beacon already tracks relay status.
		}
	}
}

// Forward processes an incoming relay packet.
// This is the main entry point for circuit traffic.
func (n *ShroudNode) Forward(packet []byte) error {
	if !n.running.Load() {
		return ErrRelayNotEnabled
	}

	// Check bandwidth limits.
	if n.config.MaxBandwidth > 0 {
		current := atomic.LoadUint64(&n.currentBandwidth)
		if current+uint64(len(packet)) > n.config.MaxBandwidth {
			return ErrBufferFull
		}
	}

	return n.relay.Forward(packet)
}

// Stats returns current Shroud Node statistics.
func (n *ShroudNode) Stats() ShroudNodeStats {
	relayStats := n.relay.Stats()

	return ShroudNodeStats{
		RelayStats:       relayStats,
		IsRunning:        n.running.Load(),
		CurrentBandwidth: atomic.LoadUint64(&n.currentBandwidth),
		CircuitCount:     atomic.LoadUint32(&n.circuitCount),
		Config:           n.config,
	}
}

// ShroudNodeStats contains comprehensive Shroud Node metrics.
type ShroudNodeStats struct {
	RelayStats       RelayStats
	IsRunning        bool
	CurrentBandwidth uint64
	CircuitCount     uint32
	Config           ShroudNodeConfig
}

// Beacon returns the node's beacon.
func (n *ShroudNode) Beacon() *Beacon {
	return n.beacon
}

// Relay returns the underlying relay.
func (n *ShroudNode) Relay() *Relay {
	return n.relay
}

// UpdateBandwidth records current bandwidth usage.
// This should be called periodically by the network layer.
func (n *ShroudNode) UpdateBandwidth(bytesPerSec uint64) {
	atomic.StoreUint64(&n.currentBandwidth, bytesPerSec)
}

// IncrementCircuits increments the active circuit count.
func (n *ShroudNode) IncrementCircuits() bool {
	if n.config.MaxCircuits > 0 {
		current := atomic.LoadUint32(&n.circuitCount)
		if current >= n.config.MaxCircuits {
			return false
		}
	}
	atomic.AddUint32(&n.circuitCount, 1)
	return true
}

// DecrementCircuits decrements the active circuit count.
func (n *ShroudNode) DecrementCircuits() {
	for {
		current := atomic.LoadUint32(&n.circuitCount)
		if current == 0 {
			return
		}
		if atomic.CompareAndSwapUint32(&n.circuitCount, current, current-1) {
			return
		}
	}
}

// PeerID returns the node's peer ID.
func (n *ShroudNode) PeerID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.config.PeerID
}

// AvailableBandwidth returns the remaining available bandwidth.
func (n *ShroudNode) AvailableBandwidth() uint64 {
	if n.config.MaxBandwidth == 0 {
		return ^uint64(0) // Unlimited.
	}
	current := atomic.LoadUint64(&n.currentBandwidth)
	if current >= n.config.MaxBandwidth {
		return 0
	}
	return n.config.MaxBandwidth - current
}

// AvailableCircuits returns the remaining circuit capacity.
func (n *ShroudNode) AvailableCircuits() uint32 {
	if n.config.MaxCircuits == 0 {
		return ^uint32(0) // Unlimited.
	}
	current := atomic.LoadUint32(&n.circuitCount)
	if current >= n.config.MaxCircuits {
		return 0
	}
	return n.config.MaxCircuits - current
}

// CapacityMetrics returns current capacity metrics for advertisement.
// Per ROADMAP, Shroud Node capacity metrics advertisement allows
// circuit builders to select optimal relays based on available capacity.
func (n *ShroudNode) CapacityMetrics() CapacityMetrics {
	n.mu.RLock()
	startTime := n.startTime
	n.mu.RUnlock()

	var uptime time.Duration
	if !startTime.IsZero() {
		uptime = time.Since(startTime)
	}

	return CapacityMetrics{
		MaxBandwidth:      n.config.MaxBandwidth,
		CurrentBandwidth:  atomic.LoadUint64(&n.currentBandwidth),
		MaxCircuits:       n.config.MaxCircuits,
		CurrentCircuits:   atomic.LoadUint32(&n.circuitCount),
		UptimeSeconds:     uint64(uptime.Seconds()),
		TotalBytesRelayed: atomic.LoadUint64(&n.totalBytesRelayed),
		Version:           n.version,
	}
}

// CapacityMetrics contains capacity information for relay selection.
type CapacityMetrics struct {
	MaxBandwidth      uint64
	CurrentBandwidth  uint64
	MaxCircuits       uint32
	CurrentCircuits   uint32
	UptimeSeconds     uint64
	AvgLatencyMs      uint32
	TotalBytesRelayed uint64
	Version           string
}

// AvailableBandwidthRatio returns the available bandwidth as a ratio (0.0-1.0).
func (m CapacityMetrics) AvailableBandwidthRatio() float64 {
	if m.MaxBandwidth == 0 {
		return 1.0 // Unlimited.
	}
	if m.CurrentBandwidth >= m.MaxBandwidth {
		return 0.0
	}
	return float64(m.MaxBandwidth-m.CurrentBandwidth) / float64(m.MaxBandwidth)
}

// AvailableCircuitsRatio returns the available circuits as a ratio (0.0-1.0).
func (m CapacityMetrics) AvailableCircuitsRatio() float64 {
	if m.MaxCircuits == 0 {
		return 1.0 // Unlimited.
	}
	if m.CurrentCircuits >= m.MaxCircuits {
		return 0.0
	}
	return float64(m.MaxCircuits-m.CurrentCircuits) / float64(m.MaxCircuits)
}

// LoadScore returns a composite load score (0.0 = fully loaded, 1.0 = fully available).
// Higher scores indicate more available capacity.
func (m CapacityMetrics) LoadScore() float64 {
	bwRatio := m.AvailableBandwidthRatio()
	circuitRatio := m.AvailableCircuitsRatio()
	// Weight bandwidth and circuits equally.
	return (bwRatio + circuitRatio) / 2.0
}

// SetVersion sets the node version string for compatibility advertisement.
func (n *ShroudNode) SetVersion(version string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.version = version
}

// AddBytesRelayed increments the total bytes relayed counter.
func (n *ShroudNode) AddBytesRelayed(bytes uint64) {
	atomic.AddUint64(&n.totalBytesRelayed, bytes)
}

// TotalBytesRelayed returns the total bytes relayed by this node.
func (n *ShroudNode) TotalBytesRelayed() uint64 {
	return atomic.LoadUint64(&n.totalBytesRelayed)
}

// Uptime returns how long this node has been running.
func (n *ShroudNode) Uptime() time.Duration {
	n.mu.RLock()
	startTime := n.startTime
	n.mu.RUnlock()

	if startTime.IsZero() {
		return 0
	}
	return time.Since(startTime)
}

// SelectRelayByCapacity selects the best relay from a list based on capacity.
// This is a helper for circuit builders to choose relays optimally.
func SelectRelayByCapacity(relays []*RelayInfo, metrics map[string]CapacityMetrics, exclude []string) *RelayInfo {
	if len(relays) == 0 {
		return nil
	}

	// Build exclude map.
	excludeMap := make(map[string]bool)
	for _, id := range exclude {
		excludeMap[id] = true
	}

	var bestRelay *RelayInfo
	var bestScore float64 = -1

	for _, relay := range relays {
		if excludeMap[relay.PeerID] {
			continue
		}

		// Get capacity metrics if available.
		m, ok := metrics[relay.PeerID]
		if !ok {
			// No metrics, use default moderate score.
			if bestRelay == nil || 0.5 > bestScore {
				bestRelay = relay
				bestScore = 0.5
			}
			continue
		}

		score := m.LoadScore()

		// Boost score for nodes with longer uptime (more reliable).
		if m.UptimeSeconds > 3600 { // > 1 hour
			score *= 1.1
		}
		if m.UptimeSeconds > 86400 { // > 1 day
			score *= 1.1
		}

		// Clamp to 1.0 max.
		if score > 1.0 {
			score = 1.0
		}

		if score > bestScore {
			bestRelay = relay
			bestScore = score
		}
	}

	return bestRelay
}
