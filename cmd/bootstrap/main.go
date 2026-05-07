// Package main provides a standalone bootstrap/reseed host binary.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-i2p/onramp"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/opd-ai/murmur/pkg/networking/discovery"
	transportpkg "github.com/opd-ai/murmur/pkg/networking/transport"
)

// Version is the current version of MURMUR. Set by build flags.
var Version = "0.0.0-alpha"

// Commit is the git commit hash. Set by build flags.
var Commit = "unknown"

const (
	defaultStateDir       = "./bootstrap-data"
	defaultP2PListenAddrs = "/ip4/0.0.0.0/tcp/4001,/ip4/0.0.0.0/udp/4001/quic-v1"
	defaultPeerLimit      = 64
	defaultPeerMaxAge     = 24 * time.Hour
	peerScanInterval      = 30 * time.Second
	identityKeyFileName   = "bootstrap_identity.key"
)

type appConfig struct {
	listenAddr     string
	stateDir       string
	p2pListenAddrs string
	announceAddrs  string
	peerMaxAge     time.Duration
	peerLimit      int

	ngrokEnabled bool
	ngrokDomain  string

	torEnabled bool
	torName    string
	torPort    string

	i2pEnabled bool
	i2pName    string
	i2pSAMAddr string

	allowOrigin string
}

type managedListener struct {
	name      string
	listener  net.Listener
	closeFunc func() error
}

type healthResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	Timestamp    int64  `json:"timestamp"`
	PeerID       string `json:"peer_id"`
	KnownPeers   int    `json:"known_peers"`
	HTTPListen   string `json:"http_listen"`
	StateDir     string `json:"state_dir"`
	P2PListeners int    `json:"p2p_listeners"`
}

type signedPeerSource interface {
	SignedPeerList() (*discovery.SignedPeerList, error)
	HealthInfo() healthResponse
}

type bootstrapRuntime struct {
	host          *transportpkg.Host
	pex           *discovery.PEX
	tracker       *peerTracker
	signKey       ed25519.PrivateKey
	announceAddrs []multiaddr.Multiaddr
	peerMaxAge    time.Duration
	httpListen    string
	stateDir      string
}

type peerTracker struct {
	mu         sync.RWMutex
	peers      map[peer.ID]trackedPeer
	peerMaxAge time.Duration
	peerLimit  int
}

type trackedPeer struct {
	addrs map[string]multiaddr.Multiaddr
	seen  time.Time
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := validateConfig(cfg); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := startBootstrapRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer runtime.Close()

	handler := newHandler(runtime, cfg.allowOrigin)

	listeners, err := buildListeners(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeManagedListeners(listeners)

	for _, l := range listeners {
		log.Printf("bootstrap HTTP listener active: %s (%s)", l.name, l.listener.Addr())
	}
	log.Printf("bootstrap peer ID: %s", runtime.host.PeerID())
	for _, addr := range runtime.advertisedAddrInfo().Addrs {
		log.Printf("bootstrap peer address: %s/p2p/%s", addr, runtime.host.PeerID())
	}

	return serveUntilShutdown(ctx, listeners, handler)
}

func parseFlags() appConfig {
	var cfg appConfig

	flag.StringVar(&cfg.listenAddr, "listen", ":8081", "HTTP listen address for bootstrap bundle serving")
	flag.StringVar(&cfg.stateDir, "state-dir", defaultStateDir, "Directory for bootstrap runtime state and identity key")
	flag.StringVar(&cfg.p2pListenAddrs, "p2p-listen", defaultP2PListenAddrs, "Comma-separated libp2p listen multiaddrs for DHT/bootstrap participation")
	flag.StringVar(&cfg.announceAddrs, "announce-addrs", "", "Optional comma-separated public multiaddrs to advertise instead of local listen addrs")
	flag.DurationVar(&cfg.peerMaxAge, "peer-max-age", defaultPeerMaxAge, "Maximum age for peers retained in the distributed bundle")
	flag.IntVar(&cfg.peerLimit, "peer-limit", defaultPeerLimit, "Maximum number of peers to distribute to new clients")
	flag.StringVar(&cfg.allowOrigin, "allow-origin", "*", "Value for Access-Control-Allow-Origin header")

	flag.BoolVar(&cfg.ngrokEnabled, "ngrok", false, "Enable ngrok HTTP listener")
	flag.StringVar(&cfg.ngrokDomain, "ngrok-domain", "", "ngrok custom domain for the bootstrap HTTP endpoint (optional)")

	flag.BoolVar(&cfg.torEnabled, "tor", false, "Enable Tor hidden-service HTTP listener")
	flag.StringVar(&cfg.torName, "tor-name", "murmur-bootstrap", "Persistent Tor instance name")
	flag.StringVar(&cfg.torPort, "tor-port", "8081", "Tor hidden-service virtual port for the HTTP endpoint")

	flag.BoolVar(&cfg.i2pEnabled, "i2p", false, "Enable I2P HTTP listener")
	flag.StringVar(&cfg.i2pName, "i2p-name", "murmur-bootstrap", "Persistent I2P tunnel name")
	flag.StringVar(&cfg.i2pSAMAddr, "i2p-sam", "127.0.0.1:7656", "I2P SAMv3 bridge address")

	flag.Parse()
	return cfg
}

func validateConfig(cfg appConfig) error {
	if strings.TrimSpace(cfg.stateDir) == "" {
		return errors.New("-state-dir is required")
	}
	if cfg.peerLimit < 1 {
		return errors.New("-peer-limit must be at least 1")
	}
	if len(splitCSV(cfg.p2pListenAddrs)) == 0 {
		return errors.New("-p2p-listen must include at least one multiaddr")
	}
	if _, err := parseMultiaddrs(splitCSV(cfg.announceAddrs)); err != nil {
		return fmt.Errorf("invalid -announce-addrs: %w", err)
	}
	return nil
}

func startBootstrapRuntime(ctx context.Context, cfg appConfig) (*bootstrapRuntime, error) {
	if err := os.MkdirAll(cfg.stateDir, 0o700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	signKey, err := loadOrCreateIdentityKey(filepath.Join(cfg.stateDir, identityKeyFileName))
	if err != nil {
		return nil, err
	}

	announceAddrs, err := parseMultiaddrs(splitCSV(cfg.announceAddrs))
	if err != nil {
		return nil, fmt.Errorf("parse announce addrs: %w", err)
	}

	hostCfg := transportpkg.DefaultConfig()
	hostCfg.PrivateKey = signKey
	hostCfg.ListenAddrs = splitCSV(cfg.p2pListenAddrs)
	hostCfg.EnableTor = cfg.torEnabled
	hostCfg.EnableI2P = cfg.i2pEnabled
	hostCfg.I2PSAMAddr = cfg.i2pSAMAddr

	host, err := transportpkg.NewHost(ctx, hostCfg)
	if err != nil {
		return nil, fmt.Errorf("create bootstrap host: %w", err)
	}

	runtime := &bootstrapRuntime{
		host:          host,
		tracker:       newPeerTracker(cfg.peerMaxAge, cfg.peerLimit),
		signKey:       signKey,
		announceAddrs: announceAddrs,
		peerMaxAge:    cfg.peerMaxAge,
		httpListen:    cfg.listenAddr,
		stateDir:      cfg.stateDir,
	}

	runtime.observePeer(runtime.advertisedAddrInfo())
	host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			runtime.observeConn(conn)
		},
	})

	if host.DHT() != nil {
		if err := host.DHT().Bootstrap(ctx); err != nil {
			host.Close()
			return nil, fmt.Errorf("bootstrap DHT: %w", err)
		}
	}

	runtime.pex = discovery.NewPEX(host.Host, runtime.observePeer)
	if err := runtime.pex.Start(ctx); err != nil {
		host.Close()
		return nil, fmt.Errorf("start peer exchange: %w", err)
	}

	go runtime.refreshPeerStoreLoop(ctx)
	return runtime, nil
}

func loadOrCreateIdentityKey(path string) (ed25519.PrivateKey, error) {
	if raw, err := os.ReadFile(path); err == nil {
		if len(raw) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("invalid bootstrap identity key size: got %d want %d", len(raw), ed25519.PrivateKeySize)
		}
		return ed25519.PrivateKey(raw), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read bootstrap identity key: %w", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate bootstrap identity key: %w", err)
	}
	if err := os.WriteFile(path, privateKey, 0o600); err != nil {
		return nil, fmt.Errorf("write bootstrap identity key: %w", err)
	}
	return privateKey, nil
}

func newHandler(source signedPeerSource, allowOrigin string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, allowOrigin)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(source.HealthInfo())
	})

	mux.HandleFunc("/peers.json", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, allowOrigin)
		signedList, err := source.SignedPeerList()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to build peers list: %v", err), http.StatusInternalServerError)
			return
		}

		payload, err := json.Marshal(signedList)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to encode peers list: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, allowOrigin)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("murmur bootstrap server\nendpoints: /health /peers.json\n"))
	})

	return mux
}

func setCORS(w http.ResponseWriter, allowOrigin string) {
	if allowOrigin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
}

func buildListeners(ctx context.Context, cfg appConfig) ([]managedListener, error) {
	listeners := make([]managedListener, 0, 4)

	tcpListener, err := net.Listen("tcp", cfg.listenAddr)
	if err != nil {
		return nil, fmt.Errorf("create tcp listener: %w", err)
	}
	listeners = append(listeners, managedListener{name: "tcp", listener: tcpListener})

	if cfg.ngrokEnabled {
		tunnel, err := newNgrokListener(ctx, cfg.ngrokDomain)
		if err != nil {
			closeManagedListeners(listeners)
			return nil, err
		}
		listeners = append(listeners, managedListener{name: "ngrok", listener: tunnel})
	}

	if cfg.torEnabled {
		torListener, closer, err := newTorListener(cfg.torName, cfg.torPort)
		if err != nil {
			closeManagedListeners(listeners)
			return nil, err
		}
		listeners = append(listeners, managedListener{name: "tor", listener: torListener, closeFunc: closer})
	}

	if cfg.i2pEnabled {
		i2pListener, closer, err := newI2PListener(cfg.i2pName, cfg.i2pSAMAddr)
		if err != nil {
			closeManagedListeners(listeners)
			return nil, err
		}
		listeners = append(listeners, managedListener{name: "i2p", listener: i2pListener, closeFunc: closer})
	}

	return listeners, nil
}

func newNgrokListener(ctx context.Context, domain string) (net.Listener, error) {
	endpointOpts := []config.HTTPEndpointOption{}
	if domain != "" {
		endpointOpts = append(endpointOpts, config.WithDomain(domain))
	}

	tun, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(endpointOpts...),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("create ngrok listener: %w", err)
	}

	return tun, nil
}

func newTorListener(name, port string) (net.Listener, func() error, error) {
	onion, err := onramp.NewOnion(name)
	if err != nil {
		return nil, nil, fmt.Errorf("create tor onion instance: %w", err)
	}

	listener, err := onion.Listen(port)
	if err != nil {
		_ = onion.Close()
		return nil, nil, fmt.Errorf("create tor listener: %w", err)
	}

	closeFunc := func() error {
		if err := listener.Close(); err != nil {
			_ = onion.Close()
			return err
		}
		return onion.Close()
	}

	return listener, closeFunc, nil
}

func newI2PListener(name, samAddr string) (net.Listener, func() error, error) {
	garlic, err := onramp.NewGarlic(name, samAddr, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create i2p garlic instance: %w", err)
	}

	listener, err := garlic.Listen()
	if err != nil {
		_ = garlic.Close()
		return nil, nil, fmt.Errorf("create i2p listener: %w", err)
	}

	closeFunc := func() error {
		if err := listener.Close(); err != nil {
			_ = garlic.Close()
			return err
		}
		return garlic.Close()
	}

	return listener, closeFunc, nil
}

func serveUntilShutdown(ctx context.Context, listeners []managedListener, handler http.Handler) error {
	httpServer := &http.Server{Handler: handler}

	errCh := make(chan error, len(listeners))
	for _, listener := range listeners {
		l := listener
		go func() {
			if err := httpServer.Serve(l.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("%s serve error: %w", l.name, err)
			}
		}()
	}

	select {
	case <-ctx.Done():
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func closeManagedListeners(listeners []managedListener) {
	for _, l := range listeners {
		if l.listener != nil {
			_ = l.listener.Close()
		}
		if l.closeFunc != nil {
			_ = l.closeFunc()
		}
	}
}

func newPeerTracker(peerMaxAge time.Duration, peerLimit int) *peerTracker {
	return &peerTracker{
		peers:      make(map[peer.ID]trackedPeer),
		peerMaxAge: peerMaxAge,
		peerLimit:  peerLimit,
	}
}

func (p *peerTracker) Record(ai peer.AddrInfo) {
	if ai.ID == "" || len(ai.Addrs) == 0 {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	tracked := p.peers[ai.ID]
	if tracked.addrs == nil {
		tracked.addrs = make(map[string]multiaddr.Multiaddr, len(ai.Addrs))
	}
	for _, addr := range ai.Addrs {
		if addr == nil {
			continue
		}
		tracked.addrs[addr.String()] = addr
	}
	tracked.seen = time.Now()
	p.peers[ai.ID] = tracked
}

func (p *peerTracker) Snapshot(self peer.AddrInfo) []peer.AddrInfo {
	p.Record(self)

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	entries := make([]peer.AddrInfo, 0, len(p.peers))
	seenAt := make(map[peer.ID]time.Time, len(p.peers))

	for id, tracked := range p.peers {
		if p.peerMaxAge > 0 && now.Sub(tracked.seen) > p.peerMaxAge && id != self.ID {
			delete(p.peers, id)
			continue
		}

		addrs := make([]multiaddr.Multiaddr, 0, len(tracked.addrs))
		for _, addr := range tracked.addrs {
			addrs = append(addrs, addr)
		}
		if len(addrs) == 0 {
			continue
		}
		entries = append(entries, peer.AddrInfo{ID: id, Addrs: addrs})
		seenAt[id] = tracked.seen
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ID == self.ID {
			return true
		}
		if entries[j].ID == self.ID {
			return false
		}
		if seenAt[entries[i].ID].Equal(seenAt[entries[j].ID]) {
			return entries[i].ID.String() < entries[j].ID.String()
		}
		return seenAt[entries[i].ID].After(seenAt[entries[j].ID])
	})

	if p.peerLimit > 0 && len(entries) > p.peerLimit {
		entries = entries[:p.peerLimit]
	}
	return entries
}

func (p *peerTracker) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.peers)
}

func (r *bootstrapRuntime) Close() error {
	if r.pex != nil {
		_ = r.pex.Stop()
	}
	if r.host != nil {
		return r.host.Close()
	}
	return nil
}

func (r *bootstrapRuntime) refreshPeerStoreLoop(ctx context.Context) {
	ticker := time.NewTicker(peerScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.observePeerStore()
		}
	}
}

func (r *bootstrapRuntime) observePeerStore() {
	for _, peerID := range r.host.Peerstore().PeersWithAddrs() {
		if peerID == r.host.PeerID() {
			continue
		}
		addrs := r.host.Peerstore().Addrs(peerID)
		if len(addrs) == 0 {
			continue
		}
		r.observePeer(peer.AddrInfo{ID: peerID, Addrs: addrs})
	}
}

func (r *bootstrapRuntime) observeConn(conn network.Conn) {
	addrs := r.host.Peerstore().Addrs(conn.RemotePeer())
	if len(addrs) == 0 && conn.RemoteMultiaddr() != nil {
		addrs = []multiaddr.Multiaddr{conn.RemoteMultiaddr()}
	}
	r.observePeer(peer.AddrInfo{ID: conn.RemotePeer(), Addrs: addrs})
}

func (r *bootstrapRuntime) observePeer(ai peer.AddrInfo) {
	r.tracker.Record(ai)
}

func (r *bootstrapRuntime) advertisedAddrInfo() peer.AddrInfo {
	if len(r.announceAddrs) > 0 {
		return peer.AddrInfo{ID: r.host.PeerID(), Addrs: append([]multiaddr.Multiaddr(nil), r.announceAddrs...)}
	}
	return r.host.AddrInfo()
}

func (r *bootstrapRuntime) SignedPeerList() (*discovery.SignedPeerList, error) {
	peers := r.tracker.Snapshot(r.advertisedAddrInfo())
	return buildSignedPeerList(r.signKey, r.peerMaxAge, peers)
}

func (r *bootstrapRuntime) HealthInfo() healthResponse {
	return healthResponse{
		Status:       "ok",
		Version:      Version,
		Commit:       Commit,
		Timestamp:    time.Now().Unix(),
		PeerID:       r.host.PeerID().String(),
		KnownPeers:   r.tracker.Count(),
		HTTPListen:   r.httpListen,
		StateDir:     r.stateDir,
		P2PListeners: len(r.advertisedAddrInfo().Addrs),
	}
}

func buildSignedPeerList(signKey ed25519.PrivateKey, peerMaxAge time.Duration, peers []peer.AddrInfo) (*discovery.SignedPeerList, error) {
	signedList := discovery.FromPeerAddrInfos(peers)
	if peerMaxAge > 0 {
		signedList.PruneStale(peerMaxAge)
	}
	if len(signedList.Peers) == 0 {
		return nil, errors.New("no peers available to distribute")
	}
	if err := signedList.Sign(signKey); err != nil {
		return nil, fmt.Errorf("sign peers list: %w", err)
	}
	return signedList, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseMultiaddrs(values []string) ([]multiaddr.Multiaddr, error) {
	if len(values) == 0 {
		return nil, nil
	}
	addrs := make([]multiaddr.Multiaddr, 0, len(values))
	for _, value := range values {
		addr, err := multiaddr.NewMultiaddr(value)
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
