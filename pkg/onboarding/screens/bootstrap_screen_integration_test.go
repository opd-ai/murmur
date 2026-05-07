//go:build test
// +build test

package screens

import (
	"context"
	"fmt"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	bootstrap "github.com/opd-ai/murmur/pkg/onboarding/bootstrap"
	"github.com/opd-ai/murmur/pkg/onboarding/flow"
)

// bootstrapLibp2pConnector adapts a real in-process libp2p host to the
// bootstrap.NetworkConnector interface.
type bootstrapLibp2pConnector struct {
	host host.Host
}

func (c *bootstrapLibp2pConnector) Connect(ctx context.Context, addr string) (string, error) {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return "", err
	}
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return "", err
	}
	if err := c.host.Connect(ctx, *info); err != nil {
		return "", err
	}
	return info.ID.String(), nil
}

func (c *bootstrapLibp2pConnector) PeerCount() int {
	return len(c.host.Network().Peers())
}

func (c *bootstrapLibp2pConnector) StartDiscovery(ctx context.Context) error {
	return nil
}

func TestBootstrapScreen_DiscoveryDoneOnLibp2pPeerConnected(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	initiator, err := libp2p.New()
	if err != nil {
		t.Fatalf("libp2p.New initiator: %v", err)
	}
	defer initiator.Close()

	bootstrapNode, err := libp2p.New()
	if err != nil {
		t.Fatalf("libp2p.New bootstrap node: %v", err)
	}
	defer bootstrapNode.Close()

	if len(bootstrapNode.Addrs()) == 0 {
		t.Fatal("bootstrap node has no listen addresses")
	}
	bootstrapAddr := fmt.Sprintf("%s/p2p/%s", bootstrapNode.Addrs()[0].String(), bootstrapNode.ID().String())

	cfg := bootstrap.Config{
		BootstrapPeers: []string{bootstrapAddr},
		MinPeers:       1,
		Timeout:        3 * time.Second,
		RetryInterval:  50 * time.Millisecond,
		MaxRetries:     1,
	}
	manager := bootstrap.NewManager(cfg, &bootstrapLibp2pConnector{host: initiator}, bootstrap.Callbacks{})

	controller := flow.NewController(flow.Callbacks{})
	controller.CompleteCurrentPhase() // Welcome -> Identity
	controller.CompleteCurrentPhase() // Identity -> Mode
	controller.CompleteCurrentPhase() // Mode -> NetworkBootstrap

	screen := NewBootstrapScreenWithPeerSource(controller, BootstrapScreenCallbacks{}, manager)
	screen.targetPeers = 1

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.Start(ctx)
	}()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(3 * time.Second)

	for !screen.IsDiscoveryDone() {
		select {
		case <-ticker.C:
		case err := <-errCh:
			if err != nil {
				t.Fatalf("manager.Start() error: %v", err)
			}
			if screen.IsDiscoveryDone() {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for discoveryDone=true after libp2p peer connection")
		}
	}
}
