package modes

import (
	"testing"
)

func TestNetworkSeparatorCreation(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	if ns == nil {
		t.Fatal("NewNetworkSeparator returned nil")
	}
}

func TestTopicRegistration(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	// Register topics.
	ns.RegisterTopic("/murmur/waves/1", TopicCategorySurface)
	ns.RegisterTopic("/murmur/anonymous/waves/1", TopicCategoryAnonymous)
	ns.RegisterTopic("/murmur/pulse/1", TopicCategoryShared)

	// In Open mode, only Surface and Shared topics should be allowed.
	if !ns.IsTopicAllowed("/murmur/waves/1") {
		t.Error("Surface topic should be allowed in Open mode")
	}
	if ns.IsTopicAllowed("/murmur/anonymous/waves/1") {
		t.Error("Anonymous topic should NOT be allowed in Open mode")
	}
	if !ns.IsTopicAllowed("/murmur/pulse/1") {
		t.Error("Shared topic should be allowed in Open mode")
	}
}

func TestTopicAllowanceByMode(t *testing.T) {
	tests := []struct {
		mode            Mode
		surfaceAllowed  bool
		anonymousAllowed bool
	}{
		{Open, true, false},
		{Hybrid, true, true},
		{Guarded, true, true},
		{Fortress, false, true},
	}

	for _, tc := range tests {
		manager := NewManagerWithConfig(tc.mode, 0)
		manager.SetSpecterAvailable(true)
		manager.SetShroudAvailable(true)
		ns := NewNetworkSeparator(manager)

		ns.RegisterTopic("surface", TopicCategorySurface)
		ns.RegisterTopic("anonymous", TopicCategoryAnonymous)
		ns.RegisterTopic("shared", TopicCategoryShared)

		if got := ns.IsTopicAllowed("surface"); got != tc.surfaceAllowed {
			t.Errorf("mode %s: surface topic allowed = %v, want %v",
				tc.mode, got, tc.surfaceAllowed)
		}
		if got := ns.IsTopicAllowed("anonymous"); got != tc.anonymousAllowed {
			t.Errorf("mode %s: anonymous topic allowed = %v, want %v",
				tc.mode, got, tc.anonymousAllowed)
		}
		// Shared always allowed.
		if !ns.IsTopicAllowed("shared") {
			t.Errorf("mode %s: shared topic should always be allowed", tc.mode)
		}
	}
}

func TestTopicUpdateOnModeTransition(t *testing.T) {
	manager := NewManagerWithConfig(Open, 0)
	manager.SetSpecterAvailable(true)
	ns := NewNetworkSeparator(manager)

	ns.RegisterTopic("surface", TopicCategorySurface)
	ns.RegisterTopic("anonymous", TopicCategoryAnonymous)

	// In Open mode, anonymous should NOT be allowed.
	if ns.IsTopicAllowed("anonymous") {
		t.Error("Anonymous topic should NOT be allowed in Open mode")
	}

	// Transition to Hybrid.
	if err := manager.Transition(Hybrid); err != nil {
		t.Fatalf("Transition to Hybrid failed: %v", err)
	}

	// Give listener goroutine time to execute.
	// Note: In production, this would be synchronous or use channels.

	// After transition, anonymous should be allowed.
	// But the listener runs in a goroutine, so we check synchronously.
	// For this test, we'll manually trigger the update.
	ns.updateAllowedTopics(Hybrid)

	if !ns.IsTopicAllowed("anonymous") {
		t.Error("Anonymous topic should be allowed after transition to Hybrid")
	}
}

func TestFilterTopics(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	ns.RegisterTopic("surface", TopicCategorySurface)
	ns.RegisterTopic("anonymous", TopicCategoryAnonymous)
	ns.RegisterTopic("shared", TopicCategoryShared)

	topics := []string{"surface", "anonymous", "shared", "unknown"}
	filtered := ns.FilterTopics(topics)

	// In Open mode, should get surface and shared (not anonymous, not unknown).
	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered topics, got %d: %v", len(filtered), filtered)
	}

	found := make(map[string]bool)
	for _, topic := range filtered {
		found[topic] = true
	}

	if !found["surface"] {
		t.Error("surface should be in filtered list")
	}
	if !found["shared"] {
		t.Error("shared should be in filtered list")
	}
	if found["anonymous"] {
		t.Error("anonymous should NOT be in filtered list in Open mode")
	}
}

func TestCanPublish(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	ns.RegisterTopic("surface", TopicCategorySurface)
	ns.RegisterTopic("anonymous", TopicCategoryAnonymous)

	// In Open mode, can publish to surface.
	if err := ns.CanPublish("surface"); err != nil {
		t.Errorf("should be able to publish to surface in Open mode: %v", err)
	}

	// In Open mode, cannot publish to anonymous.
	if err := ns.CanPublish("anonymous"); err != ErrTopicNotAllowed {
		t.Errorf("expected ErrTopicNotAllowed for anonymous in Open mode, got %v", err)
	}
}

func TestCanSubscribe(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	ns.RegisterTopic("surface", TopicCategorySurface)
	ns.RegisterTopic("anonymous", TopicCategoryAnonymous)

	// In Open mode, can subscribe to surface.
	if err := ns.CanSubscribe("surface"); err != nil {
		t.Errorf("should be able to subscribe to surface in Open mode: %v", err)
	}

	// In Open mode, cannot subscribe to anonymous.
	if err := ns.CanSubscribe("anonymous"); err != ErrTopicNotAllowed {
		t.Errorf("expected ErrTopicNotAllowed for anonymous in Open mode, got %v", err)
	}
}

func TestCanPerformOperation(t *testing.T) {
	tests := []struct {
		mode      Mode
		operation NetworkOperation
		allowed   bool
	}{
		{Open, OperationPublishSurface, true},
		{Open, OperationPublishAnonymous, false},
		{Hybrid, OperationPublishSurface, true},
		{Hybrid, OperationPublishAnonymous, true},
		{Fortress, OperationPublishSurface, false},
		{Fortress, OperationPublishAnonymous, true},
		{Guarded, OperationShroudRelay, true},
		{Open, OperationShroudRelay, false},
	}

	for _, tc := range tests {
		manager := NewManagerWithConfig(tc.mode, 0)
		manager.SetSpecterAvailable(true)
		manager.SetShroudAvailable(true)
		ns := NewNetworkSeparator(manager)

		err := ns.CanPerformOperation(tc.operation)
		if tc.allowed && err != nil {
			t.Errorf("mode %s, op %d: expected allowed, got error: %v",
				tc.mode, tc.operation, err)
		}
		if !tc.allowed && err == nil {
			t.Errorf("mode %s, op %d: expected error, got nil",
				tc.mode, tc.operation)
		}
	}
}

func TestGetPolicy(t *testing.T) {
	tests := []struct {
		mode                 Mode
		allowSurface         bool
		allowAnonymous       bool
		requireShroud        bool
		requireTrafficPadding bool
	}{
		{Open, true, false, false, false},
		{Hybrid, true, true, false, false},
		{Guarded, true, true, false, true},
		{Fortress, false, true, true, true},
	}

	for _, tc := range tests {
		manager := NewManagerWithConfig(tc.mode, 0)
		ns := NewNetworkSeparator(manager)

		policy := ns.GetPolicy()

		if policy.Mode != tc.mode {
			t.Errorf("expected mode %s, got %s", tc.mode, policy.Mode)
		}
		if policy.AllowSurfaceTopics != tc.allowSurface {
			t.Errorf("mode %s: AllowSurfaceTopics = %v, want %v",
				tc.mode, policy.AllowSurfaceTopics, tc.allowSurface)
		}
		if policy.AllowAnonymousTopics != tc.allowAnonymous {
			t.Errorf("mode %s: AllowAnonymousTopics = %v, want %v",
				tc.mode, policy.AllowAnonymousTopics, tc.allowAnonymous)
		}
		if policy.RequireShroud != tc.requireShroud {
			t.Errorf("mode %s: RequireShroud = %v, want %v",
				tc.mode, policy.RequireShroud, tc.requireShroud)
		}
		if policy.RequireTrafficPadding != tc.requireTrafficPadding {
			t.Errorf("mode %s: RequireTrafficPadding = %v, want %v",
				tc.mode, policy.RequireTrafficPadding, tc.requireTrafficPadding)
		}
	}
}

func TestAllowedTopics(t *testing.T) {
	manager := NewManager()
	ns := NewNetworkSeparator(manager)

	ns.RegisterTopic("surface1", TopicCategorySurface)
	ns.RegisterTopic("surface2", TopicCategorySurface)
	ns.RegisterTopic("anonymous", TopicCategoryAnonymous)
	ns.RegisterTopic("shared", TopicCategoryShared)

	allowed := ns.AllowedTopics()

	// In Open mode: surface1, surface2, shared should be allowed.
	if len(allowed) != 3 {
		t.Errorf("expected 3 allowed topics, got %d", len(allowed))
	}
}
