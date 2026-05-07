//go:build test
// +build test

package screens

import (
	"testing"

	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

type peerSourceStub struct {
	handler func(peerID string)
}

func (p *peerSourceStub) SetOnPeerConnected(handler func(peerID string)) {
	p.handler = handler
}

func (p *peerSourceStub) emit(peerID string) {
	if p.handler != nil {
		p.handler(peerID)
	}
}

func TestBootstrapScreen_AttachPeerConnectedSource(t *testing.T) {
	controller := flow.NewController(flow.Callbacks{})
	controller.CompleteCurrentPhase() // Welcome -> Identity
	controller.CompleteCurrentPhase() // Identity -> Mode
	controller.CompleteCurrentPhase() // Mode -> NetworkBootstrap

	source := &peerSourceStub{}
	screen := NewBootstrapScreenWithPeerSource(controller, BootstrapScreenCallbacks{}, source)

	for i := 0; i < 6; i++ {
		source.emit("peer")
	}

	if got := screen.PeersFound(); got != 6 {
		t.Fatalf("expected 6 peers from forwarded events, got %d", got)
	}
	if !screen.IsDiscoveryDone() {
		t.Fatal("expected discoveryDone=true after reaching target peers via forwarded events")
	}
}
