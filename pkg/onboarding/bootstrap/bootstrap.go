// Package bootstrap provides initial peer connection and first Wave prompt.
// Per ONBOARDING.md, bootstrap connects the new user to the network.
package bootstrap

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrBootstrapFailed indicates bootstrap could not complete.
var ErrBootstrapFailed = errors.New("bootstrap failed")

// ErrNoBootstrapPeers indicates no bootstrap peers were reachable.
var ErrNoBootstrapPeers = errors.New("no bootstrap peers reachable")

// ErrTimeout indicates bootstrap timed out.
var ErrTimeout = errors.New("bootstrap timeout")

// Status represents the current bootstrap state.
type Status int

const (
	StatusIdle Status = iota
	StatusConnecting
	StatusDiscovering
	StatusSyncing
	StatusComplete
	StatusFailed
)

// String returns a human-readable status.
func (s Status) String() string {
	switch s {
	case StatusIdle:
		return "Idle"
	case StatusConnecting:
		return "Connecting"
	case StatusDiscovering:
		return "Discovering Peers"
	case StatusSyncing:
		return "Syncing"
	case StatusComplete:
		return "Complete"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// Config holds bootstrap configuration.
type Config struct {
	// BootstrapPeers is the list of initial peer addresses to connect to.
	BootstrapPeers []string

	// MinPeers is the minimum number of peers before bootstrap is complete.
	MinPeers int

	// Timeout is the maximum time to spend bootstrapping.
	Timeout time.Duration

	// RetryInterval is the time between connection attempts.
	RetryInterval time.Duration

	// MaxRetries is the maximum number of retry attempts per peer.
	MaxRetries int
}

// DefaultConfig returns sensible default bootstrap configuration.
func DefaultConfig() Config {
	return Config{
		BootstrapPeers: []string{
			// Per ROADMAP.md, actual bootstrap nodes TBD
			"/dnsaddr/bootstrap1.murmur.network/p2p/QmPlaceholder1",
			"/dnsaddr/bootstrap2.murmur.network/p2p/QmPlaceholder2",
		},
		MinPeers:      3,
		Timeout:       60 * time.Second,
		RetryInterval: 5 * time.Second,
		MaxRetries:    3,
	}
}

// Progress represents bootstrap progress for UI display.
type Progress struct {
	Status         Status
	ConnectedPeers int
	TargetPeers    int
	AttemptedPeers int
	Errors         []error
	ElapsedTime    time.Duration
	RemainingTime  time.Duration
	Message        string
}

// Callbacks provides hooks for bootstrap events.
type Callbacks struct {
	OnStatusChange  func(status Status)
	OnPeerConnected func(peerID string)
	OnProgress      func(progress Progress)
	OnComplete      func(peerCount int)
	OnError         func(err error)
}

// NetworkConnector abstracts the network connection functionality.
type NetworkConnector interface {
	// Connect attempts to connect to a peer address.
	Connect(ctx context.Context, addr string) (string, error)

	// PeerCount returns the current number of connected peers.
	PeerCount() int

	// StartDiscovery begins peer discovery via DHT.
	StartDiscovery(ctx context.Context) error
}

// Manager coordinates the bootstrap process.
type Manager struct {
	mu        sync.RWMutex
	config    Config
	connector NetworkConnector
	callbacks Callbacks
	status    Status
	startTime time.Time
	errors    []error
	cancel    context.CancelFunc
}

// NewManager creates a new bootstrap manager.
func NewManager(cfg Config, connector NetworkConnector, cb Callbacks) *Manager {
	return &Manager{
		config:    cfg,
		connector: connector,
		callbacks: cb,
		status:    StatusIdle,
		errors:    make([]error, 0),
	}
}

// Start begins the bootstrap process.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.status != StatusIdle && m.status != StatusFailed {
		m.mu.Unlock()
		return nil // Already running or complete
	}

	ctx, m.cancel = context.WithTimeout(ctx, m.config.Timeout)
	m.startTime = time.Now()
	m.status = StatusConnecting
	m.errors = nil
	m.mu.Unlock()

	m.notifyStatusChange(StatusConnecting)

	// Connect to bootstrap peers
	connectedCount := m.connectToBootstrapPeers(ctx)

	if connectedCount == 0 {
		m.setStatus(StatusFailed)
		return ErrNoBootstrapPeers
	}

	// Start peer discovery
	m.setStatus(StatusDiscovering)
	if m.connector != nil {
		if err := m.connector.StartDiscovery(ctx); err != nil {
			m.recordError(err)
		}
	}

	// Wait for minimum peers
	if err := m.waitForMinPeers(ctx); err != nil {
		m.setStatus(StatusFailed)
		return err
	}

	m.setStatus(StatusComplete)
	if m.callbacks.OnComplete != nil {
		peerCount := 0
		if m.connector != nil {
			peerCount = m.connector.PeerCount()
		}
		m.callbacks.OnComplete(peerCount)
	}

	return nil
}

// Stop cancels an in-progress bootstrap.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// Status returns the current bootstrap status.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Progress returns current progress information.
func (m *Manager) Progress() Progress {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peerCount := 0
	if m.connector != nil {
		peerCount = m.connector.PeerCount()
	}

	elapsed := time.Duration(0)
	if !m.startTime.IsZero() {
		elapsed = time.Since(m.startTime)
	}

	remaining := m.config.Timeout - elapsed
	if remaining < 0 {
		remaining = 0
	}

	return Progress{
		Status:         m.status,
		ConnectedPeers: peerCount,
		TargetPeers:    m.config.MinPeers,
		AttemptedPeers: len(m.config.BootstrapPeers),
		Errors:         m.errors,
		ElapsedTime:    elapsed,
		RemainingTime:  remaining,
		Message:        m.status.String(),
	}
}

// connectToBootstrapPeers attempts to connect to configured bootstrap peers.
func (m *Manager) connectToBootstrapPeers(ctx context.Context) int {
	if m.connector == nil {
		return 0
	}

	var connected int
	for _, addr := range m.config.BootstrapPeers {
		if ctx.Err() != nil {
			break
		}
		if m.tryConnectPeer(ctx, addr) {
			connected++
		}
	}
	return connected
}

// tryConnectPeer attempts to connect to a single peer with retries.
func (m *Manager) tryConnectPeer(ctx context.Context, addr string) bool {
	for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return false
		}
		if m.attemptConnection(ctx, addr) {
			return true
		}
		time.Sleep(m.config.RetryInterval)
	}
	return false
}

// attemptConnection makes a single connection attempt.
func (m *Manager) attemptConnection(ctx context.Context, addr string) bool {
	peerID, err := m.connector.Connect(ctx, addr)
	if err != nil {
		m.recordError(err)
		return false
	}
	if m.callbacks.OnPeerConnected != nil {
		m.callbacks.OnPeerConnected(peerID)
	}
	m.notifyProgress()
	return true
}

// waitForMinPeers waits until minimum peer count is reached.
func (m *Manager) waitForMinPeers(ctx context.Context) error {
	if m.connector == nil {
		return nil
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ErrTimeout
		case <-ticker.C:
			if m.connector.PeerCount() >= m.config.MinPeers {
				return nil
			}
			m.notifyProgress()
		}
	}
}

// setStatus updates the status and notifies.
func (m *Manager) setStatus(status Status) {
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
	m.notifyStatusChange(status)
}

// notifyStatusChange sends status change notification.
func (m *Manager) notifyStatusChange(status Status) {
	if m.callbacks.OnStatusChange != nil {
		go m.callbacks.OnStatusChange(status)
	}
}

// notifyProgress sends progress notification.
func (m *Manager) notifyProgress() {
	if m.callbacks.OnProgress != nil {
		go m.callbacks.OnProgress(m.Progress())
	}
}

// recordError records an error and notifies.
func (m *Manager) recordError(err error) {
	m.mu.Lock()
	m.errors = append(m.errors, err)
	m.mu.Unlock()

	if m.callbacks.OnError != nil {
		go m.callbacks.OnError(err)
	}
}

// FirstWavePrompt contains data for prompting the user's first Wave.
type FirstWavePrompt struct {
	SuggestedTopics []string
	Examples        []string
	MaxLength       int
	Hint            string
}

// GetFirstWavePrompt returns content for the first Wave creation UI.
func GetFirstWavePrompt() FirstWavePrompt {
	return FirstWavePrompt{
		SuggestedTopics: []string{
			"Introduce yourself",
			"Share what brought you here",
			"Ask a question",
			"Say hello to the network",
		},
		Examples: []string{
			"👋 Hello MURMUR! Excited to explore a decentralized network.",
			"Curious about privacy-first social networking. What should I know?",
			"Just arrived. Looking forward to connecting with like-minded folks.",
		},
		MaxLength: 2048,
		Hint:      "Your first Wave will ripple through the network. What would you like to share?",
	}
}
