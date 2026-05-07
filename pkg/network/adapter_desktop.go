//go:build !js

package network

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/opd-ai/murmur/pkg/networking/gossip"
	murmurtransport "github.com/opd-ai/murmur/pkg/networking/transport"
)

type desktopAdapter struct {
	cfg Config

	mu     sync.RWMutex
	host   *murmurtransport.Host
	pubsub *gossip.PubSub

	runCtx context.Context
	cancel context.CancelFunc

	subs map[string]chan Message
}

func newDesktopAdapter(cfg Config) (Adapter, error) {
	return &desktopAdapter{
		cfg:  cfg,
		subs: make(map[string]chan Message),
	}, nil
}

func (a *desktopAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.host != nil || a.pubsub != nil {
		return ErrAlreadyStarted
	}

	hostCfg, err := a.buildTransportConfig()
	if err != nil {
		return err
	}

	a.runCtx, a.cancel = context.WithCancel(ctx)

	host, err := murmurtransport.NewHost(a.runCtx, hostCfg)
	if err != nil {
		a.cancel()
		a.cancel = nil
		return fmt.Errorf("starting desktop transport host: %w", err)
	}

	ps, err := gossip.New(a.runCtx, host.Host)
	if err != nil {
		_ = host.Close()
		a.cancel()
		a.cancel = nil
		return fmt.Errorf("starting desktop pubsub: %w", err)
	}

	a.host = host
	a.pubsub = ps
	return nil
}


func (a *desktopAdapter) Stop(context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	for topic, ch := range a.subs {
		close(ch)
		delete(a.subs, topic)
	}

	if a.pubsub != nil {
		if err := a.pubsub.Close(); err != nil {
			return fmt.Errorf("closing desktop pubsub: %w", err)
		}
		a.pubsub = nil
	}

	if a.host != nil {
		if err := a.host.Close(); err != nil {
			return fmt.Errorf("closing desktop transport host: %w", err)
		}
		a.host = nil
	}

	return nil
}

func (a *desktopAdapter) Publish(ctx context.Context, topic string, payload []byte) error {
	a.mu.RLock()
	pubsub := a.pubsub
	a.mu.RUnlock()

	if pubsub == nil {
		return ErrNotStarted
	}

	return pubsub.Publish(ctx, topic, payload)
}

func (a *desktopAdapter) Subscribe(topic string) (<-chan Message, error) {
	a.mu.Lock()
	if ch, ok := a.subs[topic]; ok {
		a.mu.Unlock()
		return ch, nil
	}

	pubsub := a.pubsub
	runCtx := a.runCtx
	if pubsub == nil || runCtx == nil {
		a.mu.Unlock()
		return nil, ErrNotStarted
	}

	ch := make(chan Message, 64)
	a.subs[topic] = ch
	a.mu.Unlock()

	err := pubsub.Subscribe(runCtx, topic, func(ctx context.Context, msg *gossippubsubMessage) {
		select {
		case ch <- Message{Topic: topic, From: msg.From, Payload: msg.Payload}:
		default:
		}
	})
	if err != nil {
		a.mu.Lock()
		delete(a.subs, topic)
		close(ch)
		a.mu.Unlock()
		return nil, err
	}

	return ch, nil
}

func (a *desktopAdapter) DialPeer(ctx context.Context, peerAddr string) error {
	a.mu.RLock()
	h := a.host
	a.mu.RUnlock()

	if h == nil {
		return ErrNotStarted
	}

	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("parsing peer address %q: %w", peerAddr, err)
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("converting peer address to addr info: %w", err)
	}

	return h.Connect(ctx, *info)
}

func (a *desktopAdapter) Name() string {
	return "libp2p-desktop"
}

func (a *desktopAdapter) buildTransportConfig() (murmurtransport.Config, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return murmurtransport.Config{}, fmt.Errorf("generating desktop adapter private key: %w", err)
	}

	cfg := murmurtransport.DefaultConfig()
	cfg.PrivateKey = privateKey

	bootstrapPeers, err := parseBootstrapPeers(a.cfg.BootstrapPeers)
	if err != nil {
		return murmurtransport.Config{}, err
	}
	cfg.BootstrapPeers = bootstrapPeers

	return cfg, nil
}

func parseBootstrapPeers(addrs []string) ([]peer.AddrInfo, error) {
	infos := make([]peer.AddrInfo, 0, len(addrs))
	for _, raw := range addrs {
		addr, err := multiaddr.NewMultiaddr(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap address %q: %w", raw, err)
		}
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap p2p address %q: %w", raw, err)
		}
		infos = append(infos, *info)
	}
	return infos, nil
}

type gossippubsubMessage struct {
	From    string
	Payload []byte
}
