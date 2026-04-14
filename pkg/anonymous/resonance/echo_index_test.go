package resonance

import (
	"testing"
	"time"
)

func TestCategoryFromEchoIndex(t *testing.T) {
	tests := []struct {
		index    float64
		category EchoCategory
	}{
		{0.0, EchoCategoryOpen},
		{0.3, EchoCategoryOpen},
		{0.4, EchoCategoryOpen},
		{0.41, EchoCategoryNeutral},
		{0.5, EchoCategoryNeutral},
		{0.6, EchoCategoryNeutral},
		{0.69, EchoCategoryNeutral},
		{0.7, EchoCategoryInsular},
		{0.8, EchoCategoryInsular},
		{1.0, EchoCategoryInsular},
	}

	for _, tt := range tests {
		got := CategoryFromEchoIndex(tt.index)
		if got != tt.category {
			t.Errorf("CategoryFromEchoIndex(%f) = %v, want %v",
				tt.index, got, tt.category)
		}
	}
}

func TestEchoCategoryString(t *testing.T) {
	tests := []struct {
		category EchoCategory
		want     string
	}{
		{EchoCategoryNeutral, "Neutral"},
		{EchoCategoryInsular, "Insular"},
		{EchoCategoryOpen, "Open"},
	}

	for _, tt := range tests {
		got := tt.category.String()
		if got != tt.want {
			t.Errorf("%v.String() = %s, want %s", tt.category, got, tt.want)
		}
	}
}

func TestNewEchoIndexComputer(t *testing.T) {
	computer := NewEchoIndexComputer()
	if computer == nil {
		t.Fatal("NewEchoIndexComputer() returned nil")
	}
	if computer.windowDays != 30 {
		t.Errorf("Default windowDays = %d, want 30", computer.windowDays)
	}
	if computer.RecordCount() != 0 {
		t.Errorf("Initial RecordCount = %d, want 0", computer.RecordCount())
	}
}

func TestNewEchoIndexComputerWithConfig(t *testing.T) {
	cfg := EchoIndexConfig{WindowDays: 60}
	computer := NewEchoIndexComputerWithConfig(cfg)
	if computer.windowDays != 60 {
		t.Errorf("Configured windowDays = %d, want 60", computer.windowDays)
	}

	// Zero should default to 30.
	cfg.WindowDays = 0
	computer = NewEchoIndexComputerWithConfig(cfg)
	if computer.windowDays != 30 {
		t.Errorf("Zero windowDays should default to 30, got %d", computer.windowDays)
	}
}

func TestEchoIndexRecordAmplification(t *testing.T) {
	computer := NewEchoIndexComputer()

	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-1",
		"cluster-a", "cluster-a", // intra-cluster
	)

	if computer.RecordCount() != 1 {
		t.Errorf("RecordCount = %d, want 1", computer.RecordCount())
	}

	computer.RecordAmplificationSimple(
		"node-1", "author-2", "wave-2",
		"cluster-a", "cluster-b", // extra-cluster
	)

	if computer.RecordCount() != 2 {
		t.Errorf("RecordCount = %d, want 2", computer.RecordCount())
	}
}

func TestEchoIndexCompute(t *testing.T) {
	computer := NewEchoIndexComputer()

	// Add intra-cluster amplifications for cluster-a.
	for i := 0; i < 8; i++ {
		computer.RecordAmplificationSimple(
			"node-1", "author-1", "wave-"+string(rune('0'+i)),
			"cluster-a", "cluster-a",
		)
	}

	// Add extra-cluster amplifications for cluster-a.
	for i := 0; i < 2; i++ {
		computer.RecordAmplificationSimple(
			"node-1", "author-2", "wave-extra-"+string(rune('0'+i)),
			"cluster-a", "cluster-b",
		)
	}

	computer.Compute()

	idx := computer.GetClusterIndex("cluster-a")
	if idx == nil {
		t.Fatal("GetClusterIndex returned nil")
	}

	// 8 intra + 2 extra = 10 total, Echo Index = 8/10 = 0.8
	expectedIndex := 0.8
	if idx.EchoIndex != expectedIndex {
		t.Errorf("EchoIndex = %f, want %f", idx.EchoIndex, expectedIndex)
	}

	if idx.Category != EchoCategoryInsular {
		t.Errorf("Category = %v, want Insular", idx.Category)
	}

	if idx.TotalAmps != 10 {
		t.Errorf("TotalAmps = %d, want 10", idx.TotalAmps)
	}
}

func TestEchoIndexOpenCluster(t *testing.T) {
	computer := NewEchoIndexComputer()

	// Add mostly extra-cluster amplifications for cluster-a.
	for i := 0; i < 2; i++ {
		computer.RecordAmplificationSimple(
			"node-1", "author-1", "wave-intra-"+string(rune('0'+i)),
			"cluster-a", "cluster-a",
		)
	}

	for i := 0; i < 8; i++ {
		computer.RecordAmplificationSimple(
			"node-1", "author-2", "wave-extra-"+string(rune('0'+i)),
			"cluster-a", "cluster-b",
		)
	}

	computer.Compute()

	idx := computer.GetClusterIndex("cluster-a")
	if idx == nil {
		t.Fatal("GetClusterIndex returned nil")
	}

	// 2 intra + 8 extra = 10 total, Echo Index = 2/10 = 0.2
	expectedIndex := 0.2
	if idx.EchoIndex != expectedIndex {
		t.Errorf("EchoIndex = %f, want %f", idx.EchoIndex, expectedIndex)
	}

	if idx.Category != EchoCategoryOpen {
		t.Errorf("Category = %v, want Open", idx.Category)
	}
}

func TestEchoIndexMultipleClusters(t *testing.T) {
	computer := NewEchoIndexComputer()

	// Cluster-a: insular (8 intra, 2 extra).
	for i := 0; i < 8; i++ {
		computer.RecordAmplificationSimple(
			"node-a1", "author-a1", "wave-a-"+string(rune('0'+i)),
			"cluster-a", "cluster-a",
		)
	}
	for i := 0; i < 2; i++ {
		computer.RecordAmplificationSimple(
			"node-a1", "author-b1", "wave-a-extra-"+string(rune('0'+i)),
			"cluster-a", "cluster-b",
		)
	}

	// Cluster-b: open (2 intra, 8 extra).
	for i := 0; i < 2; i++ {
		computer.RecordAmplificationSimple(
			"node-b1", "author-b1", "wave-b-"+string(rune('0'+i)),
			"cluster-b", "cluster-b",
		)
	}
	for i := 0; i < 8; i++ {
		computer.RecordAmplificationSimple(
			"node-b1", "author-a1", "wave-b-extra-"+string(rune('0'+i)),
			"cluster-b", "cluster-a",
		)
	}

	computer.Compute()

	if computer.ClusterCount() != 2 {
		t.Errorf("ClusterCount = %d, want 2", computer.ClusterCount())
	}

	insular := computer.GetInsularClusters()
	if len(insular) != 1 {
		t.Errorf("GetInsularClusters returned %d, want 1", len(insular))
	}

	open := computer.GetOpenClusters()
	if len(open) != 1 {
		t.Errorf("GetOpenClusters returned %d, want 1", len(open))
	}
}

func TestEchoIndexPruning(t *testing.T) {
	cfg := EchoIndexConfig{WindowDays: 1} // 1 day window for faster testing.
	computer := NewEchoIndexComputerWithConfig(cfg)

	// Add old record.
	oldRecord := AmplificationRecord{
		AmplifierID:      "node-1",
		AuthorID:         "author-1",
		WaveID:           "wave-old",
		Timestamp:        time.Now().Add(-48 * time.Hour), // 2 days old
		AmplifierCluster: "cluster-a",
		AuthorCluster:    "cluster-a",
	}
	computer.RecordAmplification(oldRecord)

	// Add recent record.
	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-new",
		"cluster-a", "cluster-a",
	)

	if computer.RecordCount() != 2 {
		t.Errorf("RecordCount before compute = %d, want 2", computer.RecordCount())
	}

	computer.Compute()

	// Old record should be pruned.
	if computer.RecordCount() != 1 {
		t.Errorf("RecordCount after compute = %d, want 1 (old record pruned)",
			computer.RecordCount())
	}
}

func TestEchoIndexClear(t *testing.T) {
	computer := NewEchoIndexComputer()

	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-1",
		"cluster-a", "cluster-a",
	)
	computer.Compute()

	if computer.RecordCount() == 0 || computer.ClusterCount() == 0 {
		t.Error("Records and clusters should exist before clear")
	}

	computer.Clear()

	if computer.RecordCount() != 0 {
		t.Errorf("RecordCount after clear = %d, want 0", computer.RecordCount())
	}
	if computer.ClusterCount() != 0 {
		t.Errorf("ClusterCount after clear = %d, want 0", computer.ClusterCount())
	}
}

func TestEchoIndexGetAllIndices(t *testing.T) {
	computer := NewEchoIndexComputer()

	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-1",
		"cluster-a", "cluster-a",
	)
	computer.RecordAmplificationSimple(
		"node-2", "author-2", "wave-2",
		"cluster-b", "cluster-b",
	)

	computer.Compute()

	all := computer.GetAllIndices()
	if len(all) != 2 {
		t.Errorf("GetAllIndices returned %d indices, want 2", len(all))
	}

	if _, ok := all["cluster-a"]; !ok {
		t.Error("Missing cluster-a in GetAllIndices")
	}
	if _, ok := all["cluster-b"]; !ok {
		t.Error("Missing cluster-b in GetAllIndices")
	}
}

func TestEchoIndexLastComputedTime(t *testing.T) {
	computer := NewEchoIndexComputer()

	if !computer.LastComputedTime().IsZero() {
		t.Error("LastComputedTime should be zero before Compute")
	}

	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-1",
		"cluster-a", "cluster-a",
	)
	computer.Compute()

	if computer.LastComputedTime().IsZero() {
		t.Error("LastComputedTime should be set after Compute")
	}
}

func TestEchoShadow(t *testing.T) {
	shadow := NewEchoShadow()
	if shadow == nil {
		t.Fatal("NewEchoShadow() returned nil")
	}

	// Record anonymous amplifications.
	shadow.RecordSpecterAmplification(
		"specter-1", "specter-author-1", "wave-1",
		"anon-cluster-a", "anon-cluster-a",
	)

	shadow.Compute()

	idx := shadow.GetSpecterClusterShadow("anon-cluster-a")
	if idx == nil {
		t.Fatal("GetSpecterClusterShadow returned nil")
	}

	if idx.EchoIndex != 1.0 {
		t.Errorf("Echo Shadow = %f, want 1.0 (perfect echo chamber)", idx.EchoIndex)
	}
}

func TestColorForEchoIndex(t *testing.T) {
	tests := []struct {
		index float64
		desc  string
	}{
		{1.0, "maximum insular"},
		{0.8, "insular"},
		{0.7, "insular threshold"},
		{0.5, "neutral"},
		{0.4, "open threshold"},
		{0.2, "open"},
		{0.0, "maximum open"},
	}

	for _, tt := range tests {
		r, g, b := ColorForEchoIndex(tt.index)
		// Just verify we get some color values.
		if r == 0 && g == 0 && b == 0 && tt.index != 0.5 {
			t.Errorf("ColorForEchoIndex(%f) returned black, desc: %s",
				tt.index, tt.desc)
		}
		t.Logf("EchoIndex %f (%s): RGB(%d, %d, %d)", tt.index, tt.desc, r, g, b)
	}

	// Verify specific colors.
	r, g, b := ColorForEchoIndex(1.0)
	if r != 255 || g != 0 || b != 0 {
		t.Errorf("Maximum insular should be red, got RGB(%d, %d, %d)", r, g, b)
	}

	r, g, b = ColorForEchoIndex(0.5)
	if r != 128 || g != 128 || b != 128 {
		t.Errorf("Neutral should be gray, got RGB(%d, %d, %d)", r, g, b)
	}

	r, g, b = ColorForEchoIndex(0.0)
	if r != 0 || g != 255 || b != 0 {
		t.Errorf("Maximum open should be green, got RGB(%d, %d, %d)", r, g, b)
	}
}

func TestDefaultEchoIndexConfig(t *testing.T) {
	cfg := DefaultEchoIndexConfig()
	if cfg.WindowDays != 30 {
		t.Errorf("Default WindowDays = %d, want 30", cfg.WindowDays)
	}
}

func TestEchoIndexEmptyCluster(t *testing.T) {
	computer := NewEchoIndexComputer()
	computer.Compute()

	idx := computer.GetClusterIndex("nonexistent")
	if idx != nil {
		t.Error("GetClusterIndex should return nil for nonexistent cluster")
	}
}

func TestEchoIndexEmptyAmplifierCluster(t *testing.T) {
	computer := NewEchoIndexComputer()

	// Record with empty amplifier cluster should be skipped.
	computer.RecordAmplificationSimple(
		"node-1", "author-1", "wave-1",
		"", "cluster-a", // empty amplifier cluster
	)

	computer.Compute()

	if computer.ClusterCount() != 0 {
		t.Errorf("ClusterCount = %d, want 0 (empty cluster should be skipped)",
			computer.ClusterCount())
	}
}

func TestEchoIndexPerfectBalance(t *testing.T) {
	computer := NewEchoIndexComputer()

	// Equal intra and extra (5 each = 0.5 Echo Index).
	for i := 0; i < 5; i++ {
		computer.RecordAmplificationSimple(
			"node-1", "author-1", "wave-intra-"+string(rune('0'+i)),
			"cluster-a", "cluster-a",
		)
		computer.RecordAmplificationSimple(
			"node-1", "author-2", "wave-extra-"+string(rune('0'+i)),
			"cluster-a", "cluster-b",
		)
	}

	computer.Compute()

	idx := computer.GetClusterIndex("cluster-a")
	if idx.EchoIndex != 0.5 {
		t.Errorf("EchoIndex = %f, want 0.5 (perfect balance)", idx.EchoIndex)
	}

	if idx.Category != EchoCategoryNeutral {
		t.Errorf("Category = %v, want Neutral", idx.Category)
	}
}
