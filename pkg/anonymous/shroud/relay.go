// Package shroud provides three-hop onion circuit construction and relay forwarding.
// Per SECURITY_PRIVACY.md §Class 2, Shroud Nodes forward encrypted traffic with
// traffic mixing (random delay injection) and dummy traffic generation.
package shroud

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Relay configuration constants per SHADOW_GRADIENT.md.
const (
	// DummyPacketRate is the default rate for constant-rate padding (1 packet/sec).
	DummyPacketRate = 1 * time.Second

	// MinMixDelay is the minimum random delay added to forwarded packets.
	MinMixDelay = 50 * time.Millisecond

	// MaxMixDelay is the maximum random delay added to forwarded packets.
	MaxMixDelay = 200 * time.Millisecond

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
	if r.shutdown.Load() {
		return ErrRelayShutdown
	}

	if !r.enabled.Load() {
		return ErrRelayNotEnabled
	}

	// Process packet to determine next hop.
	nextHop, data, err := r.handler(packet)
	if err != nil {
		atomic.AddUint64(&r.stats.PacketsDropped, 1)
		return err
	}

	// Calculate random delay for traffic mixing.
	delay := r.randomDelay()

	rp := relayPacket{
		data:      data,
		nextHop:   nextHop,
		arriveAt:  time.Now(),
		scheduled: time.Now().Add(delay),
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
		select {
		case <-ctx.Done():
			return
		case <-r.ctx.Done():
			return
		case pkt := <-r.inbound:
			pending = append(pending, pkt)
		case <-ticker.C:
			pending = r.processPendingPackets(pending)
		}
	}
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
		select {
		case <-ctx.Done():
			return
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.maybeSendDummyPacket()
		}
	}
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

// randomDelay returns a random delay between MinMixDelay and MaxMixDelay.
func (r *Relay) randomDelay() time.Duration {
	var randomBytes [2]byte
	rand.Read(randomBytes[:])

	// Convert to range [0, MaxMixDelay - MinMixDelay].
	rangeMs := MaxMixDelay - MinMixDelay
	delayFraction := float64(uint16(randomBytes[0])<<8|uint16(randomBytes[1])) / 65535.0

	return MinMixDelay + time.Duration(float64(rangeMs)*delayFraction)
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
