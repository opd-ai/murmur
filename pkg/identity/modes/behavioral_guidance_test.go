package modes

import (
	"testing"
)

func TestNewBehavioralGuidance(t *testing.T) {
	bg := NewBehavioralGuidance()
	if bg == nil {
		t.Fatal("NewBehavioralGuidance returned nil")
	}
}

func TestRecordSurfaceActivity(t *testing.T) {
	bg := NewBehavioralGuidance()

	bg.RecordSurfaceActivity("topic1", 100)
	bg.RecordSurfaceActivity("topic1", 200)
	bg.RecordSurfaceActivity("topic2", 150)

	surface, _ := bg.GetActivitySummary()

	if surface.TotalMessages != 3 {
		t.Errorf("expected 3 messages, got %d", surface.TotalMessages)
	}
	if surface.TopicCounts["topic1"] != 2 {
		t.Errorf("expected 2 topic1 messages, got %d", surface.TopicCounts["topic1"])
	}
	if surface.TopicCounts["topic2"] != 1 {
		t.Errorf("expected 1 topic2 message, got %d", surface.TopicCounts["topic2"])
	}
}

func TestRecordSpecterActivity(t *testing.T) {
	bg := NewBehavioralGuidance()

	bg.RecordSpecterActivity("anon-topic", 50)
	bg.RecordSpecterActivity("anon-topic", 60)

	_, specter := bg.GetActivitySummary()

	if specter.TotalMessages != 2 {
		t.Errorf("expected 2 messages, got %d", specter.TotalMessages)
	}
}

func TestAverageMessageLength(t *testing.T) {
	bg := NewBehavioralGuidance()

	bg.RecordSurfaceActivity("topic", 100)
	bg.RecordSurfaceActivity("topic", 200)
	bg.RecordSurfaceActivity("topic", 300)

	surface, _ := bg.GetActivitySummary()

	// Average should be 200
	if surface.AvgMessageLen != 200 {
		t.Errorf("expected avg length 200, got %f", surface.AvgMessageLen)
	}
}

func TestCorrelationRiskLow(t *testing.T) {
	bg := NewBehavioralGuidance()

	// Record distinct activity patterns (different hours, different topics).
	// This test just verifies the function runs without error on empty data.
	risk := bg.AssessCorrelationRisk()

	if risk != RiskLow {
		t.Errorf("empty data should have low risk, got %s", risk)
	}
}

func TestCorrelationRiskString(t *testing.T) {
	tests := []struct {
		risk CorrelationRisk
		want string
	}{
		{RiskLow, "Low"},
		{RiskMedium, "Medium"},
		{RiskHigh, "High"},
		{CorrelationRisk(99), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.risk.String(); got != tc.want {
			t.Errorf("risk %d: got %s, want %s", tc.risk, got, tc.want)
		}
	}
}

func TestGetRecommendations(t *testing.T) {
	bg := NewBehavioralGuidance()

	recs := bg.GetRecommendations()

	if len(recs) < 5 {
		t.Errorf("expected at least 5 base recommendations, got %d", len(recs))
	}

	// Check for expected categories.
	categories := make(map[string]bool)
	for _, rec := range recs {
		categories[rec.Category] = true
	}

	expectedCategories := []string{"Timing", "Topics", "Style", "Connections", "Devices"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("missing expected category: %s", cat)
		}
	}
}

func TestRecommendationPriorities(t *testing.T) {
	bg := NewBehavioralGuidance()

	recs := bg.GetRecommendations()

	// Check that all priorities are valid (1-3).
	for _, rec := range recs {
		if rec.Priority < 1 || rec.Priority > 3 {
			t.Errorf("invalid priority %d for recommendation %s", rec.Priority, rec.Title)
		}
	}
}

func TestGuidanceMessages(t *testing.T) {
	messages := GuidanceMessages()

	if len(messages) == 0 {
		t.Error("GuidanceMessages returned empty slice")
	}

	for i, msg := range messages {
		if msg == "" {
			t.Errorf("guidance message %d is empty", i)
		}
	}
}

func TestReset(t *testing.T) {
	bg := NewBehavioralGuidance()

	bg.RecordSurfaceActivity("topic", 100)
	bg.RecordSpecterActivity("anon", 50)

	surface, specter := bg.GetActivitySummary()
	if surface.TotalMessages == 0 || specter.TotalMessages == 0 {
		t.Error("activity should be recorded before reset")
	}

	bg.Reset()

	surface, specter = bg.GetActivitySummary()
	if surface.TotalMessages != 0 || specter.TotalMessages != 0 {
		t.Error("activity should be cleared after reset")
	}
}

func TestFindPeakHours(t *testing.T) {
	bg := NewBehavioralGuidance()

	hours := make([]int, 24)
	hours[9] = 10 // Peak 1
	hours[14] = 8 // Peak 2
	hours[20] = 5 // Peak 3
	hours[3] = 1  // Not a peak

	peaks := bg.findPeakHours(hours)

	if len(peaks) != 3 {
		t.Errorf("expected 3 peaks, got %d", len(peaks))
	}

	// Check that peak hours are in correct order (by count).
	if len(peaks) > 0 && peaks[0] != 9 {
		t.Errorf("expected hour 9 as top peak, got %d", peaks[0])
	}
}

func TestCountOverlap(t *testing.T) {
	bg := NewBehavioralGuidance()

	a := []int{1, 2, 3, 4}
	b := []int{3, 4, 5, 6}

	overlap := bg.countOverlap(a, b)
	if overlap != 2 {
		t.Errorf("expected 2 overlapping elements, got %d", overlap)
	}

	// No overlap case.
	c := []int{1, 2}
	d := []int{3, 4}
	if bg.countOverlap(c, d) != 0 {
		t.Error("expected 0 overlap for disjoint sets")
	}
}

func TestTopicOverlapAssessment(t *testing.T) {
	bg := NewBehavioralGuidance()

	// Add distinct topics.
	bg.RecordSurfaceActivity("topic1", 100)
	bg.RecordSurfaceActivity("topic2", 100)
	bg.RecordSpecterActivity("topic3", 100)
	bg.RecordSpecterActivity("topic4", 100)

	overlap := bg.assessTopicOverlap()
	if overlap != 0 {
		t.Errorf("expected 0 overlap risk for distinct topics, got %d", overlap)
	}

	// Add overlapping topics.
	bg.Reset()
	bg.RecordSurfaceActivity("shared", 100)
	bg.RecordSpecterActivity("shared", 100)

	overlap = bg.assessTopicOverlap()
	if overlap == 0 {
		t.Error("expected non-zero overlap risk for shared topics")
	}
}

func TestActivityStatsFields(t *testing.T) {
	bg := NewBehavioralGuidance()

	bg.RecordSurfaceActivity("topic", 100)

	surface, _ := bg.GetActivitySummary()

	// Check that time-based fields are updated.
	if surface.LastActivity.IsZero() {
		t.Error("LastActivity should be set")
	}

	// Check that active hours are updated.
	foundActivity := false
	for _, count := range surface.ActiveHours {
		if count > 0 {
			foundActivity = true
			break
		}
	}
	if !foundActivity {
		t.Error("ActiveHours should have at least one non-zero entry")
	}

	// Check that active days are updated.
	foundDay := false
	for _, count := range surface.ActiveDays {
		if count > 0 {
			foundDay = true
			break
		}
	}
	if !foundDay {
		t.Error("ActiveDays should have at least one non-zero entry")
	}
}
