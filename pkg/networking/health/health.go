// Package health provides HTTP health check and metrics endpoint for bootstrap nodes.
// Per AUDIT.md MEDIUM finding, bootstrap node operators need to monitor
// node reachability, connection count, and Prometheus metrics.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server provides HTTP health check and Prometheus metrics endpoints for monitoring.
// Per AUDIT.md: exposes /health endpoint (JSON status) and /metrics endpoint
// (Prometheus format with connection counts, message rates, etc.).
type Server struct {
	mu         sync.RWMutex
	host       host.Host
	pubsub     *gossip.PubSub
	httpServer *http.Server
	startTime  time.Time
}

// HealthResponse is the JSON response for the /health endpoint.
type HealthResponse struct {
	Status        string   `json:"status"`         // "ok" or "degraded"
	PeerID        string   `json:"peer_id"`        // libp2p peer ID
	Connections   int      `json:"connections"`    // number of connected peers
	Topics        []string `json:"topics"`         // list of subscribed GossipSub topics
	UptimeSeconds int64    `json:"uptime_seconds"` // seconds since server start
	Timestamp     int64    `json:"timestamp"`      // current Unix timestamp
}

// NewServer creates a new health check HTTP server.
// It does not start listening until Start() is called.
func NewServer(host host.Host, pubsub *gossip.PubSub) *Server {
	return &Server{
		host:      host,
		pubsub:    pubsub,
		startTime: time.Now(),
	}
}

// Start begins listening on the specified port.
// Returns an error if the server fails to start.
// The server runs in a background goroutine and can be stopped via ctx.
func (s *Server) Start(ctx context.Context, port int) error {
	if err := s.initializeServer(port); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	s.startServerGoroutine(errCh)
	s.startShutdownMonitor(ctx, errCh)

	return nil
}

// initializeServer checks for duplicate start and creates the HTTP server.
func (s *Server) initializeServer(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer != nil {
		return fmt.Errorf("health server already started")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return nil
}

// startServerGoroutine launches the HTTP server in a background goroutine.
func (s *Server) startServerGoroutine(errCh chan error) {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("health server error: %w", err)
		}
	}()
}

// startShutdownMonitor monitors context cancellation and server errors, initiating shutdown when needed.
func (s *Server) startShutdownMonitor(ctx context.Context, errCh chan error) {
	go func() {
		select {
		case <-ctx.Done():
			s.shutdownGracefully()
		case err := <-errCh:
			if err != nil {
				fmt.Printf("Health server error: %v\n", err)
			}
		}
	}()
}

// shutdownGracefully performs a graceful HTTP server shutdown with timeout.
func (s *Server) shutdownGracefully() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Health server shutdown error: %v\n", err)
	}
}

// handleHealth serves the /health endpoint.
// Returns JSON with node status, peer ID, connection count, topics, and uptime.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get connection count
	peers := s.host.Network().Peers()
	connectionCount := len(peers)

	// Get subscribed topics
	var topics []string
	if s.pubsub != nil {
		topics = s.pubsub.Topics()
	}

	// Calculate uptime
	uptime := int64(time.Since(s.startTime).Seconds())

	// Determine status (ok if >0 connections, degraded otherwise)
	status := "ok"
	if connectionCount == 0 && uptime > 60 {
		// If no connections after 60s, mark as degraded
		status = "degraded"
	}

	response := HealthResponse{
		Status:        status,
		PeerID:        s.host.ID().String(),
		Connections:   connectionCount,
		Topics:        topics,
		UptimeSeconds: uptime,
		Timestamp:     time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil {
		return nil
	}

	return s.httpServer.Shutdown(ctx)
}
