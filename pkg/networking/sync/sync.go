// Package sync implements Wave synchronization protocol.
// Per TECHNICAL_IMPLEMENTATION.md §5.4, the `/murmur/wave-sync/1` stream protocol
// enables request-response fetching of Waves by hash for catch-up and missed messages.
package sync

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// Protocol constants.
const (
	// WaveSyncProtocol is the protocol ID for Wave sync.
	WaveSyncProtocol = protocol.ID("/murmur/wave-sync/1")

	// MaxMessagesPerRequest limits Waves per sync request.
	MaxMessagesPerRequest = 1000

	// MaxConcurrentSessions limits concurrent sync sessions.
	MaxConcurrentSessions = 5

	// RateLimitPerSecond is messages per second rate limit.
	RateLimitPerSecond = 100

	// RequestTimeout is the timeout for sync requests.
	RequestTimeout = 30 * time.Second

	// MaxRequestSize limits request payload size.
	MaxRequestSize = 64 * 1024 // 64 KB

	// MaxResponseSize limits response payload size.
	MaxResponseSize = 4 * 1024 * 1024 // 4 MB (1000 waves * ~4KB each)
)

// Request type constants.
const (
	RequestTypeByHash    byte = 0x01
	RequestTypeByTopic   byte = 0x02
	RequestTypeByAuthor  byte = 0x03
	RequestTypeSince     byte = 0x04
	RequestTypeLatestN   byte = 0x05
)

// Response status codes.
const (
	StatusOK           byte = 0x00
	StatusNotFound     byte = 0x01
	StatusRateLimited  byte = 0x02
	StatusTooManyPeers byte = 0x03
	StatusInvalidReq   byte = 0x04
	StatusInternal     byte = 0x05
)

// Errors returned by sync operations.
var (
	ErrRateLimited      = errors.New("rate limited")
	ErrTooManySessions  = errors.New("too many concurrent sessions")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrNotFound         = errors.New("wave not found")
	ErrRequestTimeout   = errors.New("request timeout")
	ErrResponseTooLarge = errors.New("response too large")
)

// WaveProvider is an interface for retrieving Waves from local storage.
type WaveProvider interface {
	GetWaveByHash(hash []byte) ([]byte, error)
	GetWavesByTopic(topic string, limit int) ([][]byte, error)
	GetWavesByAuthor(pubkey []byte, limit int) ([][]byte, error)
	GetWavesSince(timestamp int64, limit int) ([][]byte, error)
	GetLatestWaves(n int) ([][]byte, error)
}

// SyncRequest represents a sync request.
type SyncRequest struct {
	Type      byte
	Hashes    [][]byte // For RequestTypeByHash
	Topic     string   // For RequestTypeByTopic
	Author    []byte   // For RequestTypeByAuthor
	Since     int64    // For RequestTypeSince (Unix timestamp)
	Limit     int      // For RequestTypeLatestN or pagination
}

// SyncResponse represents a sync response.
type SyncResponse struct {
	Status byte
	Waves  [][]byte
	More   bool // True if more results available
}

// SyncHandler handles Wave sync protocol.
type SyncHandler struct {
	h               host.Host
	provider        WaveProvider
	mu              sync.RWMutex
	activeSessions  int32
	rateLimiter     *rateLimiter
	callbacks       SyncCallbacks
}

// SyncCallbacks are callbacks for sync events.
type SyncCallbacks struct {
	OnSyncRequest  func(peer peer.ID, reqType byte)
	OnSyncComplete func(peer peer.ID, waveCount int)
	OnRateLimited  func(peer peer.ID)
}

// NewSyncHandler creates a new sync handler.
func NewSyncHandler(h host.Host, provider WaveProvider) *SyncHandler {
	sh := &SyncHandler{
		h:           h,
		provider:    provider,
		rateLimiter: newRateLimiter(RateLimitPerSecond),
	}

	// Register stream handler
	h.SetStreamHandler(WaveSyncProtocol, sh.handleStream)

	return sh
}

// SetCallbacks sets the sync event callbacks.
func (sh *SyncHandler) SetCallbacks(cb SyncCallbacks) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.callbacks = cb
}

// Close removes the stream handler.
func (sh *SyncHandler) Close() {
	sh.h.RemoveStreamHandler(WaveSyncProtocol)
}

// handleStream handles incoming sync streams.
func (sh *SyncHandler) handleStream(s network.Stream) {
	defer s.Close()

	remotePeer := s.Conn().RemotePeer()

	// Check concurrent sessions
	if atomic.AddInt32(&sh.activeSessions, 1) > MaxConcurrentSessions {
		atomic.AddInt32(&sh.activeSessions, -1)
		sh.writeResponse(s, &SyncResponse{Status: StatusTooManyPeers})
		return
	}
	defer atomic.AddInt32(&sh.activeSessions, -1)

	// Check rate limit
	if !sh.rateLimiter.allow(remotePeer) {
		sh.writeResponse(s, &SyncResponse{Status: StatusRateLimited})
		sh.mu.RLock()
		cb := sh.callbacks.OnRateLimited
		sh.mu.RUnlock()
		if cb != nil {
			cb(remotePeer)
		}
		return
	}

	// Set read deadline
	_ = s.SetReadDeadline(time.Now().Add(RequestTimeout))

	// Read request
	req, err := sh.readRequest(s)
	if err != nil {
		sh.writeResponse(s, &SyncResponse{Status: StatusInvalidReq})
		return
	}

	sh.mu.RLock()
	cbReq := sh.callbacks.OnSyncRequest
	sh.mu.RUnlock()
	if cbReq != nil {
		cbReq(remotePeer, req.Type)
	}

	// Process request
	resp := sh.processRequest(req)

	// Write response
	_ = s.SetWriteDeadline(time.Now().Add(RequestTimeout))
	sh.writeResponse(s, resp)

	sh.mu.RLock()
	cbComplete := sh.callbacks.OnSyncComplete
	sh.mu.RUnlock()
	if cbComplete != nil {
		cbComplete(remotePeer, len(resp.Waves))
	}
}

// readRequest reads and parses a sync request.
func (sh *SyncHandler) readRequest(r io.Reader) (*SyncRequest, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	if length > MaxRequestSize {
		return nil, ErrInvalidRequest
	}

	// Read payload
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	return parseRequest(payload)
}

// parseRequest parses request payload.
func parseRequest(data []byte) (*SyncRequest, error) {
	if len(data) < 1 {
		return nil, ErrInvalidRequest
	}

	req := &SyncRequest{
		Type: data[0],
	}

	switch req.Type {
	case RequestTypeByHash:
		return parseHashRequest(req, data[1:])
	case RequestTypeByTopic:
		return parseTopicRequest(req, data[1:])
	case RequestTypeByAuthor:
		return parseAuthorRequest(req, data[1:])
	case RequestTypeSince:
		return parseSinceRequest(req, data[1:])
	case RequestTypeLatestN:
		return parseLatestNRequest(req, data[1:])
	default:
		return nil, ErrInvalidRequest
	}
}

// parseHashRequest parses request for specific hashes.
func parseHashRequest(req *SyncRequest, data []byte) (*SyncRequest, error) {
	if len(data) < 2 {
		return nil, ErrInvalidRequest
	}

	count := int(binary.BigEndian.Uint16(data[:2]))
	if count > MaxMessagesPerRequest {
		count = MaxMessagesPerRequest
	}

	data = data[2:]
	req.Hashes = make([][]byte, 0, count)

	for i := 0; i < count && len(data) >= 32; i++ {
		hash := make([]byte, 32)
		copy(hash, data[:32])
		req.Hashes = append(req.Hashes, hash)
		data = data[32:]
	}

	return req, nil
}

// parseTopicRequest parses request for topic-based sync.
func parseTopicRequest(req *SyncRequest, data []byte) (*SyncRequest, error) {
	if len(data) < 4 {
		return nil, ErrInvalidRequest
	}

	topicLen := int(binary.BigEndian.Uint16(data[:2]))
	req.Limit = int(binary.BigEndian.Uint16(data[2:4]))

	if req.Limit > MaxMessagesPerRequest {
		req.Limit = MaxMessagesPerRequest
	}

	if len(data) < 4+topicLen {
		return nil, ErrInvalidRequest
	}

	req.Topic = string(data[4 : 4+topicLen])
	return req, nil
}

// parseAuthorRequest parses request for author-based sync.
func parseAuthorRequest(req *SyncRequest, data []byte) (*SyncRequest, error) {
	if len(data) < 34 { // 32-byte pubkey + 2-byte limit
		return nil, ErrInvalidRequest
	}

	req.Author = make([]byte, 32)
	copy(req.Author, data[:32])
	req.Limit = int(binary.BigEndian.Uint16(data[32:34]))

	if req.Limit > MaxMessagesPerRequest {
		req.Limit = MaxMessagesPerRequest
	}

	return req, nil
}

// parseSinceRequest parses request for time-based sync.
func parseSinceRequest(req *SyncRequest, data []byte) (*SyncRequest, error) {
	if len(data) < 10 { // 8-byte timestamp + 2-byte limit
		return nil, ErrInvalidRequest
	}

	req.Since = int64(binary.BigEndian.Uint64(data[:8]))
	req.Limit = int(binary.BigEndian.Uint16(data[8:10]))

	if req.Limit > MaxMessagesPerRequest {
		req.Limit = MaxMessagesPerRequest
	}

	return req, nil
}

// parseLatestNRequest parses request for latest N waves.
func parseLatestNRequest(req *SyncRequest, data []byte) (*SyncRequest, error) {
	if len(data) < 2 {
		return nil, ErrInvalidRequest
	}

	req.Limit = int(binary.BigEndian.Uint16(data[:2]))
	if req.Limit > MaxMessagesPerRequest {
		req.Limit = MaxMessagesPerRequest
	}

	return req, nil
}

// processRequest processes a sync request and returns a response.
func (sh *SyncHandler) processRequest(req *SyncRequest) *SyncResponse {
	if sh.provider == nil {
		return &SyncResponse{Status: StatusInternal}
	}

	var waves [][]byte
	var err error

	switch req.Type {
	case RequestTypeByHash:
		waves, err = sh.fetchByHashes(req.Hashes)
	case RequestTypeByTopic:
		waves, err = sh.provider.GetWavesByTopic(req.Topic, req.Limit)
	case RequestTypeByAuthor:
		waves, err = sh.provider.GetWavesByAuthor(req.Author, req.Limit)
	case RequestTypeSince:
		waves, err = sh.provider.GetWavesSince(req.Since, req.Limit)
	case RequestTypeLatestN:
		waves, err = sh.provider.GetLatestWaves(req.Limit)
	default:
		return &SyncResponse{Status: StatusInvalidReq}
	}

	if err != nil {
		return &SyncResponse{Status: StatusNotFound}
	}

	return &SyncResponse{
		Status: StatusOK,
		Waves:  waves,
		More:   len(waves) == req.Limit,
	}
}

// fetchByHashes fetches waves by their hashes.
func (sh *SyncHandler) fetchByHashes(hashes [][]byte) ([][]byte, error) {
	waves := make([][]byte, 0, len(hashes))

	for _, hash := range hashes {
		wave, err := sh.provider.GetWaveByHash(hash)
		if err == nil && wave != nil {
			waves = append(waves, wave)
		}
	}

	if len(waves) == 0 {
		return nil, ErrNotFound
	}

	return waves, nil
}

// writeResponse writes a sync response.
func (sh *SyncHandler) writeResponse(w io.Writer, resp *SyncResponse) {
	data := serializeResponse(resp)

	// Write length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))

	_, _ = w.Write(lenBuf)
	_, _ = w.Write(data)
}

// serializeResponse serializes a response to bytes.
func serializeResponse(resp *SyncResponse) []byte {
	// Calculate size: 1 (status) + 1 (more flag) + 2 (count) + waves
	size := 4
	for _, wave := range resp.Waves {
		size += 4 + len(wave) // 4-byte length prefix per wave
	}

	data := make([]byte, size)
	data[0] = resp.Status

	if resp.More {
		data[1] = 1
	}

	binary.BigEndian.PutUint16(data[2:4], uint16(len(resp.Waves)))

	offset := 4
	for _, wave := range resp.Waves {
		binary.BigEndian.PutUint32(data[offset:offset+4], uint32(len(wave)))
		offset += 4
		copy(data[offset:], wave)
		offset += len(wave)
	}

	return data
}

// rateLimiter provides per-peer rate limiting.
type rateLimiter struct {
	mu          sync.Mutex
	ratePerSec  int
	buckets     map[peer.ID]*bucket
	cleanupTime time.Time
}

type bucket struct {
	tokens     int
	lastRefill time.Time
}

func newRateLimiter(ratePerSec int) *rateLimiter {
	return &rateLimiter{
		ratePerSec:  ratePerSec,
		buckets:     make(map[peer.ID]*bucket),
		cleanupTime: time.Now(),
	}
}

func (rl *rateLimiter) allow(p peer.ID) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodic cleanup
	if time.Since(rl.cleanupTime) > time.Minute {
		rl.cleanup()
	}

	b, ok := rl.buckets[p]
	if !ok {
		b = &bucket{tokens: rl.ratePerSec, lastRefill: time.Now()}
		rl.buckets[p] = b
	}

	// Refill tokens
	elapsed := time.Since(b.lastRefill)
	refill := int(elapsed.Seconds()) * rl.ratePerSec
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.ratePerSec*10 { // Max burst
			b.tokens = rl.ratePerSec * 10
		}
		b.lastRefill = time.Now()
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

func (rl *rateLimiter) cleanup() {
	cutoff := time.Now().Add(-5 * time.Minute)
	for p, b := range rl.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(rl.buckets, p)
		}
	}
	rl.cleanupTime = time.Now()
}
