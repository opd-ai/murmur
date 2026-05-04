// Package discovery provides Peer Exchange (PEX) protocol implementation.
// Per NETWORK_ARCHITECTURE.md §3 (Peer Exchange Protocol):
// "The peer exchange protocol (`/murmur/peerex/1.0`) is a simple protocol where
// connected peers periodically share their known peer lists."

package discovery

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

// PEX protocol constants per NETWORK_ARCHITECTURE.md.
const (
	// PEXProtocolID is the protocol identifier for peer exchange.
	PEXProtocolID = protocol.ID("/murmur/peer-exchange/1")

	// PEXInterval is how often peer exchange is performed.
	// Per NETWORK_ARCHITECTURE.md: "Every 5 minutes, each node sends a random sample."
	PEXInterval = 5 * time.Minute

	// PEXSampleSize is the number of peers to share in each exchange.
	// Per NETWORK_ARCHITECTURE.md: "a random sample of 20 peers."
	PEXSampleSize = 20

	// PEXMaxAddrs is the maximum number of addresses to include per peer.
	PEXMaxAddrs = 10

	// PEXReadTimeout is the timeout for reading peer exchange data.
	PEXReadTimeout = 30 * time.Second

	// PEXWriteTimeout is the timeout for writing peer exchange data.
	PEXWriteTimeout = 30 * time.Second

	// PEXMaxMessageSize is the maximum size of a PEX message (64KB).
	PEXMaxMessageSize = 64 * 1024
)

// PeerInfo represents information about a peer for exchange.
type PeerInfo struct {
	ID    peer.ID
	Addrs []multiaddr.Multiaddr
}

// PEX manages the Peer Exchange protocol.
type PEX struct {
	h       host.Host
	handler PeerHandler
	mu      sync.RWMutex
	running bool
	cancel  context.CancelFunc
}

// NewPEX creates a new Peer Exchange service.
// The handler is called when new peers are received via PEX.
func NewPEX(h host.Host, handler PeerHandler) *PEX {
	return &PEX{
		h:       h,
		handler: handler,
	}
}

// Start begins the PEX service.
// It registers the stream handler and starts periodic peer exchanges.
func (p *PEX) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	// Register stream handler for incoming PEX requests.
	p.h.SetStreamHandler(PEXProtocolID, p.handleStream)

	// Start periodic exchange loop.
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.running = true

	go p.exchangeLoop(ctx)

	return nil
}

// Stop halts the PEX service.
func (p *PEX) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.h.RemoveStreamHandler(PEXProtocolID)
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.running = false

	return nil
}

// IsRunning returns true if PEX is active.
func (p *PEX) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// exchangeLoop periodically sends peer lists to connected peers.
func (p *PEX) exchangeLoop(ctx context.Context) {
	ticker := time.NewTicker(PEXInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.broadcastPeers(ctx)
		}
	}
}

// broadcastPeers sends a random sample of peers to all connected peers.
func (p *PEX) broadcastPeers(ctx context.Context) {
	peers := p.h.Network().Peers()
	if len(peers) == 0 {
		return
	}

	// Get a sample of peers to share.
	sample := p.samplePeers(PEXSampleSize)
	if len(sample) == 0 {
		return
	}

	// Send to each connected peer.
	for _, peerID := range peers {
		go p.sendToPeer(ctx, peerID, sample)
	}
}

// samplePeers returns a random sample of peers from the peerstore.
func (p *PEX) samplePeers(n int) []PeerInfo {
	allPeers := p.h.Peerstore().PeersWithAddrs()
	if len(allPeers) == 0 {
		return nil
	}

	// Filter out self.
	selfID := p.h.ID()
	var validPeers []peer.ID
	for _, peerID := range allPeers {
		if peerID != selfID {
			validPeers = append(validPeers, peerID)
		}
	}

	if len(validPeers) == 0 {
		return nil
	}

	// Shuffle and take up to n peers.
	rand.Shuffle(len(validPeers), func(i, j int) {
		validPeers[i], validPeers[j] = validPeers[j], validPeers[i]
	})

	if len(validPeers) > n {
		validPeers = validPeers[:n]
	}

	// Build PeerInfo list.
	result := make([]PeerInfo, 0, len(validPeers))
	for _, peerID := range validPeers {
		addrs := p.h.Peerstore().Addrs(peerID)
		if len(addrs) > PEXMaxAddrs {
			addrs = addrs[:PEXMaxAddrs]
		}
		if len(addrs) > 0 {
			result = append(result, PeerInfo{
				ID:    peerID,
				Addrs: addrs,
			})
		}
	}

	return result
}

// sendToPeer sends a list of peers to a specific peer.
func (p *PEX) sendToPeer(ctx context.Context, peerID peer.ID, peers []PeerInfo) {
	ctx, cancel := context.WithTimeout(ctx, PEXWriteTimeout)
	defer cancel()

	s, err := p.h.NewStream(ctx, peerID, PEXProtocolID)
	if err != nil {
		return // Silently ignore connection failures
	}
	defer s.Close()

	if err := s.SetWriteDeadline(time.Now().Add(PEXWriteTimeout)); err != nil {
		return
	}

	if err := writePeerList(s, peers); err != nil {
		return
	}
}

// handleStream handles incoming PEX streams.
func (p *PEX) handleStream(s network.Stream) {
	defer s.Close()

	if err := s.SetReadDeadline(time.Now().Add(PEXReadTimeout)); err != nil {
		return
	}

	peers, err := readPeerList(s)
	if err != nil {
		return
	}

	// Process received peers.
	for _, pi := range peers {
		// Skip self.
		if pi.ID == p.h.ID() {
			continue
		}

		// Add to peerstore.
		p.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)

		// Notify handler.
		if p.handler != nil {
			p.handler(peer.AddrInfo{ID: pi.ID, Addrs: pi.Addrs})
		}
	}

	// Send our peers back.
	if err := s.SetWriteDeadline(time.Now().Add(PEXWriteTimeout)); err != nil {
		return
	}
	sample := p.samplePeers(PEXSampleSize)
	_ = writePeerList(s, sample)
}

// ExchangeWithPeer performs an immediate peer exchange with a specific peer.
// This is useful for initial connection bootstrapping.
func (p *PEX) ExchangeWithPeer(ctx context.Context, peerID peer.ID) ([]PeerInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, PEXWriteTimeout+PEXReadTimeout)
	defer cancel()

	s, err := p.h.NewStream(ctx, peerID, PEXProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()

	if err := p.sendPeerList(s); err != nil {
		return nil, err
	}

	if err := s.CloseWrite(); err != nil {
		return nil, fmt.Errorf("failed to close write: %w", err)
	}

	peers, err := p.receivePeerList(s)
	if err != nil {
		return nil, err
	}

	p.processReceivedPeers(peers)
	return peers, nil
}

// sendPeerList sends our peer sample to the remote peer.
func (p *PEX) sendPeerList(s network.Stream) error {
	sample := p.samplePeers(PEXSampleSize)
	if err := s.SetWriteDeadline(time.Now().Add(PEXWriteTimeout)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	if err := writePeerList(s, sample); err != nil {
		return fmt.Errorf("failed to write peer list: %w", err)
	}
	return nil
}

// receivePeerList reads the peer list from the remote peer.
func (p *PEX) receivePeerList(s network.Stream) ([]PeerInfo, error) {
	if err := s.SetReadDeadline(time.Now().Add(PEXReadTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}
	peers, err := readPeerList(s)
	if err != nil {
		return nil, fmt.Errorf("failed to read peer list: %w", err)
	}
	return peers, nil
}

// processReceivedPeers adds received peers to the peerstore and invokes the handler.
func (p *PEX) processReceivedPeers(peers []PeerInfo) {
	for _, pi := range peers {
		if pi.ID != p.h.ID() {
			p.h.Peerstore().AddAddrs(pi.ID, pi.Addrs, time.Hour)
			if p.handler != nil {
				p.handler(peer.AddrInfo{ID: pi.ID, Addrs: pi.Addrs})
			}
		}
	}
}

// Wire format for peer lists:
// - 4 bytes: number of peers (uint32, little-endian)
// - For each peer:
//   - 2 bytes: peer ID length (uint16, little-endian)
//   - N bytes: peer ID
//   - 2 bytes: number of addresses (uint16, little-endian)
//   - For each address:
//     - 2 bytes: address length (uint16, little-endian)
//     - N bytes: multiaddr bytes

// writePeerList writes a list of peers to a stream.
func writePeerList(w io.Writer, peers []PeerInfo) error {
	bw := bufio.NewWriter(w)

	if err := binary.Write(bw, binary.LittleEndian, uint32(len(peers))); err != nil {
		return err
	}

	for _, pi := range peers {
		if err := writePeerInfo(bw, pi); err != nil {
			return err
		}
	}

	return bw.Flush()
}

// writePeerInfo writes a single PeerInfo to the buffer.
func writePeerInfo(bw *bufio.Writer, pi PeerInfo) error {
	idBytes := []byte(pi.ID)
	if err := binary.Write(bw, binary.LittleEndian, uint16(len(idBytes))); err != nil {
		return err
	}
	if _, err := bw.Write(idBytes); err != nil {
		return err
	}

	if err := binary.Write(bw, binary.LittleEndian, uint16(len(pi.Addrs))); err != nil {
		return err
	}

	for _, addr := range pi.Addrs {
		if err := writeMultiaddr(bw, addr); err != nil {
			return err
		}
	}

	return nil
}

// writeMultiaddr writes a multiaddr to the buffer.
func writeMultiaddr(bw *bufio.Writer, addr multiaddr.Multiaddr) error {
	addrBytes := addr.Bytes()
	if err := binary.Write(bw, binary.LittleEndian, uint16(len(addrBytes))); err != nil {
		return err
	}
	if _, err := bw.Write(addrBytes); err != nil {
		return err
	}
	return nil
}

// readPeerList reads a list of peers from a stream.
func readPeerList(r io.Reader) ([]PeerInfo, error) {
	br := bufio.NewReader(io.LimitReader(r, PEXMaxMessageSize))

	numPeers, err := readPeerCount(br)
	if err != nil {
		return nil, err
	}

	peers := make([]PeerInfo, 0, numPeers)
	for i := uint32(0); i < numPeers; i++ {
		pi, err := readSinglePeer(br)
		if err != nil {
			return nil, err
		}
		if pi != nil {
			peers = append(peers, *pi)
		}
	}

	return peers, nil
}

// readPeerCount reads and validates the peer count header.
func readPeerCount(br *bufio.Reader) (uint32, error) {
	var numPeers uint32
	if err := binary.Read(br, binary.LittleEndian, &numPeers); err != nil {
		return 0, err
	}
	if numPeers > 100 {
		return 0, fmt.Errorf("too many peers: %d", numPeers)
	}
	return numPeers, nil
}

// readSinglePeer reads one peer's ID and addresses.
func readSinglePeer(br *bufio.Reader) (*PeerInfo, error) {
	peerID, err := readPeerID(br)
	if err != nil {
		return nil, err
	}

	addrs, err := readPeerAddresses(br)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, nil // Skip peers with no valid addresses.
	}
	return &PeerInfo{ID: peerID, Addrs: addrs}, nil
}

// readPeerID reads a peer ID from the buffer.
func readPeerID(br *bufio.Reader) (peer.ID, error) {
	var idLen uint16
	if err := binary.Read(br, binary.LittleEndian, &idLen); err != nil {
		return "", err
	}
	if idLen > 256 {
		return "", fmt.Errorf("peer ID too long: %d", idLen)
	}
	idBytes := make([]byte, idLen)
	if _, err := io.ReadFull(br, idBytes); err != nil {
		return "", err
	}
	return peer.ID(idBytes), nil
}

// readPeerAddresses reads multiaddresses for a peer.
func readPeerAddresses(br *bufio.Reader) ([]multiaddr.Multiaddr, error) {
	var numAddrs uint16
	if err := binary.Read(br, binary.LittleEndian, &numAddrs); err != nil {
		return nil, err
	}
	if numAddrs > PEXMaxAddrs {
		numAddrs = PEXMaxAddrs
	}

	addrs := make([]multiaddr.Multiaddr, 0, numAddrs)
	for j := uint16(0); j < numAddrs; j++ {
		addr, err := readSingleAddress(br)
		if err != nil {
			return nil, err
		}
		if addr != nil {
			addrs = append(addrs, addr)
		}
	}
	return addrs, nil
}

// readSingleAddress reads one multiaddr from the buffer.
func readSingleAddress(br *bufio.Reader) (multiaddr.Multiaddr, error) {
	var addrLen uint16
	if err := binary.Read(br, binary.LittleEndian, &addrLen); err != nil {
		return nil, err
	}
	if addrLen > 512 {
		return nil, fmt.Errorf("address too long: %d", addrLen)
	}
	addrBytes := make([]byte, addrLen)
	if _, err := io.ReadFull(br, addrBytes); err != nil {
		return nil, err
	}
	addr, err := multiaddr.NewMultiaddrBytes(addrBytes)
	if err != nil {
		return nil, nil // Skip malformed addresses.
	}
	return addr, nil
}
