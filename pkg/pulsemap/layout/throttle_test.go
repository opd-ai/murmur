// Package layout - Tests for data update throttling.
package layout

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewUpdateThrottler(t *testing.T) {
	throttler := NewUpdateThrottler()

	if throttler == nil {
		t.Fatal("NewUpdateThrottler returned nil")
	}

	config := throttler.GetConfig()
	if config.NodeInterval != DefaultNodeInterval {
		t.Errorf("expected node interval %v, got %v", DefaultNodeInterval, config.NodeInterval)
	}
	if config.StateInterval != DefaultStateInterval {
		t.Errorf("expected state interval %v, got %v", DefaultStateInterval, config.StateInterval)
	}
}

func TestDefaultThrottleConfig(t *testing.T) {
	config := DefaultThrottleConfig()

	if config.NodeInterval != time.Second/30 {
		t.Error("NodeInterval should be ~33ms (30Hz)")
	}
	if config.StateInterval != time.Second/10 {
		t.Error("StateInterval should be 100ms (10Hz)")
	}
	if config.ConnectionInterval != time.Second/5 {
		t.Error("ConnectionInterval should be 200ms (5Hz)")
	}
	if config.ContentInterval != time.Second/2 {
		t.Error("ContentInterval should be 500ms (2Hz)")
	}
}

func TestUpdateThrottler_SetConfig(t *testing.T) {
	throttler := NewUpdateThrottler()

	newConfig := ThrottleConfig{
		NodeInterval:       50 * time.Millisecond,
		StateInterval:      200 * time.Millisecond,
		ConnectionInterval: 400 * time.Millisecond,
		ContentInterval:    1 * time.Second,
	}

	throttler.SetConfig(newConfig)

	config := throttler.GetConfig()
	if config.NodeInterval != 50*time.Millisecond {
		t.Error("config not updated correctly")
	}
}

func TestUpdateThrottler_SetInterval(t *testing.T) {
	throttler := NewUpdateThrottler()

	throttler.SetInterval(UpdateNodes, 100*time.Millisecond)
	if throttler.GetInterval(UpdateNodes) != 100*time.Millisecond {
		t.Error("SetInterval failed for UpdateNodes")
	}

	throttler.SetInterval(UpdateState, 200*time.Millisecond)
	if throttler.GetInterval(UpdateState) != 200*time.Millisecond {
		t.Error("SetInterval failed for UpdateState")
	}

	throttler.SetInterval(UpdateConnections, 300*time.Millisecond)
	if throttler.GetInterval(UpdateConnections) != 300*time.Millisecond {
		t.Error("SetInterval failed for UpdateConnections")
	}

	throttler.SetInterval(UpdateContent, 400*time.Millisecond)
	if throttler.GetInterval(UpdateContent) != 400*time.Millisecond {
		t.Error("SetInterval failed for UpdateContent")
	}
}

func TestUpdateThrottler_ShouldUpdate(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 50*time.Millisecond)

	// First update should always be allowed.
	if !throttler.ShouldUpdate(UpdateNodes) {
		t.Error("first update should be allowed")
	}

	// Immediate second update should be throttled.
	if throttler.ShouldUpdate(UpdateNodes) {
		t.Error("immediate second update should be throttled")
	}

	// Wait for interval to pass.
	time.Sleep(60 * time.Millisecond)

	// Now should be allowed.
	if !throttler.ShouldUpdate(UpdateNodes) {
		t.Error("update should be allowed after interval")
	}
}

func TestUpdateThrottler_ShouldUpdateNow(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 100*time.Millisecond)

	now := time.Now()

	// First update.
	if !throttler.ShouldUpdateNow(UpdateNodes, now) {
		t.Error("first update should be allowed")
	}

	// Same time - should be throttled.
	if throttler.ShouldUpdateNow(UpdateNodes, now) {
		t.Error("same time update should be throttled")
	}

	// 50ms later - still throttled.
	if throttler.ShouldUpdateNow(UpdateNodes, now.Add(50*time.Millisecond)) {
		t.Error("50ms later should still be throttled")
	}

	// 100ms later - allowed.
	if !throttler.ShouldUpdateNow(UpdateNodes, now.Add(100*time.Millisecond)) {
		t.Error("100ms later should be allowed")
	}
}

func TestUpdateThrottler_ForceUpdate(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 1*time.Second)

	// First update.
	throttler.ShouldUpdate(UpdateNodes)

	// Force update.
	throttler.ForceUpdate(UpdateNodes)

	stats := throttler.GetStats(UpdateNodes)
	if stats.UpdateCount != 2 {
		t.Errorf("expected 2 updates, got %d", stats.UpdateCount)
	}
}

func TestUpdateThrottler_TimeUntilUpdate(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 100*time.Millisecond)

	// Before any update, time should be 0.
	if throttler.TimeUntilUpdate(UpdateNodes) != 0 {
		t.Error("time until first update should be 0")
	}

	// After update, time should be positive.
	throttler.ShouldUpdate(UpdateNodes)
	timeUntil := throttler.TimeUntilUpdate(UpdateNodes)
	if timeUntil <= 0 {
		t.Error("time until next update should be positive after update")
	}
	if timeUntil > 100*time.Millisecond {
		t.Error("time until next should not exceed interval")
	}
}

func TestUpdateThrottler_IsPending(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 1*time.Second)

	// First update.
	throttler.ShouldUpdate(UpdateNodes)

	// Not pending initially after successful update.
	if throttler.IsPending(UpdateNodes) {
		t.Error("should not be pending after successful update")
	}

	// Try to update again - will be throttled.
	throttler.ShouldUpdate(UpdateNodes)

	// Now should be pending.
	if !throttler.IsPending(UpdateNodes) {
		t.Error("should be pending after throttled update")
	}
}

func TestUpdateThrottler_SetCallback(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 10*time.Millisecond)

	callCount := int32(0)
	throttler.SetCallback(UpdateNodes, func() {
		atomic.AddInt32(&callCount, 1)
	})

	// Trigger should call callback.
	if !throttler.TriggerIfReady(UpdateNodes) {
		t.Error("first trigger should succeed")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Error("callback should have been called")
	}

	// Immediate trigger should fail.
	if throttler.TriggerIfReady(UpdateNodes) {
		t.Error("immediate trigger should fail")
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Error("callback should not be called when throttled")
	}

	// Wait and trigger again.
	time.Sleep(15 * time.Millisecond)
	if !throttler.TriggerIfReady(UpdateNodes) {
		t.Error("trigger after wait should succeed")
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Error("callback should have been called again")
	}
}

func TestUpdateThrottler_GetStats(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 50*time.Millisecond)

	// Make some updates.
	throttler.ShouldUpdate(UpdateNodes) // Allowed.
	throttler.ShouldUpdate(UpdateNodes) // Dropped.
	throttler.ShouldUpdate(UpdateNodes) // Dropped.

	stats := throttler.GetStats(UpdateNodes)

	if stats.UpdateCount != 1 {
		t.Errorf("expected 1 update, got %d", stats.UpdateCount)
	}
	if stats.DropCount != 2 {
		t.Errorf("expected 2 drops, got %d", stats.DropCount)
	}
	if stats.TotalCount != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalCount)
	}
	if stats.DropRatio < 0.6 || stats.DropRatio > 0.7 {
		t.Errorf("expected drop ratio ~0.67, got %f", stats.DropRatio)
	}
}

func TestUpdateThrottler_GetAllStats(t *testing.T) {
	throttler := NewUpdateThrottler()

	stats := throttler.GetAllStats()

	if len(stats) != 4 {
		t.Errorf("expected 4 category stats, got %d", len(stats))
	}

	// Check categories are in order.
	expected := []UpdateCategory{UpdateNodes, UpdateState, UpdateConnections, UpdateContent}
	for i, s := range stats {
		if s.Category != expected[i] {
			t.Errorf("stats[%d] has wrong category", i)
		}
	}
}

func TestUpdateThrottler_Reset(t *testing.T) {
	throttler := NewUpdateThrottler()

	// Make some updates.
	throttler.ShouldUpdate(UpdateNodes)
	throttler.ShouldUpdate(UpdateNodes)

	// Reset.
	throttler.Reset()

	// Stats should be cleared.
	stats := throttler.GetStats(UpdateNodes)
	if stats.UpdateCount != 0 || stats.DropCount != 0 {
		t.Error("counters should be reset")
	}

	// Next update should be allowed immediately.
	if !throttler.ShouldUpdate(UpdateNodes) {
		t.Error("update should be allowed after reset")
	}
}

func TestUpdateThrottler_ResetCounters(t *testing.T) {
	throttler := NewUpdateThrottler()
	throttler.SetInterval(UpdateNodes, 1*time.Second)

	// Make some updates.
	throttler.ShouldUpdate(UpdateNodes)
	throttler.ShouldUpdate(UpdateNodes) // Dropped.

	// Reset counters only.
	throttler.ResetCounters()

	// Stats should be cleared.
	stats := throttler.GetStats(UpdateNodes)
	if stats.UpdateCount != 0 || stats.DropCount != 0 {
		t.Error("counters should be reset")
	}

	// Timing should be preserved - next update still throttled.
	if throttler.ShouldUpdate(UpdateNodes) {
		t.Error("timing should be preserved - update should be throttled")
	}
}

func TestCategoryName(t *testing.T) {
	tests := []struct {
		category UpdateCategory
		expected string
	}{
		{UpdateNodes, "Nodes"},
		{UpdateState, "State"},
		{UpdateConnections, "Connections"},
		{UpdateContent, "Content"},
		{UpdateCategory(99), "Unknown"},
	}

	for _, test := range tests {
		name := CategoryName(test.category)
		if name != test.expected {
			t.Errorf("CategoryName(%d) = %s, expected %s", test.category, name, test.expected)
		}
	}
}

func TestBatchThrottler_CheckAll(t *testing.T) {
	throttler := NewUpdateThrottler()
	batch := NewBatchThrottler(throttler)

	// First check - all should be ready.
	result := batch.CheckAll()

	if !result.NodesReady || !result.StateReady ||
		!result.ConnectionsReady || !result.ContentReady {
		t.Error("all categories should be ready on first check")
	}

	// Immediate second check - none should be ready.
	result = batch.CheckAll()

	if result.NodesReady || result.StateReady ||
		result.ConnectionsReady || result.ContentReady {
		t.Error("no categories should be ready on immediate second check")
	}
}

func TestBatchResult_ReadyCount(t *testing.T) {
	result := BatchResult{
		NodesReady:       true,
		StateReady:       false,
		ConnectionsReady: true,
		ContentReady:     false,
	}

	if result.ReadyCount() != 2 {
		t.Errorf("expected ReadyCount 2, got %d", result.ReadyCount())
	}
}

func TestBatchResult_AnyReady(t *testing.T) {
	result := BatchResult{}
	if result.AnyReady() {
		t.Error("empty result should have no ready categories")
	}

	result.ContentReady = true
	if !result.AnyReady() {
		t.Error("should return true when any category is ready")
	}
}

func TestThrottledUpdater(t *testing.T) {
	updater := NewThrottledUpdater()
	updater.Throttler().SetInterval(UpdateNodes, 10*time.Millisecond)

	counter := int32(0)

	// Queue an update.
	updater.QueueNodeUpdate(func() {
		atomic.AddInt32(&counter, 1)
	})

	if !updater.HasPending() {
		t.Error("should have pending update")
	}

	// Process - should run.
	result := updater.ProcessUpdates()
	if !result.NodesReady {
		t.Error("nodes update should have run")
	}
	if atomic.LoadInt32(&counter) != 1 {
		t.Error("update function should have been called")
	}

	if updater.HasPending() {
		t.Error("should not have pending update after processing")
	}
}

func TestThrottledUpdater_AllCategories(t *testing.T) {
	updater := NewThrottledUpdater()

	counters := make([]int32, 4)

	updater.QueueNodeUpdate(func() { atomic.AddInt32(&counters[0], 1) })
	updater.QueueStateUpdate(func() { atomic.AddInt32(&counters[1], 1) })
	updater.QueueConnectionUpdate(func() { atomic.AddInt32(&counters[2], 1) })
	updater.QueueContentUpdate(func() { atomic.AddInt32(&counters[3], 1) })

	result := updater.ProcessUpdates()

	if !result.NodesReady || !result.StateReady ||
		!result.ConnectionsReady || !result.ContentReady {
		t.Error("all categories should have processed")
	}

	for i, c := range counters {
		if atomic.LoadInt32(&c) != 1 {
			t.Errorf("counter %d should be 1", i)
		}
	}
}

func TestUpdateCategory_Constants(t *testing.T) {
	// Verify constants have expected values.
	if UpdateNodes != 0 {
		t.Error("UpdateNodes should be 0")
	}
	if UpdateState != 1 {
		t.Error("UpdateState should be 1")
	}
	if UpdateConnections != 2 {
		t.Error("UpdateConnections should be 2")
	}
	if UpdateContent != 3 {
		t.Error("UpdateContent should be 3")
	}
}

func TestDefaultIntervals(t *testing.T) {
	// 30Hz = ~33.33ms
	expected30Hz := time.Second / 30
	if DefaultNodeInterval != expected30Hz {
		t.Errorf("DefaultNodeInterval should be %v", expected30Hz)
	}

	// 10Hz = 100ms
	if DefaultStateInterval != 100*time.Millisecond {
		t.Error("DefaultStateInterval should be 100ms")
	}

	// 5Hz = 200ms
	if DefaultConnectionInterval != 200*time.Millisecond {
		t.Error("DefaultConnectionInterval should be 200ms")
	}

	// 2Hz = 500ms
	if DefaultContentInterval != 500*time.Millisecond {
		t.Error("DefaultContentInterval should be 500ms")
	}
}
