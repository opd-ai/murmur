// Package shroud provides three-hop onion circuit construction.
// This file implements BeaconWave wire format encoding/decoding for relay discovery.
package shroud

import (
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type BeaconWave struct {
	// Version is the beacon wave protocol version.
	Version byte

	// Type is the wave type (BeaconWaveType).
	Type byte

	// RelayPeerID is the relay's libp2p peer ID.
	RelayPeerID string

	// PublicKey is the relay's Curve25519 public key.
	PublicKey [32]byte

	// Bandwidth is the advertised bandwidth capacity (bytes/sec).
	Bandwidth uint64

	// MaxCircuits is the maximum circuits this relay can handle.
	MaxCircuits uint32

	// CurrentLoad is the current circuit count.
	CurrentLoad uint32

	// Latency is the relay's estimated latency in milliseconds.
	LatencyMs uint32

	// Uptime is the relay's uptime in seconds.
	Uptime uint64

	// Timestamp is when this wave was created (Unix seconds).
	Timestamp int64

	// TTL is the time-to-live in seconds.
	TTL uint32

	// Signature signs (Version || Type || RelayPeerID || PublicKey || Bandwidth || MaxCircuits || CurrentLoad || LatencyMs || Uptime || Timestamp || TTL).
	Signature []byte
}

// IsExpired returns true if the beacon wave has exceeded its TTL.
func (b *BeaconWave) IsExpired() bool {
	expiresAt := time.Unix(b.Timestamp, 0).Add(time.Duration(b.TTL) * time.Second)
	return time.Now().After(expiresAt)
}

// LoadFactor returns the current load as a fraction of max capacity.
func (b *BeaconWave) LoadFactor() float64 {
	if b.MaxCircuits == 0 {
		return 1.0
	}
	return float64(b.CurrentLoad) / float64(b.MaxCircuits)
}

// ToRelayInfo converts a BeaconWave to RelayInfo for the registry.
func (b *BeaconWave) ToRelayInfo() *RelayInfo {
	return &RelayInfo{
		PeerID:    b.RelayPeerID,
		PublicKey: b.PublicKey,
		Bandwidth: b.Bandwidth,
		SeenAt:    time.Unix(b.Timestamp, 0),
	}
}

// EncodeBeaconWave encodes a BeaconWave for transmission.
func EncodeBeaconWave(wave *BeaconWave) ([]byte, error) {
	// Calculate size.
	peerIDLen := len(wave.RelayPeerID)
	sigLen := len(wave.Signature)

	// Format: Version(1) + Type(1) + PeerIDLen(2) + PeerID + PublicKey(32) +
	//         Bandwidth(8) + MaxCircuits(4) + CurrentLoad(4) + LatencyMs(4) +
	//         Uptime(8) + Timestamp(8) + TTL(4) + SigLen(2) + Signature
	headerSize := 1 + 1 + 2 + peerIDLen + 32 + 8 + 4 + 4 + 4 + 8 + 8 + 4 + 2 + sigLen

	buf := make([]byte, headerSize)
	offset := 0

	// Version and Type.
	buf[offset] = wave.Version
	offset++
	buf[offset] = wave.Type
	offset++

	// PeerID length and value.
	buf[offset] = byte(peerIDLen >> 8)
	buf[offset+1] = byte(peerIDLen & 0xFF)
	offset += 2
	copy(buf[offset:], wave.RelayPeerID)
	offset += peerIDLen

	// PublicKey.
	copy(buf[offset:], wave.PublicKey[:])
	offset += 32

	// Bandwidth (big-endian uint64).
	binary.BigEndian.PutUint64(buf[offset:], wave.Bandwidth)
	offset += 8

	// MaxCircuits (big-endian uint32).
	binary.BigEndian.PutUint32(buf[offset:], wave.MaxCircuits)
	offset += 4

	// CurrentLoad (big-endian uint32).
	binary.BigEndian.PutUint32(buf[offset:], wave.CurrentLoad)
	offset += 4

	// LatencyMs (big-endian uint32).
	binary.BigEndian.PutUint32(buf[offset:], wave.LatencyMs)
	offset += 4

	// Uptime (big-endian int64).
	binary.BigEndian.PutUint64(buf[offset:], uint64(wave.Uptime))
	offset += 8

	// Timestamp (big-endian int64).
	binary.BigEndian.PutUint64(buf[offset:], uint64(wave.Timestamp))
	offset += 8

	// TTL (big-endian uint32).
	binary.BigEndian.PutUint32(buf[offset:], wave.TTL)
	offset += 4

	// Signature length and value.
	buf[offset] = byte(sigLen >> 8)
	buf[offset+1] = byte(sigLen & 0xFF)
	offset += 2
	copy(buf[offset:], wave.Signature)

	return buf, nil
}

// DecodeBeaconWave decodes a BeaconWave from bytes.
func DecodeBeaconWave(data []byte) (*BeaconWave, error) {
	if len(data) < 4 {
		return nil, ErrBeaconWaveInvalid
	}

	reader := &beaconReader{data: data, offset: 0}
	wave := &BeaconWave{}

	if err := reader.decodeHeader(wave); err != nil {
		return nil, err
	}

	if err := reader.decodePeerID(wave); err != nil {
		return nil, err
	}

	if err := reader.decodePublicKey(wave); err != nil {
		return nil, err
	}

	if err := reader.decodeMetrics(wave); err != nil {
		return nil, err
	}

	if err := reader.decodeSignature(wave); err != nil {
		return nil, err
	}

	return wave, nil
}

// beaconReader provides sequential reading of BeaconWave fields.
type beaconReader struct {
	data   []byte
	offset int
}

// decodeHeader reads version and type fields.
func (r *beaconReader) decodeHeader(wave *BeaconWave) error {
	wave.Version = r.data[r.offset]
	r.offset++
	if wave.Version != BeaconWaveVersion {
		return ErrBeaconWaveBadVersion
	}

	wave.Type = r.data[r.offset]
	r.offset++
	if wave.Type != BeaconWaveType {
		return ErrBeaconWaveInvalid
	}
	return nil
}

// decodePeerID reads the variable-length peer ID.
func (r *beaconReader) decodePeerID(wave *BeaconWave) error {
	if len(r.data) < r.offset+2 {
		return ErrBeaconWaveInvalid
	}
	peerIDLen := int(r.data[r.offset])<<8 | int(r.data[r.offset+1])
	r.offset += 2

	if len(r.data) < r.offset+peerIDLen {
		return ErrBeaconWaveInvalid
	}
	wave.RelayPeerID = string(r.data[r.offset : r.offset+peerIDLen])
	r.offset += peerIDLen
	return nil
}

// decodePublicKey reads the 32-byte public key.
func (r *beaconReader) decodePublicKey(wave *BeaconWave) error {
	if len(r.data) < r.offset+32 {
		return ErrBeaconWaveInvalid
	}
	copy(wave.PublicKey[:], r.data[r.offset:r.offset+32])
	r.offset += 32
	return nil
}

// decodeMetrics reads bandwidth, circuit, and timing metrics.
func (r *beaconReader) decodeMetrics(wave *BeaconWave) error {
	var err error

	wave.Bandwidth, err = r.readUint64()
	if err != nil {
		return err
	}
	wave.MaxCircuits, err = r.readUint32()
	if err != nil {
		return err
	}
	wave.CurrentLoad, err = r.readUint32()
	if err != nil {
		return err
	}
	wave.LatencyMs, err = r.readUint32()
	if err != nil {
		return err
	}
	wave.Uptime, err = r.readUint64()
	if err != nil {
		return err
	}
	ts, err := r.readUint64()
	if err != nil {
		return err
	}
	wave.Timestamp = int64(ts)
	wave.TTL, err = r.readUint32()
	return err
}

// decodeSignature reads the variable-length signature.
func (r *beaconReader) decodeSignature(wave *BeaconWave) error {
	if len(r.data) < r.offset+2 {
		return ErrBeaconWaveInvalid
	}
	sigLen := int(r.data[r.offset])<<8 | int(r.data[r.offset+1])
	r.offset += 2

	if len(r.data) < r.offset+sigLen {
		return ErrBeaconWaveInvalid
	}
	wave.Signature = make([]byte, sigLen)
	copy(wave.Signature, r.data[r.offset:r.offset+sigLen])
	return nil
}

// readUint64 reads a big-endian 8-byte unsigned integer.
func (r *beaconReader) readUint64() (uint64, error) {
	if len(r.data) < r.offset+8 {
		return 0, ErrBeaconWaveInvalid
	}
	var val uint64
	for i := 0; i < 8; i++ {
		val = (val << 8) | uint64(r.data[r.offset+i])
	}
	r.offset += 8
	return val, nil
}

// readUint32 reads a big-endian 4-byte unsigned integer.
func (r *beaconReader) readUint32() (uint32, error) {
	if len(r.data) < r.offset+4 {
		return 0, ErrBeaconWaveInvalid
	}
	var val uint32
	for i := 0; i < 4; i++ {
		val = (val << 8) | uint32(r.data[r.offset+i])
	}
	r.offset += 4
	return val, nil
}

// BeaconWavePublisher handles publishing relay advertisements.
type BeaconWavePublisher struct {
	mu        sync.RWMutex
	beacon    *Beacon
	peerID    string
	interval  time.Duration
	publisher func(data []byte) error
	stop      chan struct{}
	running   atomic.Bool

	// Current metrics.
	maxCircuits uint32
	currentLoad atomic.Uint32
	latencyMs   uint32
	startTime   time.Time
}

// NewBeaconWavePublisher creates a new beacon wave publisher.
func NewBeaconWavePublisher(beacon *Beacon, peerID string, publisher func(data []byte) error) *BeaconWavePublisher {
	return &BeaconWavePublisher{
		beacon:    beacon,
		peerID:    peerID,
		interval:  BeaconWaveInterval,
		publisher: publisher,
		stop:      make(chan struct{}),
		startTime: time.Now(),
	}
}

// SetCapacity sets the relay's capacity metrics.
func (p *BeaconWavePublisher) SetCapacity(maxCircuits, latencyMs uint32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxCircuits = maxCircuits
	p.latencyMs = latencyMs
}

// SetCurrentLoad sets the current circuit count.
func (p *BeaconWavePublisher) SetCurrentLoad(load uint32) {
	p.currentLoad.Store(load)
}

// SetInterval sets the broadcast interval.
func (p *BeaconWavePublisher) SetInterval(interval time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interval = interval
}

// Start begins periodic beacon wave broadcasting.
func (p *BeaconWavePublisher) Start() {
	if p.running.Swap(true) {
		return // Already running.
	}

	go p.publishLoop()
}

// Stop stops beacon wave broadcasting.
func (p *BeaconWavePublisher) Stop() {
	if !p.running.Swap(false) {
		return // Not running.
	}

	close(p.stop)
}

// PublishNow immediately publishes a beacon wave.
func (p *BeaconWavePublisher) PublishNow() error {
	wave, err := p.createBeaconWave()
	if err != nil {
		return err
	}

	encoded, err := EncodeBeaconWave(wave)
	if err != nil {
		return err
	}

	return p.publisher(encoded)
}

func (p *BeaconWavePublisher) publishLoop() {
	p.mu.RLock()
	interval := p.interval
	p.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Publish immediately on start.
	if err := p.PublishNow(); err != nil {
		// Log error but continue.
	}

	for {
		select {
		case <-ticker.C:
			if p.beacon.IsRelay() {
				if err := p.PublishNow(); err != nil {
					// Log error but continue.
				}
			}
		case <-p.stop:
			return
		}
	}
}

func (p *BeaconWavePublisher) createBeaconWave() (*BeaconWave, error) {
	selfInfo := p.beacon.SelfInfo()
	if selfInfo == nil {
		return nil, errors.New("not configured as relay")
	}

	p.mu.RLock()
	maxCircuits := p.maxCircuits
	latencyMs := p.latencyMs
	p.mu.RUnlock()

	uptime := uint64(time.Since(p.startTime).Seconds())

	wave := &BeaconWave{
		Version:     BeaconWaveVersion,
		Type:        BeaconWaveType,
		RelayPeerID: p.peerID,
		PublicKey:   selfInfo.PublicKey,
		Bandwidth:   selfInfo.Bandwidth,
		MaxCircuits: maxCircuits,
		CurrentLoad: p.currentLoad.Load(),
		LatencyMs:   latencyMs,
		Uptime:      uptime,
		Timestamp:   time.Now().Unix(),
		TTL:         uint32(BeaconWaveTTL.Seconds()),
	}

	// Signature is optional (can be added if verification is needed).
	wave.Signature = nil

	return wave, nil
}

// BeaconWaveReceiver handles receiving and processing relay advertisements.
type BeaconWaveReceiver struct {
	mu       sync.RWMutex
	beacon   *Beacon
	selfID   string
	stats    BeaconWaveStats
	handlers []BeaconWaveHandler
}

// BeaconWaveHandler is called when a beacon wave is received.
type BeaconWaveHandler func(wave *BeaconWave) error

// BeaconWaveStats tracks beacon wave statistics.
type BeaconWaveStats struct {
	WavesReceived    uint64
	WavesProcessed   uint64
	WavesExpired     uint64
	RelaysDiscovered uint64
	RelaysUpdated    uint64
}

// NewBeaconWaveReceiver creates a new beacon wave receiver.
func NewBeaconWaveReceiver(beacon *Beacon, selfID string) *BeaconWaveReceiver {
	return &BeaconWaveReceiver{
		beacon: beacon,
		selfID: selfID,
	}
}

// HandleIncoming processes a received beacon wave.
func (r *BeaconWaveReceiver) HandleIncoming(data []byte) error {
	atomic.AddUint64(&r.stats.WavesReceived, 1)

	wave, err := DecodeBeaconWave(data)
	if err != nil {
		return err
	}

	// Ignore our own advertisements.
	if wave.RelayPeerID == r.selfID {
		return nil
	}

	// Check expiry.
	if wave.IsExpired() {
		atomic.AddUint64(&r.stats.WavesExpired, 1)
		return ErrBeaconWaveExpired
	}

	// Check if this is a new relay or an update.
	_, existed := r.beacon.GetRelay(wave.RelayPeerID)

	// Register the relay.
	r.beacon.AddRelay(wave.ToRelayInfo())

	if existed {
		atomic.AddUint64(&r.stats.RelaysUpdated, 1)
	} else {
		atomic.AddUint64(&r.stats.RelaysDiscovered, 1)
	}

	atomic.AddUint64(&r.stats.WavesProcessed, 1)

	// Call handlers.
	r.mu.RLock()
	handlers := r.handlers
	r.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(wave); err != nil {
			// Log but continue.
		}
	}

	return nil
}

// RegisterHandler registers a handler for beacon waves.
func (r *BeaconWaveReceiver) RegisterHandler(handler BeaconWaveHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, handler)
}

// Stats returns beacon wave statistics.
func (r *BeaconWaveReceiver) Stats() BeaconWaveStats {
	return BeaconWaveStats{
		WavesReceived:    atomic.LoadUint64(&r.stats.WavesReceived),
		WavesProcessed:   atomic.LoadUint64(&r.stats.WavesProcessed),
		WavesExpired:     atomic.LoadUint64(&r.stats.WavesExpired),
		RelaysDiscovered: atomic.LoadUint64(&r.stats.RelaysDiscovered),
		RelaysUpdated:    atomic.LoadUint64(&r.stats.RelaysUpdated),
	}
}

// RelayDiscovery orchestrates Shroud relay discovery via Beacon Waves.
type RelayDiscovery struct {
	beacon    *Beacon
	publisher *BeaconWavePublisher
	receiver  *BeaconWaveReceiver
}

// NewRelayDiscovery creates a new relay discovery system.
func NewRelayDiscovery(beacon *Beacon, peerID string, publisher func(data []byte) error) *RelayDiscovery {
	return &RelayDiscovery{
		beacon:    beacon,
		publisher: NewBeaconWavePublisher(beacon, peerID, publisher),
		receiver:  NewBeaconWaveReceiver(beacon, peerID),
	}
}

// Start begins relay discovery (publishing and receiving).
func (rd *RelayDiscovery) Start() {
	rd.publisher.Start()
}

// Stop stops relay discovery.
func (rd *RelayDiscovery) Stop() {
	rd.publisher.Stop()
}

// HandleBeaconWave processes an incoming beacon wave.
func (rd *RelayDiscovery) HandleBeaconWave(data []byte) error {
	return rd.receiver.HandleIncoming(data)
}

// SetCapacity configures this node's relay capacity.
func (rd *RelayDiscovery) SetCapacity(maxCircuits, latencyMs uint32) {
	rd.publisher.SetCapacity(maxCircuits, latencyMs)
}

// SetCurrentLoad updates the current circuit count.
func (rd *RelayDiscovery) SetCurrentLoad(load uint32) {
	rd.publisher.SetCurrentLoad(load)
}

// Publisher returns the beacon wave publisher.
func (rd *RelayDiscovery) Publisher() *BeaconWavePublisher {
	return rd.publisher
}

// Receiver returns the beacon wave receiver.
func (rd *RelayDiscovery) Receiver() *BeaconWaveReceiver {
	return rd.receiver
}

// Beacon returns the underlying beacon.
func (rd *RelayDiscovery) Beacon() *Beacon {
	return rd.beacon
}

// CleanupStaleRelays removes relays that haven't been seen recently.
func (rd *RelayDiscovery) CleanupStaleRelays(maxAge time.Duration) int {
	relays := rd.beacon.ListRelays()
	removed := 0
	cutoff := time.Now().Add(-maxAge)

	for _, relay := range relays {
		if relay.SeenAt.Before(cutoff) {
			rd.beacon.RemoveRelay(relay.PeerID)
			removed++
		}
	}

	return removed
}
