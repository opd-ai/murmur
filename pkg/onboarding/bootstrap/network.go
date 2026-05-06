// Package bootstrap provides initial peer connection and first Wave prompt.
// Per ONBOARDING.md, bootstrap connects the new user to the network.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
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

	// InvitationURI is an optional invitation for warm-start bootstrap.
	// Format: murmur://invite/[Base64]. If provided, bootstrap will
	// prioritize connecting to the inviter's node.
	InvitationURI string
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
	if !m.tryStartBootstrap() {
		return nil // Already running or complete
	}

	ctx, m.cancel = context.WithTimeout(ctx, m.config.Timeout)
	m.initializeBootstrap()
	m.notifyStatusChange(StatusConnecting)

	if err := m.runBootstrapSequence(ctx); err != nil {
		return err
	}

	m.completeBootstrap()
	return nil
}

// tryStartBootstrap attempts to transition to bootstrap state. Returns false if already running.
func (m *Manager) tryStartBootstrap() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.status != StatusIdle && m.status != StatusFailed {
		return false
	}
	return true
}

// initializeBootstrap sets up initial bootstrap state.
func (m *Manager) initializeBootstrap() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startTime = time.Now()
	m.status = StatusConnecting
	m.errors = nil
}

// runBootstrapSequence executes the bootstrap phases.
func (m *Manager) runBootstrapSequence(ctx context.Context) error {
	// If an invitation was provided, prioritize connecting to the inviter first.
	// Per ROADMAP.md line 790, warm start pre-forms connection between inviter and invitee.
	if m.config.InvitationURI != "" {
		if err := m.connectToInviter(ctx); err != nil {
			// Log the error but continue with normal bootstrap.
			m.recordError(fmt.Errorf("inviter connection failed: %w", err))
		}
	}

	connectedCount := m.connectToBootstrapPeers(ctx)
	if connectedCount == 0 {
		m.setStatus(StatusFailed)
		return ErrNoBootstrapPeers
	}

	m.setStatus(StatusDiscovering)
	m.startPeerDiscovery(ctx)

	if err := m.waitForMinPeers(ctx); err != nil {
		m.setStatus(StatusFailed)
		return err
	}
	return nil
}

// startPeerDiscovery initiates DHT peer discovery.
func (m *Manager) startPeerDiscovery(ctx context.Context) {
	if m.connector == nil {
		return
	}
	if err := m.connector.StartDiscovery(ctx); err != nil {
		m.recordError(err)
	}
}

// completeBootstrap finalizes successful bootstrap.
func (m *Manager) completeBootstrap() {
	m.setStatus(StatusComplete)
	if m.callbacks.OnComplete != nil {
		peerCount := 0
		if m.connector != nil {
			peerCount = m.connector.PeerCount()
		}
		m.callbacks.OnComplete(peerCount)
	}
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
		done, err := m.waitForMinPeersIteration(ctx, ticker)
		if done {
			return err
		}
	}
}

// waitForMinPeersIteration handles one iteration of the peer wait loop.
// Returns (done, error) where done indicates whether the loop should exit.
func (m *Manager) waitForMinPeersIteration(ctx context.Context, ticker *time.Ticker) (bool, error) {
	select {
	case <-ctx.Done():
		return true, ErrTimeout
	case <-ticker.C:
		if m.hasMinPeers() {
			return true, nil
		}
		m.notifyProgress()
	}
	return false, nil
}

// hasMinPeers returns true if the minimum peer count has been reached.
func (m *Manager) hasMinPeers() bool {
	return m.connector.PeerCount() >= m.config.MinPeers
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

// connectToInviter attempts to connect to the inviter's node from an invitation.
// Per ROADMAP.md lines 789-790, invitations provide bootstrap advantage and warm start.
// Returns error if invitation decoding or connection fails.
func (m *Manager) connectToInviter(ctx context.Context) error {
	inv, err := AcceptInvitation(m.config.InvitationURI)
	if err != nil {
		return fmt.Errorf("accepting invitation: %w", err)
	}

	addr := BuildBootstrapAddrFromInvitation(inv)

	if err := m.connectWithRetries(ctx, addr); err != nil {
		return err
	}

	if m.callbacks.OnPeerConnected != nil {
		go m.callbacks.OnPeerConnected(inv.PeerID.String())
	}
	return nil
}

// connectWithRetries attempts connection with retries until success or context cancellation.
func (m *Manager) connectWithRetries(ctx context.Context, addr string) error {
	for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if m.attemptConnection(ctx, addr) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.config.RetryInterval):
		}
	}
	return fmt.Errorf("failed to connect to inviter after %d attempts", m.config.MaxRetries)
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
