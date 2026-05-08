// Package screens provides test stubs for Completion screen.
//
//go:build test
// +build test

package screens

import (
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/opd-ai/murmur/pkg/identity"
	"github.com/opd-ai/murmur/pkg/identity/keys"
	"github.com/opd-ai/murmur/pkg/identity/modes"
)

// CompletionScreenState tracks completion sub-screens.
type CompletionScreenState int

const (
	CompletionStateSummary CompletionScreenState = iota
	CompletionStateInvite
	CompletionStateDone
)

// CompletionScreen handles Phase 6 (stub for testing).
type CompletionScreen struct {
	state     CompletionScreenState
	startTime time.Time

	// Identity summary
	displayName    string
	surfaceKeypair *keys.KeyPair
	specterName    string
	selectedMode   modes.Mode
	peersConnected int

	// Invitation
	peerID          peer.ID
	inviteGenerated bool
	inviteCode      string

	callbacks CompletionScreenCallbacks
}

// CompletionScreenCallbacks provides hooks for completion events.
type CompletionScreenCallbacks struct {
	OnInviteGenerated  func(code string)
	OnOnboardingFinish func()
}

// NewCompletionScreen creates a new completion screen (stub).
func NewCompletionScreen(
	displayName string,
	surfaceSigil interface{}, // *ebiten.Image in real build
	surfaceKeypair *keys.KeyPair,
	specterName string,
	specterSigil interface{}, // *ebiten.Image in real build
	selectedMode modes.Mode,
	peersConnected int,
	callbacks CompletionScreenCallbacks,
) *CompletionScreen {
	return &CompletionScreen{
		state:          CompletionStateSummary,
		startTime:      time.Now(),
		displayName:    displayName,
		surfaceKeypair: surfaceKeypair,
		specterName:    specterName,
		selectedMode:   selectedMode,
		peersConnected: peersConnected,
		callbacks:      callbacks,
	}
}

// SetPeerID provides the local node's peer ID for proper invitation generation.
func (s *CompletionScreen) SetPeerID(id peer.ID) {
	s.peerID = id
}

// Update advances animations (stub - no-op).
func (s *CompletionScreen) Update() error {
	return nil
}

// CompletionState returns the current screen state.
func (s *CompletionScreen) CompletionState() CompletionScreenState {
	return s.state
}

// InviteCode returns the generated invite code.
func (s *CompletionScreen) InviteCode() string {
	return s.inviteCode
}

// IsInviteGenerated returns whether an invite was generated.
func (s *CompletionScreen) IsInviteGenerated() bool {
	return s.inviteGenerated
}

// --- Simulation Methods for Testing ---

// SimulateGoToInvite navigates to the invite screen.
func (s *CompletionScreen) SimulateGoToInvite() {
	s.state = CompletionStateInvite
}

// SimulateGenerateInvite simulates invite code generation using the proper
// identity.GenerateInvitation path, producing a real murmur:// URI.
func (s *CompletionScreen) SimulateGenerateInvite() {
	if s.surfaceKeypair == nil {
		s.inviteCode = "MURMUR-XXXX-YYYY"
		s.inviteGenerated = true
		s.notifyInviteGenerated()
		return
	}

	peerID := s.peerID
	if peerID == "" {
		peerID = deriveStubPeerID(s.surfaceKeypair.PublicKey)
	}

	inv, err := identity.GenerateInvitation(peerID, s.surfaceKeypair.PublicKey, "")
	if err == nil {
		if uri, err := inv.EncodeURI(); err == nil {
			s.inviteCode = uri
			s.inviteGenerated = true
			s.notifyInviteGenerated()
			return
		}
	}

	// Fallback to legacy format.
	s.inviteCode = generateInviteCode(s.surfaceKeypair.PublicKey)
	s.inviteGenerated = true
	s.notifyInviteGenerated()
}

// deriveStubPeerID derives a peer.ID from an Ed25519 public key for testing.
func deriveStubPeerID(pubKey []byte) peer.ID {
	libp2pPub, err := libp2pcrypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return ""
	}
	id, err := peer.IDFromPublicKey(libp2pPub)
	if err != nil {
		return ""
	}
	return id
}

// notifyInviteGenerated fires the OnInviteGenerated callback if registered.
func (s *CompletionScreen) notifyInviteGenerated() {
	if s.callbacks.OnInviteGenerated != nil {
		s.callbacks.OnInviteGenerated(s.inviteCode)
	}
}

// SimulateContinueToComplete navigates to the completion screen.
func (s *CompletionScreen) SimulateContinueToComplete() {
	s.state = CompletionStateDone
}

// SimulateFinish completes onboarding.
func (s *CompletionScreen) SimulateFinish() {
	if s.callbacks.OnOnboardingFinish != nil {
		s.callbacks.OnOnboardingFinish()
	}
}

func generateInviteCode(pubKey []byte) string {
	if len(pubKey) < 6 {
		return "MURMUR-XXXX-YYYY"
	}
	return "MURMUR-" + hexNibble(pubKey[0]) + hexNibble(pubKey[1]) + hexNibble(pubKey[2]) +
		"-" + hexNibble(pubKey[3]) + hexNibble(pubKey[4]) + hexNibble(pubKey[5])
}

func hexNibble(b byte) string {
	const hex = "0123456789ABCDEF"
	return string(hex[(b>>4)&0x0F]) + string(hex[b&0x0F])
}
