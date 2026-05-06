// Package accounting tracks tunnel traffic separately from social traffic.
package accounting

import (
	"sync"
	"sync/atomic"

	"github.com/opd-ai/murmur/pkg/tunneling"
)

// TunnelMetrics tracks traffic for a single tunnel.
type TunnelMetrics struct {
	BytesSent     atomic.Uint64
	BytesReceived atomic.Uint64
	RequestCount  atomic.Uint64
	ErrorCount    atomic.Uint64
	RebuildCount  atomic.Uint64
}

// Recorder tracks metrics for all active tunnels.
type Recorder struct {
	mu      sync.RWMutex
	tunnels map[tunneling.TunnelID]*TunnelMetrics
}

// NewRecorder creates a new traffic recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		tunnels: make(map[tunneling.TunnelID]*TunnelMetrics),
	}
}

// Register creates metrics tracking for a new tunnel.
func (r *Recorder) Register(id tunneling.TunnelID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tunnels[id]; !exists {
		r.tunnels[id] = &TunnelMetrics{}
	}
}

// Unregister removes metrics tracking for a closed tunnel.
func (r *Recorder) Unregister(id tunneling.TunnelID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tunnels, id)
}

// RecordBytesSent increments sent bytes for a tunnel.
func (r *Recorder) RecordBytesSent(id tunneling.TunnelID, n uint64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metrics, ok := r.tunnels[id]; ok {
		metrics.BytesSent.Add(n)
	}
}

// RecordBytesReceived increments received bytes for a tunnel.
func (r *Recorder) RecordBytesReceived(id tunneling.TunnelID, n uint64) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metrics, ok := r.tunnels[id]; ok {
		metrics.BytesReceived.Add(n)
	}
}

// RecordRequest increments request count for a tunnel.
func (r *Recorder) RecordRequest(id tunneling.TunnelID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metrics, ok := r.tunnels[id]; ok {
		metrics.RequestCount.Add(1)
	}
}

// RecordError increments error count for a tunnel.
func (r *Recorder) RecordError(id tunneling.TunnelID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metrics, ok := r.tunnels[id]; ok {
		metrics.ErrorCount.Add(1)
	}
}

// RecordRebuild increments circuit rebuild count for a tunnel.
func (r *Recorder) RecordRebuild(id tunneling.TunnelID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if metrics, ok := r.tunnels[id]; ok {
		metrics.RebuildCount.Add(1)
	}
}

// TotalBytesSent returns the sum of bytes sent across all tunnels.
func (r *Recorder) TotalBytesSent() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total uint64
	for _, m := range r.tunnels {
		total += m.BytesSent.Load()
	}
	return total
}

// TotalBytesReceived returns the sum of bytes received across all tunnels.
func (r *Recorder) TotalBytesReceived() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var total uint64
	for _, m := range r.tunnels {
		total += m.BytesReceived.Load()
	}
	return total
}

// ActiveTunnelCount returns the number of tunnels currently tracked.
func (r *Recorder) ActiveTunnelCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tunnels)
}

// QuotaExceeded checks if a tunnel has exceeded its daily bandwidth limit.
func (r *Recorder) QuotaExceeded(id tunneling.TunnelID, limitBytes uint64) bool {
	if limitBytes == 0 {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, ok := r.tunnels[id]
	if !ok {
		return false
	}

	totalBytes := metrics.BytesSent.Load() + metrics.BytesReceived.Load()
	return totalBytes > limitBytes
}
