//go:build js && wasm

package network

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

const (
	wasmDataChannelLabel = "murmur-data"
	wasmOpenTimeout      = 10 * time.Second
)

type wasmAdapter struct {
	cfg Config

	mu sync.RWMutex

	api      *webrtc.API
	offerPC  *webrtc.PeerConnection
	answerPC *webrtc.PeerConnection
	dataCh   *webrtc.DataChannel

	runCtx context.Context
	cancel context.CancelFunc

	subs map[string]chan Message

	discoveryPeers []string
	nextPeer       int
	dialedPeers    map[string]struct{}
}

func newWASMAdapter(cfg Config) (Adapter, error) {
	return &wasmAdapter{
		cfg:         cfg,
		subs:        make(map[string]chan Message),
		dialedPeers: make(map[string]struct{}),
	}, nil
}

func (a *wasmAdapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dataCh != nil || a.offerPC != nil || a.answerPC != nil {
		return ErrAlreadyStarted
	}

	a.runCtx, a.cancel = context.WithCancel(ctx)
	a.api = webrtc.NewAPI()

	offerPC, answerPC, dataCh, err := a.createLoopbackPeerPair(a.runCtx)
	if err != nil {
		a.cancel()
		a.cancel = nil
		a.runCtx = nil
		a.api = nil
		return err
	}

	a.offerPC = offerPC
	a.answerPC = answerPC
	a.dataCh = dataCh
	a.discoveryPeers = buildBrowserDiscoveryPeers(a.cfg.RelayPeers, a.cfg.BootstrapPeers)

	return nil
}

func (a *wasmAdapter) Stop(context.Context) error {
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

	if a.dataCh != nil {
		_ = a.dataCh.Close()
		a.dataCh = nil
	}

	if a.offerPC != nil {
		if err := a.offerPC.Close(); err != nil {
			return fmt.Errorf("closing wasm offer peer connection: %w", err)
		}
		a.offerPC = nil
	}

	if a.answerPC != nil {
		if err := a.answerPC.Close(); err != nil {
			return fmt.Errorf("closing wasm answer peer connection: %w", err)
		}
		a.answerPC = nil
	}

	a.runCtx = nil
	a.api = nil
	a.discoveryPeers = nil
	a.nextPeer = 0
	a.dialedPeers = make(map[string]struct{})

	return nil
}

func (a *wasmAdapter) Publish(_ context.Context, topic string, payload []byte) error {
	a.mu.RLock()
	dc := a.dataCh
	from := a.cfg.PeerID
	a.mu.RUnlock()

	if dc == nil {
		return ErrNotStarted
	}
	if from == "" {
		from = "wasm-local"
	}

	encoded, err := json.Marshal(dataChannelEnvelope{Topic: topic, From: from, Payload: payload})
	if err != nil {
		return fmt.Errorf("encoding wasm data-channel message: %w", err)
	}

	if err := dc.Send(encoded); err != nil {
		return fmt.Errorf("sending wasm data-channel message: %w", err)
	}

	return nil
}

func (a *wasmAdapter) Subscribe(topic string) (<-chan Message, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if ch, ok := a.subs[topic]; ok {
		return ch, nil
	}
	if a.dataCh == nil {
		return nil, ErrNotStarted
	}

	ch := make(chan Message, 64)
	a.subs[topic] = ch
	return ch, nil
}

func (a *wasmAdapter) DialPeer(_ context.Context, peerAddr string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dataCh == nil {
		return ErrNotStarted
	}

	if peerAddr == "" {
		next, err := a.selectNextDiscoveryPeer()
		if err != nil {
			return err
		}
		peerAddr = next
	}

	if !a.isAllowedDiscoveryPeer(peerAddr) {
		return fmt.Errorf("peer %q is not in browser relay/bootstrap discovery set", peerAddr)
	}

	a.dialedPeers[peerAddr] = struct{}{}

	// In-browser peer signaling and connection establishment follows this selection policy.
	// This adapter stage records and validates relay/bootstrap-only targets (no mDNS paths).
	return nil
}

func (a *wasmAdapter) Name() string {
	return "pion-webrtc-wasm"
}

type dataChannelEnvelope struct {
	Topic   string `json:"topic"`
	From    string `json:"from"`
	Payload []byte `json:"payload"`
}

func (a *wasmAdapter) createLoopbackPeerPair(ctx context.Context) (*webrtc.PeerConnection, *webrtc.PeerConnection, *webrtc.DataChannel, error) {
	config := webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: a.stunServers()}}}

	offerPC, err := a.api.NewPeerConnection(config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating wasm offer peer connection: %w", err)
	}

	answerPC, err := a.api.NewPeerConnection(config)
	if err != nil {
		_ = offerPC.Close()
		return nil, nil, nil, fmt.Errorf("creating wasm answer peer connection: %w", err)
	}

	offerPC.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		_ = answerPC.AddICECandidate(c.ToJSON())
	})
	answerPC.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		_ = offerPC.AddICECandidate(c.ToJSON())
	})

	answerReady := make(chan *webrtc.DataChannel, 1)
	answerPC.OnDataChannel(func(dc *webrtc.DataChannel) {
		select {
		case answerReady <- dc:
		default:
		}
	})

	dataCh, err := offerPC.CreateDataChannel(wasmDataChannelLabel, nil)
	if err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("creating wasm data channel: %w", err)
	}

	openReady := make(chan struct{}, 1)
	dataCh.OnOpen(func() {
		select {
		case openReady <- struct{}{}:
		default:
		}
	})

	offer, err := offerPC.CreateOffer(nil)
	if err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("creating wasm offer: %w", err)
	}
	if err := offerPC.SetLocalDescription(offer); err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("setting wasm offer local description: %w", err)
	}
	if err := answerPC.SetRemoteDescription(offer); err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("setting wasm offer remote description: %w", err)
	}

	answer, err := answerPC.CreateAnswer(nil)
	if err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("creating wasm answer: %w", err)
	}
	if err := answerPC.SetLocalDescription(answer); err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("setting wasm answer local description: %w", err)
	}
	if err := offerPC.SetRemoteDescription(answer); err != nil {
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("setting wasm answer remote description: %w", err)
	}

	var answerDC *webrtc.DataChannel
	select {
	case answerDC = <-answerReady:
	case <-time.After(wasmOpenTimeout):
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("timed out waiting for wasm answer data channel")
	case <-ctx.Done():
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, ctx.Err()
	}

	answerDC.OnMessage(func(msg webrtc.DataChannelMessage) {
		a.handleInbound(msg.Data)
	})

	select {
	case <-openReady:
	case <-time.After(wasmOpenTimeout):
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, fmt.Errorf("timed out waiting for wasm data channel open")
	case <-ctx.Done():
		_ = offerPC.Close()
		_ = answerPC.Close()
		return nil, nil, nil, ctx.Err()
	}

	return offerPC, answerPC, dataCh, nil
}

func (a *wasmAdapter) handleInbound(raw []byte) {
	var envelope dataChannelEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}

	a.mu.RLock()
	ch := a.subs[envelope.Topic]
	a.mu.RUnlock()
	if ch == nil {
		return
	}

	select {
	case ch <- Message{Topic: envelope.Topic, From: envelope.From, Payload: envelope.Payload}:
	default:
	}
}

func (a *wasmAdapter) stunServers() []string {
	if len(a.cfg.STUNServers) != 0 {
		return append([]string(nil), a.cfg.STUNServers...)
	}
	return []string{"stun:stun.l.google.com:19302"}
}

func (a *wasmAdapter) isAllowedDiscoveryPeer(peerAddr string) bool {
	if len(a.discoveryPeers) == 0 {
		return false
	}
	for _, allowed := range a.discoveryPeers {
		if peerAddr == allowed {
			return true
		}
	}
	return false
}

func (a *wasmAdapter) selectNextDiscoveryPeer() (string, error) {
	if len(a.discoveryPeers) == 0 {
		return "", fmt.Errorf("no browser relay/bootstrap peers configured")
	}

	peerAddr := a.discoveryPeers[a.nextPeer%len(a.discoveryPeers)]
	a.nextPeer = (a.nextPeer + 1) % len(a.discoveryPeers)
	return peerAddr, nil
}
