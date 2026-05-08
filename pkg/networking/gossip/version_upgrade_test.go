// Package gossip – tests for the version upgrade / dual-subscription protocol.
//
//go:build test
// +build test

package gossip

import (
	"testing"
)

// TestTopicUpgradePairs verifies the canonical v1→v2 mapping is complete and consistent.
func TestTopicUpgradePairs(t *testing.T) {
	t.Parallel()

	// Every v1 topic must have a v2 pair.
	for _, v1 := range []string{TopicWaves, TopicIdentity, TopicShroud, TopicPulse} {
		v2, ok := topicUpgradePairs[v1]
		if !ok {
			t.Errorf("v1 topic %q has no v2 mapping", v1)
			continue
		}
		if v2 == v1 {
			t.Errorf("v2 topic for %q is identical to v1", v1)
		}
		// v2 string should end in /2, not /1.
		if len(v2) < 2 || v2[len(v2)-1] != '2' {
			t.Errorf("v2 topic %q does not end in /2", v2)
		}
	}
}

// TestV2TopicNameHelper checks the V2TopicName convenience function.
func TestV2TopicNameHelper(t *testing.T) {
	t.Parallel()

	if got := V2TopicName(TopicWaves); got != TopicWavesV2 {
		t.Errorf("V2TopicName(%q) = %q, want %q", TopicWaves, got, TopicWavesV2)
	}
	if got := V2TopicName("unknown"); got != "" {
		t.Errorf("V2TopicName(unknown) = %q, want empty string", got)
	}
}

// TestAllTopicPairs verifies AllTopicPairs returns a copy (not the internal map).
func TestAllTopicPairs(t *testing.T) {
	t.Parallel()

	pairs := AllTopicPairs()
	if len(pairs) != len(topicUpgradePairs) {
		t.Errorf("AllTopicPairs() length = %d, want %d", len(pairs), len(topicUpgradePairs))
	}
	// Mutating the returned copy must not affect the internal map.
	pairs[TopicWaves] = "tampered"
	if topicUpgradePairs[TopicWaves] == "tampered" {
		t.Error("AllTopicPairs returned a reference to the internal map instead of a copy")
	}
}

// TestDualTopicManagerActiveVersion verifies the publish version is stored correctly.
func TestDualTopicManagerActiveVersion(t *testing.T) {
	t.Parallel()

	// Construct minimal DualTopicManager without needing a real PubSub.
	d := &DualTopicManager{publishVersion: TopicVersionV1}
	if got := d.ActiveVersion(); got != TopicVersionV1 {
		t.Errorf("ActiveVersion() = %d, want %d", got, TopicVersionV1)
	}

	d.publishVersion = TopicVersionV2
	if got := d.ActiveVersion(); got != TopicVersionV2 {
		t.Errorf("ActiveVersion() = %d, want %d", got, TopicVersionV2)
	}
}

// TestTopicVersionV2Constants ensures v2 topic strings differ from v1 by exactly the suffix.
func TestTopicVersionV2Constants(t *testing.T) {
	t.Parallel()

	cases := []struct{ v1, v2 string }{
		{TopicWaves, TopicWavesV2},
		{TopicIdentity, TopicIdentityV2},
		{TopicShroud, TopicShroudV2},
		{TopicPulse, TopicPulseV2},
	}

	for _, tc := range cases {
		// The last character of v1 topic is '1'; v2 is '2'.
		if tc.v1[len(tc.v1)-1] != '1' {
			t.Errorf("v1 topic %q does not end in /1", tc.v1)
		}
		if tc.v2[len(tc.v2)-1] != '2' {
			t.Errorf("v2 topic %q does not end in /2", tc.v2)
		}
		// The prefix before the version digit should be identical.
		if tc.v1[:len(tc.v1)-1] != tc.v2[:len(tc.v2)-1] {
			t.Errorf("v1 %q and v2 %q have different prefixes", tc.v1, tc.v2)
		}
	}
}
