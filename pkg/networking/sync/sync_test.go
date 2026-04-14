package sync

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

// mockWaveProvider implements WaveProvider for testing.
type mockWaveProvider struct {
	mu       sync.RWMutex
	waves    map[string][]byte // hash (hex) -> wave data
	byTopic  map[string][][]byte
	byAuthor map[string][][]byte
	latest   [][]byte
}

func newMockWaveProvider() *mockWaveProvider {
	return &mockWaveProvider{
		waves:    make(map[string][]byte),
		byTopic:  make(map[string][][]byte),
		byAuthor: make(map[string][][]byte),
		latest:   make([][]byte, 0),
	}
}

func (m *mockWaveProvider) AddWave(hash, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := string(hash)
	m.waves[key] = data
	m.latest = append(m.latest, data)
}

func (m *mockWaveProvider) AddToTopic(topic string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byTopic[topic] = append(m.byTopic[topic], data)
}

func (m *mockWaveProvider) AddByAuthor(author, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := string(author)
	m.byAuthor[key] = append(m.byAuthor[key], data)
}

func (m *mockWaveProvider) GetWaveByHash(hash []byte) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if data, ok := m.waves[string(hash)]; ok {
		return data, nil
	}
	return nil, ErrNotFound
}

func (m *mockWaveProvider) GetWavesByTopic(topic string, limit int) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	waves := m.byTopic[topic]
	if len(waves) == 0 {
		return nil, ErrNotFound
	}
	if limit > len(waves) {
		limit = len(waves)
	}
	return waves[:limit], nil
}

func (m *mockWaveProvider) GetWavesByAuthor(pubkey []byte, limit int) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	waves := m.byAuthor[string(pubkey)]
	if len(waves) == 0 {
		return nil, ErrNotFound
	}
	if limit > len(waves) {
		limit = len(waves)
	}
	return waves[:limit], nil
}

func (m *mockWaveProvider) GetWavesSince(timestamp int64, limit int) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return all latest for simplicity
	if len(m.latest) == 0 {
		return nil, ErrNotFound
	}
	if limit > len(m.latest) {
		limit = len(m.latest)
	}
	return m.latest[:limit], nil
}

func (m *mockWaveProvider) GetLatestWaves(n int) ([][]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.latest) == 0 {
		return nil, ErrNotFound
	}
	if n > len(m.latest) {
		n = len(m.latest)
	}
	return m.latest[:n], nil
}

func TestNewSyncHandler(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	provider := newMockWaveProvider()
	handler := NewSyncHandler(h, provider)
	defer handler.Close()

	if handler.h != h {
		t.Error("Host not set correctly")
	}
	if handler.provider != provider {
		t.Error("Provider not set correctly")
	}
}

func TestSyncHandlerCallbacks(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	handler := NewSyncHandler(h, newMockWaveProvider())
	defer handler.Close()

	called := false
	handler.SetCallbacks(SyncCallbacks{
		OnSyncRequest: func(p peer.ID, reqType byte) {
			called = true
		},
	})

	_ = called // Silence unused warning in test

	handler.mu.RLock()
	hasCallback := handler.callbacks.OnSyncRequest != nil
	handler.mu.RUnlock()

	if !hasCallback {
		t.Error("Callback not set")
	}
}

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		reqType byte
	}{
		{
			name:    "empty request",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid type",
			data:    []byte{0xFF},
			wantErr: true,
		},
		{
			name:    "hash request too short",
			data:    []byte{RequestTypeByHash},
			wantErr: true,
		},
		{
			name:    "valid latest N request",
			data:    []byte{RequestTypeLatestN, 0x00, 0x0A}, // 10 waves
			wantErr: false,
			reqType: RequestTypeLatestN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := parseRequest(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && req.Type != tt.reqType {
				t.Errorf("parseRequest() type = %v, want %v", req.Type, tt.reqType)
			}
		})
	}
}

func TestParseHashRequest(t *testing.T) {
	// Create request with 2 hashes
	data := make([]byte, 2+64) // 2-byte count + 2*32-byte hashes
	data[0] = 0
	data[1] = 2 // count = 2

	// Fill with test hashes
	for i := 0; i < 32; i++ {
		data[2+i] = byte(i)
		data[34+i] = byte(i + 32)
	}

	req := &SyncRequest{Type: RequestTypeByHash}
	result, err := parseHashRequest(req, data)
	if err != nil {
		t.Fatalf("parseHashRequest() error = %v", err)
	}

	if len(result.Hashes) != 2 {
		t.Errorf("Expected 2 hashes, got %d", len(result.Hashes))
	}
}

func TestParseTopicRequest(t *testing.T) {
	topic := "/murmur/waves/1"
	topicBytes := []byte(topic)

	// Build request: 2-byte topic len + 2-byte limit + topic
	data := make([]byte, 4+len(topicBytes))
	data[0] = 0
	data[1] = byte(len(topicBytes))
	data[2] = 0
	data[3] = 100 // limit = 100
	copy(data[4:], topicBytes)

	req := &SyncRequest{Type: RequestTypeByTopic}
	result, err := parseTopicRequest(req, data)
	if err != nil {
		t.Fatalf("parseTopicRequest() error = %v", err)
	}

	if result.Topic != topic {
		t.Errorf("Expected topic %q, got %q", topic, result.Topic)
	}
	if result.Limit != 100 {
		t.Errorf("Expected limit 100, got %d", result.Limit)
	}
}

func TestParseSinceRequest(t *testing.T) {
	timestamp := int64(1700000000)

	data := make([]byte, 10)
	data[0] = byte(timestamp >> 56)
	data[1] = byte(timestamp >> 48)
	data[2] = byte(timestamp >> 40)
	data[3] = byte(timestamp >> 32)
	data[4] = byte(timestamp >> 24)
	data[5] = byte(timestamp >> 16)
	data[6] = byte(timestamp >> 8)
	data[7] = byte(timestamp)
	data[8] = 0
	data[9] = 50 // limit = 50

	req := &SyncRequest{Type: RequestTypeSince}
	result, err := parseSinceRequest(req, data)
	if err != nil {
		t.Fatalf("parseSinceRequest() error = %v", err)
	}

	if result.Since != timestamp {
		t.Errorf("Expected timestamp %d, got %d", timestamp, result.Since)
	}
	if result.Limit != 50 {
		t.Errorf("Expected limit 50, got %d", result.Limit)
	}
}

func TestSerializeResponse(t *testing.T) {
	resp := &SyncResponse{
		Status: StatusOK,
		More:   true,
		Waves: [][]byte{
			[]byte("wave1"),
			[]byte("wave2"),
		},
	}

	data := serializeResponse(resp)

	if data[0] != StatusOK {
		t.Errorf("Expected status %d, got %d", StatusOK, data[0])
	}
	if data[1] != 1 { // More = true
		t.Errorf("Expected more flag 1, got %d", data[1])
	}

	// Parse and verify
	parsed, err := parseResponse(data)
	if err != nil {
		t.Fatalf("parseResponse() error = %v", err)
	}

	if len(parsed.Waves) != 2 {
		t.Errorf("Expected 2 waves, got %d", len(parsed.Waves))
	}
	if !bytes.Equal(parsed.Waves[0], []byte("wave1")) {
		t.Error("Wave 0 mismatch")
	}
	if !bytes.Equal(parsed.Waves[1], []byte("wave2")) {
		t.Error("Wave 1 mismatch")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := newRateLimiter(10) // 10 per second

	// Create a test peer ID
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	p := h.ID()

	// Should allow first 10 requests
	for i := 0; i < 10; i++ {
		if !rl.allow(p) {
			t.Errorf("Request %d should be allowed", i)
		}
	}
}

func TestNewSyncClient(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	client := NewSyncClient(h)

	if client.h != h {
		t.Error("Host not set correctly")
	}
	if client.maxSessions != MaxConcurrentSessions {
		t.Errorf("maxSessions = %d, want %d", client.maxSessions, MaxConcurrentSessions)
	}
}

func TestSyncClientSetMaxSessions(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	client := NewSyncClient(h)
	client.SetMaxSessions(10)

	if client.maxSessions != 10 {
		t.Errorf("maxSessions = %d, want 10", client.maxSessions)
	}
}

func TestSyncSession(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	client := NewSyncClient(h)
	session := NewSyncSession(client, h.ID())

	if session.client != client {
		t.Error("Client not set correctly")
	}

	// Test Clear
	session.mu.Lock()
	session.receivedWaves = [][]byte{[]byte("test")}
	session.mu.Unlock()

	session.Clear()

	waves := session.ReceivedWaves()
	if len(waves) != 0 {
		t.Errorf("Expected 0 waves after clear, got %d", len(waves))
	}
}

func TestSyncSessionCallback(t *testing.T) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host: %v", err)
	}
	defer h.Close()

	client := NewSyncClient(h)
	session := NewSyncSession(client, h.ID())

	var receivedCount int
	session.SetWaveCallback(func(wave []byte) {
		receivedCount++
	})

	_ = receivedCount // Silence unused warning

	// Verify callback was set
	session.mu.Lock()
	hasCallback := session.onWave != nil
	session.mu.Unlock()

	if !hasCallback {
		t.Error("Callback not set")
	}
}

func TestEndToEndSync(t *testing.T) {
	// Create two hosts
	h1, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host1: %v", err)
	}
	defer h1.Close()

	h2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host2: %v", err)
	}
	defer h2.Close()

	// Setup provider with test data
	provider := newMockWaveProvider()
	testHash := make([]byte, 32)
	for i := range testHash {
		testHash[i] = byte(i)
	}
	testWave := []byte("test wave data")
	provider.AddWave(testHash, testWave)
	provider.AddToTopic("/murmur/waves/1", testWave)

	// Setup sync handler on h1
	handler := NewSyncHandler(h1, provider)
	defer handler.Close()

	// Connect h2 to h1
	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), time.Hour)
	if err := h2.Connect(context.Background(), peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()}); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create client on h2
	client := NewSyncClient(h2)

	// Request by hash
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.RequestByHashes(ctx, h1.ID(), [][]byte{testHash})
	if err != nil {
		t.Fatalf("RequestByHashes() error = %v", err)
	}

	if resp.Status != StatusOK {
		t.Errorf("Response status = %d, want %d", resp.Status, StatusOK)
	}
	if len(resp.Waves) != 1 {
		t.Errorf("Expected 1 wave, got %d", len(resp.Waves))
	}
	if !bytes.Equal(resp.Waves[0], testWave) {
		t.Error("Wave data mismatch")
	}

	// Request by topic
	resp, err = client.RequestByTopic(ctx, h1.ID(), "/murmur/waves/1", 10)
	if err != nil {
		t.Fatalf("RequestByTopic() error = %v", err)
	}

	if resp.Status != StatusOK {
		t.Errorf("Response status = %d, want %d", resp.Status, StatusOK)
	}
	if len(resp.Waves) != 1 {
		t.Errorf("Expected 1 wave, got %d", len(resp.Waves))
	}

	// Request latest
	resp, err = client.RequestLatest(ctx, h1.ID(), 10)
	if err != nil {
		t.Fatalf("RequestLatest() error = %v", err)
	}

	if resp.Status != StatusOK {
		t.Errorf("Response status = %d, want %d", resp.Status, StatusOK)
	}
}

func TestSyncNotFound(t *testing.T) {
	h1, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host1: %v", err)
	}
	defer h1.Close()

	h2, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("Failed to create host2: %v", err)
	}
	defer h2.Close()

	// Empty provider
	provider := newMockWaveProvider()
	handler := NewSyncHandler(h1, provider)
	defer handler.Close()

	// Connect
	h2.Peerstore().AddAddrs(h1.ID(), h1.Addrs(), time.Hour)
	if err := h2.Connect(context.Background(), peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()}); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	client := NewSyncClient(h2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Request non-existent wave
	nonExistentHash := make([]byte, 32)
	_, err = client.RequestByHashes(ctx, h1.ID(), [][]byte{nonExistentHash})
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}
