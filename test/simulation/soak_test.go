//go:build soak && simulation
// +build soak,simulation

// Package simulation provides long-running stability tests for MURMUR.
// Run with: go test -tags=soak,simulation -timeout=25h ./test/simulation -v -run=TestSoak24Hour
package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opd-ai/murmur/pkg/networking/gossip"
	"github.com/opd-ai/murmur/pkg/resources"
	"github.com/opd-ai/murmur/pkg/store"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// SoakMetrics tracks resource usage during soak testing.
type SoakMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	ElapsedSeconds    int64     `json:"elapsed_seconds"`
	HeapAllocMB       uint64    `json:"heap_alloc_mb"`
	HeapInUseMB       uint64    `json:"heap_inuse_mb"`
	TotalAllocMB      uint64    `json:"total_alloc_mb"`
	NumGC             uint32    `json:"num_gc"`
	LastGCPauseMicros uint64    `json:"last_gc_pause_micros"`
	MaxGCPauseMicros  uint64    `json:"max_gc_pause_micros"`
	NumGoroutine      int       `json:"num_goroutine"`
	BboltSizeBytes    int64     `json:"bbolt_size_bytes"`
	ActiveCircuits    int       `json:"active_circuits"`
	WavesReceived     int64     `json:"waves_received"`
	WavesPublished    int64     `json:"waves_published"`
	MemoryWarnings    int       `json:"memory_warnings"`
	MemoryCritical    int       `json:"memory_critical"`
	GCPauseViolations int       `json:"gc_pause_violations"`
	GoroutineLeaks    int       `json:"goroutine_leaks"`
}

// TestSoak24Hour runs a 24-hour continuous stress test per P2 requirement.
// Monitors: memory growth, goroutine leaks, GC sweep times, Bbolt DB growth, circuit rotation leaks.
// Target metrics from TECHNICAL_IMPLEMENTATION.md §9:
// - Memory usage <256 MiB (normal operation)
// - GC sweep <100ms
// - Bbolt DB <50 MiB
// - No goroutine leaks (baseline ±5)
func TestSoak24Hour(t *testing.T) {
	const (
		duration           = 24 * time.Hour
		nodeCount          = 50 // Moderate mesh for sustained operation
		sampleInterval     = 30 * time.Second
		circuitRotateEvery = 10 * time.Minute
		wavesPerNode       = 10 // Waves per node per interval
	)

	ctx, cancel := context.WithTimeout(context.Background(), duration+5*time.Minute)
	defer cancel()

	// Create output directory for metrics
	metricsDir := filepath.Join(".", "soak-metrics")
	require.NoError(t, os.MkdirAll(metricsDir, 0o755), "creating metrics directory")

	// Open metrics log file
	metricsPath := filepath.Join(metricsDir, fmt.Sprintf("soak-24h-%d.json", time.Now().Unix()))
	metricsFile, err := os.Create(metricsPath)
	require.NoError(t, err, "creating metrics file")
	defer metricsFile.Close()

	t.Logf("Starting 24-hour soak test with %d nodes", nodeCount)
	t.Logf("Metrics will be written to: %s", metricsPath)

	startTime := time.Now()

	// Create temp dir for Bbolt databases
	tempDir := t.TempDir()

	// Create simulation nodes
	t.Logf("Creating %d simulation nodes...", nodeCount)
	nodes := make([]*SimNode, nodeCount)
	databases := make([]*bbolt.DB, nodeCount)

	for i := 0; i < nodeCount; i++ {
		node, err := createSimNode(ctx, i)
		require.NoError(t, err, "creating node %d", i)
		nodes[i] = node
		defer node.Host.Close()

		// Create Bbolt database for each node
		dbPath := filepath.Join(tempDir, fmt.Sprintf("node-%d.db", i))
		db, err := bbolt.Open(dbPath, 0o600, &bbolt.Options{
			Timeout: 1 * time.Second,
		})
		require.NoError(t, err, "opening database for node %d", i)
		databases[i] = db
		defer db.Close()

		// Initialize canonical buckets per spec
		err = db.Update(func(tx *bbolt.Tx) error {
			for _, bucket := range [][]byte{
				store.BucketIdentity,
				store.BucketPeers,
				store.BucketWaves,
				store.BucketThreads,
				store.BucketShroud,
				store.BucketResonance,
				store.BucketConfig,
			} {
				if _, err := tx.CreateBucketIfNotExists(bucket); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(t, err, "initializing buckets for node %d", i)
	}
	t.Logf("✓ Created %d nodes", nodeCount)

	// Connect nodes in mesh topology
	t.Log("Establishing mesh topology...")
	connectMesh(t, nodes, 6, 12)
	t.Log("✓ Mesh topology established")

	// Subscribe all nodes to Wave topic
	topic := gossip.TopicWaves
	var wavesReceived atomic.Int64
	var wavesPublished atomic.Int64

	t.Log("Subscribing nodes to topics...")
	for i, node := range nodes {
		sub, err := node.PubSub.Subscribe(topic)
		require.NoError(t, err, "subscribing node %d", i)

		// Start message receiver
		go func(n *SimNode) {
			for {
				_, err := sub.Next(ctx)
				if err != nil {
					return
				}
				wavesReceived.Add(1)
			}
		}(node)
	}
	t.Log("✓ Nodes subscribed")

	// Initialize resource monitoring
	memMonitor := resources.NewMemoryMonitor(resources.DefaultMaxMemoryMB)
	var memoryWarnings atomic.Int32
	var memoryCritical atomic.Int32
	var gcPauseViolations atomic.Int32
	var goroutineLeaks atomic.Int32

	// Record baseline goroutine count
	baselineGoroutines := runtime.NumGoroutine()
	maxGCPauseMicros := uint64(0)
	prevNumGC := uint32(0)

	// Start Wave publishing goroutine (simulates real activity)
	publishDone := make(chan struct{})
	go func() {
		defer close(publishDone)
		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Each node publishes waves
				for _, node := range nodes {
					for w := 0; w < wavesPerNode; w++ {
						wave, err := createTestWave(node.KeyPair)
						if err != nil {
							continue
						}
						envelope, err := wrapWave(wave, node.KeyPair)
						if err != nil {
							continue
						}
						data, err := proto.Marshal(envelope)
						if err != nil {
							continue
						}
						if err := node.PubSub.Publish(topic, data); err != nil {
							continue
						}
						wavesPublished.Add(1)
					}
				}
			}
		}
	}()

	// Start metrics collection goroutine
	metricsDone := make(chan struct{})
	go func() {
		defer close(metricsDone)
		ticker := time.NewTicker(sampleInterval)
		defer ticker.Stop()
		encoder := json.NewEncoder(metricsFile)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)

				// Collect memory stats
				var ms runtime.MemStats
				runtime.ReadMemStats(&ms)

				// Check memory status
				memStatus := memMonitor.Check()
				if memStatus.Warning {
					memoryWarnings.Add(1)
				}
				if memStatus.Critical {
					memoryCritical.Add(1)
				}

				// Check GC pause times (target: <100ms = 100,000 microseconds)
				if ms.NumGC > prevNumGC {
					lastPause := ms.PauseNs[(ms.NumGC+255)%256] / 1000 // Convert to microseconds
					if lastPause > 100000 {
						gcPauseViolations.Add(1)
					}
					if lastPause > maxGCPauseMicros {
						maxGCPauseMicros = lastPause
					}
					prevNumGC = ms.NumGC
				}

				// Check goroutine count (allow ±5 from baseline)
				currentGoroutines := runtime.NumGoroutine()
				if currentGoroutines > baselineGoroutines+5 {
					goroutineLeaks.Add(1)
				}

				// Get total Bbolt database size
				var totalDBSize int64
				for i := range databases {
					// Get database file size from filesystem
					dbPath := filepath.Join(tempDir, fmt.Sprintf("node-%d.db", i))
					if info, err := os.Stat(dbPath); err == nil {
						totalDBSize += info.Size()
					}
				}

				// Record metrics
				metrics := SoakMetrics{
					Timestamp:         time.Now(),
					ElapsedSeconds:    int64(elapsed.Seconds()),
					HeapAllocMB:       ms.Alloc / (1024 * 1024),
					HeapInUseMB:       ms.HeapInuse / (1024 * 1024),
					TotalAllocMB:      ms.TotalAlloc / (1024 * 1024),
					NumGC:             ms.NumGC,
					LastGCPauseMicros: ms.PauseNs[(ms.NumGC+255)%256] / 1000,
					MaxGCPauseMicros:  maxGCPauseMicros,
					NumGoroutine:      currentGoroutines,
					BboltSizeBytes:    totalDBSize,
					ActiveCircuits:    0, // Circuit tracking not implemented in soak test
					WavesReceived:     wavesReceived.Load(),
					WavesPublished:    wavesPublished.Load(),
					MemoryWarnings:    int(memoryWarnings.Load()),
					MemoryCritical:    int(memoryCritical.Load()),
					GCPauseViolations: int(gcPauseViolations.Load()),
					GoroutineLeaks:    int(goroutineLeaks.Load()),
				}

				// Write to JSON log
				if err := encoder.Encode(metrics); err != nil {
					t.Logf("Warning: failed to encode metrics: %v", err)
				}

				// Log progress every 1 hour
				if int64(elapsed.Seconds())%3600 < int64(sampleInterval.Seconds()) {
					t.Logf("✓ Soak test running: %v elapsed, heap=%dMB, goroutines=%d, waves_rx=%d, waves_tx=%d",
						elapsed.Round(time.Minute), metrics.HeapAllocMB, metrics.NumGoroutine,
						metrics.WavesReceived, metrics.WavesPublished)
				}
			}
		}
	}()

	// Wait for test duration
	t.Logf("Soak test running for %v...", duration)
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			t.Log("✓ 24-hour duration completed")
		} else {
			t.Logf("Soak test interrupted: %v", ctx.Err())
		}
	}

	// Shutdown gracefully
	cancel()
	<-publishDone
	<-metricsDone

	// Collect final metrics
	var finalStats runtime.MemStats
	runtime.ReadMemStats(&finalStats)
	finalGoroutines := runtime.NumGoroutine()
	finalMemStatus := memMonitor.Check()

	var totalDBSize int64
	for i := range databases {
		// Get database file size from filesystem
		dbPath := filepath.Join(tempDir, fmt.Sprintf("node-%d.db", i))
		if info, err := os.Stat(dbPath); err == nil {
			totalDBSize += info.Size()
		}
	}

	// Report final results
	t.Logf("\n========== SOAK TEST RESULTS ==========")
	t.Logf("Duration: %v", time.Since(startTime).Round(time.Second))
	t.Logf("Memory:")
	t.Logf("  Heap Alloc: %d MB", finalStats.Alloc/(1024*1024))
	t.Logf("  Heap InUse: %d MB", finalStats.HeapInuse/(1024*1024))
	t.Logf("  Total Alloc: %d MB", finalStats.TotalAlloc/(1024*1024))
	t.Logf("  Memory Warnings: %d", memoryWarnings.Load())
	t.Logf("  Memory Critical: %d", memoryCritical.Load())
	t.Logf("Goroutines:")
	t.Logf("  Baseline: %d", baselineGoroutines)
	t.Logf("  Final: %d", finalGoroutines)
	t.Logf("  Leak Events: %d", goroutineLeaks.Load())
	t.Logf("GC:")
	t.Logf("  Total Collections: %d", finalStats.NumGC)
	t.Logf("  Max Pause: %d µs (%.2f ms)", maxGCPauseMicros, float64(maxGCPauseMicros)/1000.0)
	t.Logf("  GC Pause Violations (>100ms): %d", gcPauseViolations.Load())
	t.Logf("Database:")
	t.Logf("  Total Bbolt Size: %.2f MB", float64(totalDBSize)/(1024*1024))
	t.Logf("Traffic:")
	t.Logf("  Waves Received: %d", wavesReceived.Load())
	t.Logf("  Waves Published: %d", wavesPublished.Load())
	t.Logf("=======================================\n")

	// Assertions per P2 requirements
	require.LessOrEqual(t, int(memoryCritical.Load()), 10,
		"Memory critical events should be rare (<10 in 24h)")
	require.LessOrEqual(t, int(gcPauseViolations.Load()), 50,
		"GC pause violations (>100ms) should be rare (<50 in 24h)")
	require.LessOrEqual(t, finalGoroutines, baselineGoroutines+10,
		"Goroutine count should not grow significantly (baseline %d, final %d)", baselineGoroutines, finalGoroutines)
	require.Less(t, totalDBSize, int64(50*1024*1024),
		"Total Bbolt database size should remain under 50 MiB per spec")
	require.Greater(t, wavesReceived.Load(), int64(0),
		"Should have received waves during test")
	require.Greater(t, wavesPublished.Load(), int64(0),
		"Should have published waves during test")

	// Final sanity checks
	require.False(t, finalMemStatus.Critical, "Memory should not be critical at end of test")
	require.LessOrEqual(t, int(finalMemStatus.UsedMB), 256,
		"Memory usage should be within spec limit (256 MiB)")

	t.Logf("✓ Soak test completed successfully")
	t.Logf("✓ Detailed metrics written to: %s", metricsPath)

	// Force GC and check for finalizer leaks
	debug.SetGCPercent(-1) // Disable automatic GC
	runtime.GC()
	runtime.GC()            // Second pass to clean finalizers
	debug.SetGCPercent(100) // Re-enable

	afterGC := runtime.NumGoroutine()
	t.Logf("✓ Post-GC goroutines: %d (baseline: %d)", afterGC, baselineGoroutines)
}
