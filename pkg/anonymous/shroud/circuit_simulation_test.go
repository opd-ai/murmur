//go:build simulation

// Package shroud simulation tests verify traffic analysis resistance.
// Per ROADMAP.md Priority 6 Validation: "Anonymous Wave cannot be correlated
// to origin by passive observer in 100-node simulation".
package shroud

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// SimNode represents a node in the simulation network.
type SimNode struct {
	ID          string
	Beacon      *Beacon
	Relay       *Relay
	Manager     *CircuitManager
	directPeers []string // Simulated direct mesh neighbors
	sentPackets []SimPacket
	recvPackets []SimPacket
	mu          sync.Mutex
}

// SimPacket tracks packet metadata for traffic analysis.
type SimPacket struct {
	SourceID   string
	DestID     string
	Size       int
	Timestamp  time.Time
	OrigWaveID string // The original Wave identifier (known only to origin)
	IsDummy    bool
	HopCount   int
}

// SimNetwork manages the simulation network.
type SimNetwork struct {
	nodes       map[string]*SimNode
	mu          sync.RWMutex
	packetLog   []SimPacket       // Global packet log (passive observer view)
	waveOrigins map[string]string // waveID -> originNodeID (ground truth, hidden from observer)
}

// NewSimNetwork creates a new simulation network.
func NewSimNetwork(nodeCount int) (*SimNetwork, error) {
	net := &SimNetwork{
		nodes:       make(map[string]*SimNode),
		packetLog:   make([]SimPacket, 0),
		waveOrigins: make(map[string]string),
	}

	// Create nodes.
	for i := 0; i < nodeCount; i++ {
		node, err := net.createNode(fmt.Sprintf("node-%d", i))
		if err != nil {
			return nil, err
		}
		net.nodes[node.ID] = node
	}

	// Build mesh: each node gets 6-12 random direct peers per NETWORK_ARCHITECTURE.md.
	for _, node := range net.nodes {
		peerCount := 6 + (hashByte(node.ID) % 7) // 6-12 peers
		node.directPeers = net.selectRandomPeers(node.ID, peerCount)
	}

	// Register all nodes as potential relays.
	for _, node := range net.nodes {
		for _, otherNode := range net.nodes {
			if otherNode.ID != node.ID {
				node.Beacon.AddRelay(&RelayInfo{
					PeerID:    otherNode.ID,
					PublicKey: otherNode.Beacon.PublicKey(),
					Bandwidth: 1000000,
				})
			}
		}
	}

	return net, nil
}

// createNode creates a simulation node.
func (net *SimNetwork) createNode(id string) (*SimNode, error) {
	beacon, err := NewBeacon()
	if err != nil {
		return nil, err
	}

	beacon.EnableRelay(id, 1000000)

	node := &SimNode{
		ID:          id,
		Beacon:      beacon,
		directPeers: make([]string, 0),
		sentPackets: make([]SimPacket, 0),
		recvPackets: make([]SimPacket, 0),
	}

	// Create relay with simulated packet handling.
	handler := func(packet []byte) (string, []byte, error) {
		// Simplified: forward to a random relay.
		relays := beacon.ListRelays()
		if len(relays) == 0 {
			return "", nil, ErrRelayNotFound
		}
		var randByte [1]byte
		rand.Read(randByte[:])
		return relays[int(randByte[0])%len(relays)].PeerID, packet, nil
	}

	sender := func(peerID string, data []byte) error {
		net.logPacket(id, peerID, data, "", 0)
		return nil
	}

	node.Relay = NewRelay(beacon, handler, sender)
	node.Relay.Enable()

	return node, nil
}

// selectRandomPeers selects n random peers excluding selfID.
func (net *SimNetwork) selectRandomPeers(selfID string, n int) []string {
	peers := make([]string, 0, n)
	for id := range net.nodes {
		if id != selfID && len(peers) < n {
			peers = append(peers, id)
		}
	}
	return peers
}

// logPacket records a packet in the global log (passive observer view).
// Note: The passive observer can only see src/dst/size/time - NOT the waveID.
// We track waveID internally for verification but the attacker doesn't have it.
func (net *SimNetwork) logPacket(srcID, dstID string, data []byte, waveID string, hopCount int) {
	net.mu.Lock()
	defer net.mu.Unlock()

	pkt := SimPacket{
		SourceID:   srcID,
		DestID:     dstID,
		Size:       len(data),
		Timestamp:  time.Now(),
		OrigWaveID: waveID, // Hidden from attacker - only for verification
		IsDummy:    IsDummyPacket(data),
		HopCount:   hopCount,
	}
	net.packetLog = append(net.packetLog, pkt)
}

// sendAnonymousWave simulates sending an anonymous Wave through Shroud.
// This properly simulates the three-hop relay chain where each relay only
// sees the previous hop as source, not the true origin.
func (net *SimNetwork) sendAnonymousWave(originID, waveID string, payload []byte) error {
	net.mu.Lock()
	net.waveOrigins[waveID] = originID
	net.mu.Unlock()

	node, ok := net.nodes[originID]
	if !ok {
		return fmt.Errorf("node not found: %s", originID)
	}

	// Build circuit (3 hops).
	relays, err := node.Beacon.SelectRelays(node.directPeers)
	if err != nil {
		return err
	}

	circuit, err := node.Beacon.BuildCircuit(relays)
	if err != nil {
		return err
	}
	defer circuit.Close()

	// Encrypt payload with onion layers.
	encrypted, err := circuit.Encrypt(payload)
	if err != nil {
		return err
	}

	// Simulate the relay chain properly:
	// Hop 0: origin -> relay[0] (passive observer sees origin as source)
	// Hop 1: relay[0] -> relay[1] (passive observer sees relay[0] as source)
	// Hop 2: relay[1] -> relay[2] (passive observer sees relay[1] as source)
	// Each hop adds random delay per SHADOW_GRADIENT.md traffic mixing.

	// Add small random jitter to simulate real network conditions.
	var jitterBytes [1]byte
	rand.Read(jitterBytes[:])
	baseJitter := time.Duration(jitterBytes[0]) * time.Microsecond * 50

	// Log hop 0: origin -> first relay.
	// The passive observer can see this.
	time.Sleep(baseJitter)
	net.logPacket(originID, relays[0].PeerID, encrypted, waveID, 0)

	// Simulate relay forwarding with traffic mixing delays.
	// Hop 1: first relay -> second relay.
	rand.Read(jitterBytes[:])
	mixDelay1 := MinMixDelay + time.Duration(jitterBytes[0])*time.Microsecond*500
	time.Sleep(mixDelay1)
	net.logPacket(relays[0].PeerID, relays[1].PeerID, encrypted, waveID, 1)

	// Hop 2: second relay -> third relay.
	rand.Read(jitterBytes[:])
	mixDelay2 := MinMixDelay + time.Duration(jitterBytes[0])*time.Microsecond*500
	time.Sleep(mixDelay2)
	net.logPacket(relays[1].PeerID, relays[2].PeerID, encrypted, waveID, 2)

	return nil
}

// TrafficAnalysisResult holds analysis metrics.
type TrafficAnalysisResult struct {
	TotalPackets       int
	RealPackets        int
	DummyPackets       int
	UniqueOrigins      int
	CorrectGuesses     int
	RandomGuessRate    float64
	ActualGuessRate    float64
	AnalysisResistance float64 // 1.0 = perfect, 0.0 = no resistance
}

// analyzeTraffic performs passive traffic analysis on the packet log.
// A passive observer can see: source/dest of each hop, timing, packet size.
// They cannot see: encrypted payload content, the original Wave origin.
func (net *SimNetwork) analyzeTraffic() TrafficAnalysisResult {
	net.mu.RLock()
	defer net.mu.RUnlock()

	result := TrafficAnalysisResult{
		TotalPackets:  len(net.packetLog),
		UniqueOrigins: len(net.waveOrigins),
	}

	// Count real vs dummy packets.
	for _, pkt := range net.packetLog {
		if pkt.IsDummy {
			result.DummyPackets++
		} else {
			result.RealPackets++
		}
	}

	// Attempt timing correlation attack.
	// For each Wave, try to identify the origin by timing patterns.
	correctGuesses := 0

	for waveID, trueOrigin := range net.waveOrigins {
		guessedOrigin := net.attemptTimingCorrelation(waveID)
		if guessedOrigin == trueOrigin {
			correctGuesses++
		}
	}

	result.CorrectGuesses = correctGuesses

	// Random guess rate = 1/nodeCount.
	nodeCount := float64(len(net.nodes))
	result.RandomGuessRate = 1.0 / nodeCount

	// Actual guess rate from timing analysis.
	if result.UniqueOrigins > 0 {
		result.ActualGuessRate = float64(correctGuesses) / float64(result.UniqueOrigins)
	}

	// Analysis resistance: how much better than random is the attacker?
	// If attacker does no better than random, resistance = 1.0.
	// If attacker achieves 100% correlation, resistance = 0.0.
	if result.ActualGuessRate <= result.RandomGuessRate {
		result.AnalysisResistance = 1.0
	} else {
		// Normalize: (1 - actualRate) / (1 - randomRate)
		result.AnalysisResistance = (1.0 - result.ActualGuessRate) / (1.0 - result.RandomGuessRate)
	}

	return result
}

// attemptTimingCorrelation simulates a passive attacker trying to
// correlate a Wave to its origin using only timing information.
// The attacker can observe all packets but doesn't know which ones
// belong to which Wave (since all packets are encrypted and same size).
func (net *SimNetwork) attemptTimingCorrelation(waveID string) string {
	// Important: The attacker does NOT have access to waveID in real life.
	// They only see packet streams and must try to correlate by timing/size.
	// We use waveID here only to evaluate their success rate.

	// Realistic attack: find all packets within a time window and try to
	// identify chains. This is hard because:
	// 1. All packets look identical (same size due to padding)
	// 2. Dummy traffic adds noise
	// 3. Mix delays break timing correlation
	// 4. Multiple Waves are being sent simultaneously

	// Find packets in the time window around this Wave.
	var wavePackets []SimPacket
	var firstWaveTime time.Time
	for _, pkt := range net.packetLog {
		if pkt.OrigWaveID == waveID && pkt.HopCount == 0 {
			firstWaveTime = pkt.Timestamp
			break
		}
	}

	if firstWaveTime.IsZero() {
		return randomNodeID(net.nodes)
	}

	// Attacker looks at packets in a window around the suspected time.
	// They see many packets from many sources due to concurrent activity.
	windowStart := firstWaveTime.Add(-200 * time.Millisecond)
	windowEnd := firstWaveTime.Add(500 * time.Millisecond)

	for _, pkt := range net.packetLog {
		if pkt.Timestamp.After(windowStart) && pkt.Timestamp.Before(windowEnd) {
			if !pkt.IsDummy { // Attacker can't distinguish, but we filter for analysis
				wavePackets = append(wavePackets, pkt)
			}
		}
	}

	if len(wavePackets) == 0 {
		return randomNodeID(net.nodes)
	}

	// Count unique sources in the window - attacker must guess among these.
	sourceCounts := make(map[string]int)
	for _, pkt := range wavePackets {
		sourceCounts[pkt.SourceID]++
	}

	// Attacker's strategy: guess the most frequent source in the window.
	// This is a reasonable heuristic but should fail due to mixing.
	var bestGuess string
	var maxCount int
	for src, count := range sourceCounts {
		if count > maxCount {
			maxCount = count
			bestGuess = src
		}
	}

	if bestGuess != "" {
		return bestGuess
	}

	return randomNodeID(net.nodes)
}

// randomNodeID returns a random node ID (for random guessing).
func randomNodeID(nodes map[string]*SimNode) string {
	var randByte [1]byte
	rand.Read(randByte[:])
	i := int(randByte[0]) % len(nodes)
	j := 0
	for id := range nodes {
		if j == i {
			return id
		}
		j++
	}
	return ""
}

// hashByte returns a deterministic byte from a string.
func hashByte(s string) int {
	h := 0
	for _, c := range s {
		h = (h*31 + int(c)) % 256
	}
	return h
}

// TestShroudTrafficAnalysisResistance is the main 100-node simulation test.
// Per ROADMAP.md Priority 6: "Anonymous Wave cannot be correlated to origin
// by passive observer in 100-node simulation".
func TestShroudTrafficAnalysisResistance(t *testing.T) {
	const nodeCount = 100
	const waveCount = 50

	t.Logf("Creating %d-node simulation network...", nodeCount)
	net, err := NewSimNetwork(nodeCount)
	if err != nil {
		t.Fatalf("Failed to create simulation network: %v", err)
	}

	// Start relay processing on all nodes.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, node := range net.nodes {
		node.Relay.Start(ctx)
	}

	// Let dummy traffic flow for warm-up.
	t.Log("Warming up with dummy traffic...")
	time.Sleep(500 * time.Millisecond)

	// Send anonymous Waves from random origins.
	t.Logf("Sending %d anonymous Waves...", waveCount)
	nodeIDs := make([]string, 0, len(net.nodes))
	for id := range net.nodes {
		nodeIDs = append(nodeIDs, id)
	}

	var wg sync.WaitGroup
	var sendErrors int64

	for i := 0; i < waveCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Random origin node.
			var randBytes [2]byte
			rand.Read(randBytes[:])
			originIdx := int(randBytes[0]) % len(nodeIDs)
			originID := nodeIDs[originIdx]

			waveID := fmt.Sprintf("wave-%d", i)
			payload := make([]byte, 100)
			rand.Read(payload)

			if err := net.sendAnonymousWave(originID, waveID, payload); err != nil {
				atomic.AddInt64(&sendErrors, 1)
			}

			// Random inter-Wave delay to simulate real usage.
			delay := time.Duration(randBytes[1]) * time.Microsecond * 100
			time.Sleep(delay)
		}(i)
	}

	wg.Wait()

	// Let final traffic settle.
	time.Sleep(500 * time.Millisecond)

	// Analyze traffic for correlation attacks.
	t.Log("Analyzing traffic patterns...")
	result := net.analyzeTraffic()

	t.Logf("Traffic Analysis Results:")
	t.Logf("  Total packets logged: %d", result.TotalPackets)
	t.Logf("  Real packets: %d", result.RealPackets)
	t.Logf("  Dummy packets: %d", result.DummyPackets)
	t.Logf("  Unique Wave origins: %d", result.UniqueOrigins)
	t.Logf("  Correct timing guesses: %d", result.CorrectGuesses)
	t.Logf("  Random guess rate: %.2f%%", result.RandomGuessRate*100)
	t.Logf("  Actual guess rate: %.2f%%", result.ActualGuessRate*100)
	t.Logf("  Analysis resistance: %.2f%%", result.AnalysisResistance*100)

	// Validation criteria:
	// 1. Attacker should not do significantly better than random guessing.
	//    We allow up to 5x random rate to account for statistical variance
	//    in smaller sample sizes. With 50 waves and 100 nodes, random would
	//    expect ~0.5 correct, so 2-3 correct is within statistical noise.
	maxAllowedGuessRate := result.RandomGuessRate * 5.0
	if result.ActualGuessRate > maxAllowedGuessRate {
		t.Errorf("Traffic analysis attack too successful: %.2f%% > %.2f%% (5x random)",
			result.ActualGuessRate*100, maxAllowedGuessRate*100)
	}

	// 2. Analysis resistance should be high (>0.9 = 90% of theoretical maximum).
	//    This is stricter than the guess rate check and ensures overall resistance.
	minResistance := 0.90
	if result.AnalysisResistance < minResistance {
		t.Errorf("Traffic analysis resistance too low: %.2f%% < %.2f%%",
			result.AnalysisResistance*100, minResistance*100)
	}

	// 3. Send errors should be minimal (<10%).
	maxSendErrors := int64(waveCount / 10)
	if sendErrors > maxSendErrors {
		t.Errorf("Too many send errors: %d > %d", sendErrors, maxSendErrors)
	}

	t.Log("✓ Anonymous Waves cannot be correlated to origin by passive observer")
}

// TestShroudDummyTrafficDistribution verifies dummy traffic provides cover.
func TestShroudDummyTrafficDistribution(t *testing.T) {
	const nodeCount = 100

	net, err := NewSimNetwork(nodeCount)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, node := range net.nodes {
		node.Relay.Start(ctx)
	}

	// Let dummy traffic accumulate.
	time.Sleep(3 * time.Second)

	net.mu.RLock()
	dummyCount := 0
	for _, pkt := range net.packetLog {
		if pkt.IsDummy {
			dummyCount++
		}
	}
	totalPackets := len(net.packetLog)
	net.mu.RUnlock()

	// Dummy traffic should be present.
	if dummyCount == 0 {
		t.Error("No dummy traffic observed")
	}

	t.Logf("Dummy traffic: %d/%d packets (%.1f%%)",
		dummyCount, totalPackets, float64(dummyCount)/float64(totalPackets)*100)
}

// TestShroudCircuitDiversity verifies circuit hop diversity.
func TestShroudCircuitDiversity(t *testing.T) {
	const nodeCount = 100
	const circuitCount = 1000

	net, err := NewSimNetwork(nodeCount)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	// Track hop selection frequency.
	hopCounts := make(map[string]int)
	var mu sync.Mutex

	for _, node := range net.nodes {
		for i := 0; i < circuitCount/nodeCount; i++ {
			relays, err := node.Beacon.SelectRelays(node.directPeers)
			if err != nil {
				continue
			}

			mu.Lock()
			for _, relay := range relays {
				hopCounts[relay.PeerID]++
			}
			mu.Unlock()
		}
	}

	// Calculate standard deviation of hop selection.
	var total float64
	for _, count := range hopCounts {
		total += float64(count)
	}
	mean := total / float64(len(hopCounts))

	var variance float64
	for _, count := range hopCounts {
		diff := float64(count) - mean
		variance += diff * diff
	}
	variance /= float64(len(hopCounts))
	stddev := math.Sqrt(variance)
	cv := stddev / mean // Coefficient of variation

	t.Logf("Hop selection: mean=%.1f, stddev=%.1f, CV=%.2f", mean, stddev, cv)

	// Coefficient of variation should be low (<0.5 = reasonably uniform).
	if cv > 0.5 {
		t.Errorf("Hop selection too uneven: CV=%.2f (should be <0.5)", cv)
	}
}

// TestShroudPacketSizeUniformity verifies all packets have uniform size.
func TestShroudPacketSizeUniformity(t *testing.T) {
	const nodeCount = 20
	const waveCount = 50

	net, err := NewSimNetwork(nodeCount)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, node := range net.nodes {
		node.Relay.Start(ctx)
	}

	// Send Waves with varying payload sizes.
	nodeIDs := make([]string, 0, len(net.nodes))
	for id := range net.nodes {
		nodeIDs = append(nodeIDs, id)
	}

	for i := 0; i < waveCount; i++ {
		payloadSize := 10 + (i * 20) // 10 to 1000 bytes
		if payloadSize > FixedPacketSize-100 {
			payloadSize = FixedPacketSize - 100
		}

		payload := make([]byte, payloadSize)
		rand.Read(payload)

		originID := nodeIDs[i%len(nodeIDs)]
		waveID := fmt.Sprintf("wave-%d", i)

		net.sendAnonymousWave(originID, waveID, payload)
	}

	time.Sleep(200 * time.Millisecond)

	// All encrypted packets should be the same size (FixedPacketSize + overhead).
	net.mu.RLock()
	defer net.mu.RUnlock()

	sizeCounts := make(map[int]int)
	for _, pkt := range net.packetLog {
		sizeCounts[pkt.Size]++
	}

	// Should have very few unique sizes (ideally 1-2 for encrypted + dummy).
	if len(sizeCounts) > 5 {
		t.Errorf("Too many packet sizes (%d) - breaks traffic analysis resistance", len(sizeCounts))
	}

	t.Logf("Unique packet sizes: %d", len(sizeCounts))
	for size, count := range sizeCounts {
		t.Logf("  Size %d: %d packets", size, count)
	}
}
