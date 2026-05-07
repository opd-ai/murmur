// Package main provides a standalone bootstrap/reseed host binary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-i2p/onramp"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"

	"github.com/opd-ai/murmur/pkg/networking/discovery"
)

// Version is the current version of MURMUR. Set by build flags.
var Version = "0.0.0-alpha"

// Commit is the git commit hash. Set by build flags.
var Commit = "unknown"

type appConfig struct {
	listenAddr string
	peersFile  string

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
	Status      string `json:"status"`
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	Timestamp   int64  `json:"timestamp"`
	PeersSource string `json:"peers_source"`
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

	handler, err := newHandler(cfg.peersFile, cfg.allowOrigin)
	if err != nil {
		return err
	}

	listeners, err := buildListeners(ctx, cfg)
	if err != nil {
		return err
	}
	defer closeManagedListeners(listeners)

	if len(listeners) == 0 {
		return errors.New("no listeners configured")
	}

	for _, l := range listeners {
		log.Printf("bootstrap listener active: %s (%s)", l.name, l.listener.Addr())
	}

	return serveUntilShutdown(ctx, listeners, handler)
}

func parseFlags() appConfig {
	var cfg appConfig

	flag.StringVar(&cfg.listenAddr, "listen", ":8081", "TCP listen address for bootstrap HTTP server")
	flag.StringVar(&cfg.peersFile, "peers-file", "", "Path to signed peers.json")
	flag.StringVar(&cfg.allowOrigin, "allow-origin", "*", "Value for Access-Control-Allow-Origin header")

	flag.BoolVar(&cfg.ngrokEnabled, "ngrok", false, "Enable ngrok listener")
	flag.StringVar(&cfg.ngrokDomain, "ngrok-domain", "", "ngrok custom domain (optional)")

	flag.BoolVar(&cfg.torEnabled, "tor", false, "Enable Tor hidden-service listener")
	flag.StringVar(&cfg.torName, "tor-name", "murmur-bootstrap", "Persistent Tor instance name")
	flag.StringVar(&cfg.torPort, "tor-port", "8081", "Tor hidden-service virtual port")

	flag.BoolVar(&cfg.i2pEnabled, "i2p", false, "Enable I2P listener")
	flag.StringVar(&cfg.i2pName, "i2p-name", "murmur-bootstrap", "Persistent I2P tunnel name")
	flag.StringVar(&cfg.i2pSAMAddr, "i2p-sam", "127.0.0.1:7656", "I2P SAMv3 bridge address")

	flag.Parse()
	return cfg
}

func validateConfig(cfg appConfig) error {
	if cfg.peersFile == "" {
		return errors.New("-peers-file is required")
	}

	if _, _, err := loadSignedPeers(cfg.peersFile); err != nil {
		return fmt.Errorf("invalid peers file: %w", err)
	}

	return nil
}

func newHandler(peersFile, allowOrigin string) (http.Handler, error) {
	provider := &peerFileProvider{path: peersFile}
	if _, err := provider.load(); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, allowOrigin)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:      "ok",
			Version:     Version,
			Commit:      Commit,
			Timestamp:   time.Now().Unix(),
			PeersSource: peersFile,
		})
	})

	mux.HandleFunc("/peers.json", func(w http.ResponseWriter, r *http.Request) {
		setCORS(w, allowOrigin)
		payload, err := provider.load()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to load peers: %v", err), http.StatusInternalServerError)
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

	return mux, nil
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

type peerFileProvider struct {
	path string

	mu      sync.RWMutex
	payload []byte
	modTime time.Time
	loaded  bool
}

func (p *peerFileProvider) load() ([]byte, error) {
	data, modTime, err := loadSignedPeers(p.path)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.loaded || modTime.After(p.modTime) {
		p.payload = data
		p.modTime = modTime
		p.loaded = true
	}

	out := make([]byte, len(p.payload))
	copy(out, p.payload)
	return out, nil
}

func loadSignedPeers(path string) ([]byte, time.Time, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat peers file: %w", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read peers file: %w", err)
	}

	var signed discovery.SignedPeerList
	if err := json.Unmarshal(raw, &signed); err != nil {
		return nil, time.Time{}, fmt.Errorf("parse peers json: %w", err)
	}

	if len(signed.Peers) == 0 {
		return nil, time.Time{}, errors.New("signed peers file has no peers")
	}

	return raw, stat.ModTime(), nil
}
