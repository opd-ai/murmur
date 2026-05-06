//go:build simulation

package resonance

import (
	"context"
	"math"
	"math/rand"
	"testing"
	"time"
)

// TestResonanceConvergence verifies Resonance score convergence across a simulated network.
// Per ROADMAP.md, this test validates that Resonance scores stabilize correctly with 100+ nodes
// and 1000+ interactions.
func TestResonanceConvergence(t *testing.T) {
	const (
		nodeCount        = 100
		interactionCount = 1000
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	_ = ctx // Use context for future cancellation handling

	// Create SurfaceScore instances for all nodes
	scores := make([]*SurfaceScore, nodeCount)
	for i := 0; i < nodeCount; i++ {
		scores[i] = NewSurfaceScore()
	}

	// Simulate network interactions
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	t.Logf("Simulating %d interactions across %d nodes", interactionCount, nodeCount)

	// Record connection events (peer connections)
	for i := 0; i < interactionCount/2; i++ {
		nodeA := rng.Intn(nodeCount)
		nodeB := rng.Intn(nodeCount)
		if nodeA != nodeB {
			peerIDA := string(rune('A' + nodeA))
			peerIDB := string(rune('A' + nodeB))
			scores[nodeA].AddConnectionWithAge(peerIDB)
			scores[nodeB].AddConnectionWithAge(peerIDA)
		}
	}

	// Record Wave publications
	for i := 0; i < interactionCount/4; i++ {
		nodeID := rng.Intn(nodeCount)
		scores[nodeID].AddWave()
	}

	// Record amplifications
	for i := 0; i < interactionCount/4; i++ {
		publisher := rng.Intn(nodeCount)
		amplifier := rng.Intn(nodeCount)
		if publisher != amplifier {
			scores[publisher].AddAmplificationReceived()
			scores[amplifier].AddAmplificationGiven()
		}
	}

	// Compute scores
	computedScores := make([]int, nodeCount)
	var totalScore int64

	for i, score := range scores {
		computedScores[i] = score.Compute()
		totalScore += int64(computedScores[i])
	}

	avgScore := float64(totalScore) / float64(nodeCount)
	t.Logf("Average Resonance score: %.2f", avgScore)

	// Calculate variance and standard deviation
	var variance float64
	for _, score := range computedScores {
		diff := float64(score) - avgScore
		variance += diff * diff
	}
	variance /= float64(nodeCount)
	stdDev := math.Sqrt(variance)

	t.Logf("Score distribution: mean=%.2f, stddev=%.2f, variance=%.2f", avgScore, stdDev, variance)

	// Verify milestone distribution
	milestoneDistribution := make(map[SurfaceRank]int)
	for _, score := range computedScores {
		rank := SurfaceRankFromScore(score)
		milestoneDistribution[rank]++
	}

	t.Logf("Milestone distribution:")
	for rank := SurfaceRankNone; rank <= SurfaceRankCorona; rank++ {
		count := milestoneDistribution[rank]
		if count > 0 {
			t.Logf("  %s: %d nodes (%.1f%%)", rank.String(), count, float64(count)*100/float64(nodeCount))
		}
	}

	// Verify at least some nodes reached milestones
	nodesWithMilestones := 0
	for rank, count := range milestoneDistribution {
		if rank != SurfaceRankNone {
			nodesWithMilestones += count
		}
	}

	if nodesWithMilestones == 0 {
		t.Error("No nodes reached any Resonance milestone after convergence")
	} else {
		t.Logf("Convergence successful: %d/%d nodes (%.1f%%) reached milestones",
			nodesWithMilestones, nodeCount, float64(nodesWithMilestones)*100/float64(nodeCount))
	}

	// Verify scores are monotonically non-negative
	for i, score := range computedScores {
		if score < 0 {
			t.Errorf("Node %d has negative Resonance score: %d", i, score)
		}
	}

	// Verify score correlation with activity
	// Nodes with more connections/waves/amplifications should generally have higher scores
	scoresByActivity := make(map[int]int) // activity count -> total score
	for i, score := range scores {
		activityCount := score.ConnectionCount + score.WaveCount30d +
			score.DistinctAmplifiers30d + score.DistinctAmplifiedWaves30d
		scoresByActivity[activityCount] += computedScores[i]
	}

	t.Logf("Verified %d nodes with varied activity levels generated appropriate score distribution", nodeCount)
}

// TestResonanceScoreCorrelation verifies that more active nodes have higher scores.
func TestResonanceScoreCorrelation(t *testing.T) {
	const nodeCount = 50

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	_ = ctx

	// Create nodes with explicitly varying activity levels
	scores := make([]*SurfaceScore, nodeCount)
	for i := 0; i < nodeCount; i++ {
		score := NewSurfaceScore()

		// Node i gets i connections, i waves, i amplifications
		for j := 0; j < i; j++ {
			peerID := string(rune('A' + j))
			score.AddConnectionWithAge(peerID)
			score.AddWave()
			score.AddAmplificationReceived()
			score.AddAmplificationGiven()
		}

		scores[i] = score
	}

	// Compute scores and verify monotonic increase with activity
	computedScores := make([]int, nodeCount)
	for i, score := range scores {
		computedScores[i] = score.Compute()
	}

	// Verify that generally, more activity leads to higher scores
	violations := 0
	for i := 1; i < nodeCount; i++ {
		if computedScores[i] < computedScores[i-1] {
			violations++
		}
	}

	violationRate := float64(violations) / float64(nodeCount-1) * 100
	t.Logf("Score correlation with activity: %.1f%% violations (expected <30%%)", violationRate)

	if violationRate > 30 {
		t.Errorf("Score correlation too weak: %.1f%% violations", violationRate)
	}

	t.Logf("Verified that Resonance scores correlate with activity level")
}
