// Package onramp provides shared utilities for onramp-based transports (I2P and Tor).
package onramp

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/transport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
)

// upgradeConnection wraps a net.Conn, applies resource management, and upgrades it
// to a transport.CapableConn. This consolidates the identical upgrade logic used by
// both I2P and Tor transports after obtaining a raw connection.
func UpgradeConnection(
	ctx context.Context,
	rawConn net.Conn,
	t transport.Transport,
	upgrader transport.Upgrader,
	rcmgr network.ResourceManager,
	raddr ma.Multiaddr,
	p peer.ID,
) (transport.CapableConn, error) {
	maConn, err := manet.WrapNetConn(rawConn)
	if err != nil {
		rawConn.Close()
		return nil, fmt.Errorf("failed to wrap connection: %w", err)
	}

	scope, err := rcmgr.OpenConnection(network.DirOutbound, false, raddr)
	if err != nil {
		maConn.Close()
		return nil, fmt.Errorf("resource manager rejected connection: %w", err)
	}

	capableConn, err := upgrader.Upgrade(ctx, t, maConn, network.DirOutbound, p, scope)
	if err != nil {
		scope.Done()
		maConn.Close()
		return nil, fmt.Errorf("upgrade failed: %w", err)
	}

	return capableConn, nil
}

// upgradeListener wraps a net.Listener, converts it to a multiaddr listener,
// gates and upgrades it. This consolidates the identical upgrade logic used by
// both I2P and Tor transports after obtaining a raw listener.
func UpgradeListener(
	netListener net.Listener,
	listenerMultiaddr ma.Multiaddr,
	t transport.Transport,
	upgrader transport.Upgrader,
) (transport.Listener, error) {
	maListener, err := manet.WrapNetListener(netListener)
	if err != nil {
		netListener.Close()
		return nil, fmt.Errorf("failed to wrap listener: %w", err)
	}

	gatedListener := upgrader.GateMaListener(maListener)
	upgradedListener := upgrader.UpgradeGatedMaListener(t, gatedListener)

	return &Listener{
		Listener:  upgradedListener,
		multiaddr: listenerMultiaddr,
	}, nil
}

// Listener wraps a libp2p Listener with the correct multiaddr.
// Shared by both I2P and Tor transports.
type Listener struct {
	transport.Listener
	multiaddr ma.Multiaddr
}

func (l *Listener) Multiaddr() ma.Multiaddr {
	return l.multiaddr
}

// safeClose is a thread-safe idempotent close helper for transport implementations.
// It ensures the transport's closed flag is set atomically and the underlying closer
// is called exactly once. Returns nil if already closed.
func SafeClose(mu *sync.Mutex, closed *bool, closer io.Closer) error {
	mu.Lock()
	defer mu.Unlock()

	if *closed {
		return nil
	}
	*closed = true

	if closer != nil {
		return closer.Close()
	}
	return nil
}
