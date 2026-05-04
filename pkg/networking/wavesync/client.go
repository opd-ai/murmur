// Package sync implements Wave synchronization protocol.
// This file provides the client-side sync functionality.
package wavesync

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

// SyncClient provides client-side sync functionality.
type SyncClient struct {
	h              host.Host
	mu             sync.RWMutex
	activeSessions int32
	maxSessions    int32
	callbacks      ClientCallbacks
}

// ClientCallbacks are callbacks for client sync events.
type ClientCallbacks struct {
	OnSyncStart    func(peer peer.ID)
	OnSyncComplete func(peer peer.ID, received int)
	OnSyncError    func(peer peer.ID, err error)
}

// NewSyncClient creates a new sync client.
func NewSyncClient(h host.Host) *SyncClient {
	return &SyncClient{
		h:           h,
		maxSessions: MaxConcurrentSessions,
	}
}

// SetCallbacks sets client event callbacks.
func (sc *SyncClient) SetCallbacks(cb ClientCallbacks) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.callbacks = cb
}

// SetMaxSessions sets max concurrent outgoing sessions.
func (sc *SyncClient) SetMaxSessions(max int32) {
	atomic.StoreInt32(&sc.maxSessions, max)
}

// ActiveSessions returns count of active outgoing sessions.
func (sc *SyncClient) ActiveSessions() int32 {
	return atomic.LoadInt32(&sc.activeSessions)
}

// RequestByHashes requests specific Waves by their hashes.
func (sc *SyncClient) RequestByHashes(ctx context.Context, p peer.ID, hashes [][]byte) (*SyncResponse, error) {
	if len(hashes) > MaxMessagesPerRequest {
		hashes = hashes[:MaxMessagesPerRequest]
	}

	// Build request payload
	payload := make([]byte, 3+len(hashes)*32)
	payload[0] = RequestTypeByHash
	binary.BigEndian.PutUint16(payload[1:3], uint16(len(hashes)))

	offset := 3
	for _, hash := range hashes {
		if len(hash) == 32 {
			copy(payload[offset:], hash)
		}
		offset += 32
	}

	return sc.sendRequest(ctx, p, payload)
}

// RequestByTopic requests Waves from a specific topic.
func (sc *SyncClient) RequestByTopic(ctx context.Context, p peer.ID, topic string, limit int) (*SyncResponse, error) {
	if limit > MaxMessagesPerRequest || limit <= 0 {
		limit = MaxMessagesPerRequest
	}

	topicBytes := []byte(topic)
	payload := make([]byte, 5+len(topicBytes))
	payload[0] = RequestTypeByTopic
	binary.BigEndian.PutUint16(payload[1:3], uint16(len(topicBytes)))
	binary.BigEndian.PutUint16(payload[3:5], uint16(limit))
	copy(payload[5:], topicBytes)

	return sc.sendRequest(ctx, p, payload)
}

// RequestByAuthor requests Waves from a specific author.
func (sc *SyncClient) RequestByAuthor(ctx context.Context, p peer.ID, author []byte, limit int) (*SyncResponse, error) {
	if limit > MaxMessagesPerRequest || limit <= 0 {
		limit = MaxMessagesPerRequest
	}
	if len(author) != 32 {
		return nil, ErrInvalidRequest
	}

	payload := make([]byte, 35)
	payload[0] = RequestTypeByAuthor
	copy(payload[1:33], author)
	binary.BigEndian.PutUint16(payload[33:35], uint16(limit))

	return sc.sendRequest(ctx, p, payload)
}

// RequestSince requests Waves since a timestamp.
func (sc *SyncClient) RequestSince(ctx context.Context, p peer.ID, since time.Time, limit int) (*SyncResponse, error) {
	if limit > MaxMessagesPerRequest || limit <= 0 {
		limit = MaxMessagesPerRequest
	}

	payload := make([]byte, 11)
	payload[0] = RequestTypeSince
	binary.BigEndian.PutUint64(payload[1:9], uint64(since.Unix()))
	binary.BigEndian.PutUint16(payload[9:11], uint16(limit))

	return sc.sendRequest(ctx, p, payload)
}

// RequestLatest requests the N most recent Waves.
func (sc *SyncClient) RequestLatest(ctx context.Context, p peer.ID, n int) (*SyncResponse, error) {
	if n > MaxMessagesPerRequest || n <= 0 {
		n = MaxMessagesPerRequest
	}

	payload := make([]byte, 3)
	payload[0] = RequestTypeLatestN
	binary.BigEndian.PutUint16(payload[1:3], uint16(n))

	return sc.sendRequest(ctx, p, payload)
}

// sendRequest sends a sync request to a peer.
func (sc *SyncClient) sendRequest(ctx context.Context, p peer.ID, payload []byte) (*SyncResponse, error) {
	if err := sc.acquireSession(); err != nil {
		return nil, err
	}
	defer sc.releaseSession()

	sc.notifyStart(p)

	resp, err := sc.executeRequest(ctx, p, payload)
	if err != nil {
		sc.notifyError(p, err)
		return nil, err
	}

	sc.notifyComplete(p, len(resp.Waves))
	return resp, nil
}

// acquireSession checks and increments the active session counter.
func (sc *SyncClient) acquireSession() error {
	if atomic.AddInt32(&sc.activeSessions, 1) > atomic.LoadInt32(&sc.maxSessions) {
		atomic.AddInt32(&sc.activeSessions, -1)
		return ErrTooManySessions
	}
	return nil
}

// releaseSession decrements the active session counter.
func (sc *SyncClient) releaseSession() {
	atomic.AddInt32(&sc.activeSessions, -1)
}

// notifyStart calls the OnSyncStart callback if set.
func (sc *SyncClient) notifyStart(p peer.ID) {
	sc.mu.RLock()
	cb := sc.callbacks.OnSyncStart
	sc.mu.RUnlock()
	if cb != nil {
		cb(p)
	}
}

// notifyComplete calls the OnSyncComplete callback if set.
func (sc *SyncClient) notifyComplete(p peer.ID, waveCount int) {
	sc.mu.RLock()
	cb := sc.callbacks.OnSyncComplete
	sc.mu.RUnlock()
	if cb != nil {
		cb(p, waveCount)
	}
}

// notifyError calls the OnSyncError callback if set.
func (sc *SyncClient) notifyError(p peer.ID, err error) {
	sc.mu.RLock()
	cb := sc.callbacks.OnSyncError
	sc.mu.RUnlock()
	if cb != nil {
		cb(p, err)
	}
}

// executeRequest performs the actual stream communication.
func (sc *SyncClient) executeRequest(ctx context.Context, p peer.ID, payload []byte) (*SyncResponse, error) {
	s, err := sc.h.NewStream(ctx, p, WaveSyncProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}
	defer s.Close()

	sc.setStreamDeadline(ctx, s)

	if err := sc.writePayload(s, payload); err != nil {
		return nil, err
	}

	return sc.readResponse(s)
}

// setStreamDeadline configures the stream timeout.
func (sc *SyncClient) setStreamDeadline(ctx context.Context, s network.Stream) {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(RequestTimeout)
	}
	_ = s.SetDeadline(deadline)
}

// writePayload writes the request payload with length prefix.
func (sc *SyncClient) writePayload(s network.Stream, payload []byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))

	if _, err := s.Write(lenBuf); err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}
	if _, err := s.Write(payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}
	return nil
}

// readResponse reads and parses a sync response.
func (sc *SyncClient) readResponse(r io.Reader) (*SyncResponse, error) {
	// Read length prefix
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf)
	if length > MaxResponseSize {
		return nil, ErrResponseTooLarge
	}

	// Read payload
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	return parseResponse(payload)
}

// parseResponse parses response payload.
func parseResponse(data []byte) (*SyncResponse, error) {
	if len(data) < 4 {
		return nil, ErrInvalidRequest
	}

	resp := &SyncResponse{
		Status: data[0],
		More:   data[1] == 1,
	}

	// Check for error status
	switch resp.Status {
	case StatusRateLimited:
		return resp, ErrRateLimited
	case StatusTooManyPeers:
		return resp, ErrTooManySessions
	case StatusInvalidReq:
		return resp, ErrInvalidRequest
	case StatusNotFound:
		return resp, ErrNotFound
	}

	count := int(binary.BigEndian.Uint16(data[2:4]))
	resp.Waves = make([][]byte, 0, count)

	offset := 4
	for i := 0; i < count && offset+4 <= len(data); i++ {
		waveLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		offset += 4

		if offset+waveLen > len(data) {
			break
		}

		wave := make([]byte, waveLen)
		copy(wave, data[offset:offset+waveLen])
		resp.Waves = append(resp.Waves, wave)
		offset += waveLen
	}

	return resp, nil
}

// SyncSession manages a catch-up sync session with a peer.
type SyncSession struct {
	client        *SyncClient
	peer          peer.ID
	lastSync      time.Time
	mu            sync.Mutex
	receivedWaves [][]byte
	onWave        func([]byte)
}

// NewSyncSession creates a new sync session.
func NewSyncSession(client *SyncClient, p peer.ID) *SyncSession {
	return &SyncSession{
		client:        client,
		peer:          p,
		receivedWaves: make([][]byte, 0),
	}
}

// SetWaveCallback sets callback for each received Wave.
func (ss *SyncSession) SetWaveCallback(cb func([]byte)) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.onWave = cb
}

// CatchUp performs a catch-up sync from the last sync time.
func (ss *SyncSession) CatchUp(ctx context.Context) (int, error) {
	ss.mu.Lock()
	since := ss.lastSync
	cb := ss.onWave
	ss.mu.Unlock()

	if since.IsZero() {
		since = time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	}

	totalReceived := 0

	for {
		resp, err := ss.client.RequestSince(ctx, ss.peer, since, MaxMessagesPerRequest)
		if err != nil {
			if totalReceived > 0 {
				// Partial success
				break
			}
			return totalReceived, err
		}

		for _, wave := range resp.Waves {
			ss.mu.Lock()
			ss.receivedWaves = append(ss.receivedWaves, wave)
			ss.mu.Unlock()

			if cb != nil {
				cb(wave)
			}
		}

		totalReceived += len(resp.Waves)

		if !resp.More || len(resp.Waves) == 0 {
			break
		}

		// Update since for next page (would need Wave timestamp parsing)
		// For now, just break after first batch
		break
	}

	ss.mu.Lock()
	ss.lastSync = time.Now()
	ss.mu.Unlock()

	return totalReceived, nil
}

// FetchMissing fetches specific missing Waves by hash.
func (ss *SyncSession) FetchMissing(ctx context.Context, hashes [][]byte) (int, error) {
	ss.mu.Lock()
	cb := ss.onWave
	ss.mu.Unlock()

	totalReceived := 0

	// Process in batches
	for i := 0; i < len(hashes); i += MaxMessagesPerRequest {
		end := i + MaxMessagesPerRequest
		if end > len(hashes) {
			end = len(hashes)
		}

		batch := hashes[i:end]
		resp, err := ss.client.RequestByHashes(ctx, ss.peer, batch)
		if err != nil {
			if totalReceived > 0 {
				break
			}
			return totalReceived, err
		}

		for _, wave := range resp.Waves {
			ss.mu.Lock()
			ss.receivedWaves = append(ss.receivedWaves, wave)
			ss.mu.Unlock()

			if cb != nil {
				cb(wave)
			}
		}

		totalReceived += len(resp.Waves)
	}

	return totalReceived, nil
}

// ReceivedWaves returns all Waves received in this session.
func (ss *SyncSession) ReceivedWaves() [][]byte {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	result := make([][]byte, len(ss.receivedWaves))
	copy(result, ss.receivedWaves)
	return result
}

// Clear clears received Waves from the session.
func (ss *SyncSession) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.receivedWaves = make([][]byte, 0)
}
