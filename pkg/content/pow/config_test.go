package pow

import (
	"sync"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.GetStandard() != StandardDifficulty {
		t.Errorf("Standard = %d, want %d", cfg.GetStandard(), StandardDifficulty)
	}
	if cfg.GetBeacon() != BeaconDifficultyDefault {
		t.Errorf("Beacon = %d, want %d", cfg.GetBeacon(), BeaconDifficultyDefault)
	}
	if cfg.GetAbyssal() != AbyssalDifficultyDefault {
		t.Errorf("Abyssal = %d, want %d", cfg.GetAbyssal(), AbyssalDifficultyDefault)
	}
	if cfg.GetMinAcceptable() != MinAcceptableDifficulty {
		t.Errorf("MinAcceptable = %d, want %d", cfg.GetMinAcceptable(), MinAcceptableDifficulty)
	}
	if cfg.GetMaxAcceptable() != MaxAcceptableDifficulty {
		t.Errorf("MaxAcceptable = %d, want %d", cfg.GetMaxAcceptable(), MaxAcceptableDifficulty)
	}
}

func TestSetStandard(t *testing.T) {
	cfg := DefaultConfig()

	// Valid difficulty should succeed.
	if !cfg.SetStandard(18) {
		t.Error("SetStandard(18) should succeed")
	}
	if cfg.GetStandard() != 18 {
		t.Errorf("Standard = %d, want 18", cfg.GetStandard())
	}

	// Too low difficulty should fail.
	if cfg.SetStandard(10) {
		t.Error("SetStandard(10) should fail (below min)")
	}
	if cfg.GetStandard() != 18 {
		t.Errorf("Standard should not change on failure, got %d", cfg.GetStandard())
	}

	// Too high difficulty should fail.
	if cfg.SetStandard(40) {
		t.Error("SetStandard(40) should fail (above max)")
	}
}

func TestSetBeacon(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.SetBeacon(26) {
		t.Error("SetBeacon(26) should succeed")
	}
	if cfg.GetBeacon() != 26 {
		t.Errorf("Beacon = %d, want 26", cfg.GetBeacon())
	}

	// Below min should fail.
	if cfg.SetBeacon(15) {
		t.Error("SetBeacon(15) should fail")
	}
}

func TestSetAbyssal(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.SetAbyssal(23) {
		t.Error("SetAbyssal(23) should succeed")
	}
	if cfg.GetAbyssal() != 23 {
		t.Errorf("Abyssal = %d, want 23", cfg.GetAbyssal())
	}

	// Above max should fail.
	if cfg.SetAbyssal(35) {
		t.Error("SetAbyssal(35) should fail")
	}
}

func TestValidateDifficulty(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		d        uint8
		expected bool
	}{
		{"at min", MinAcceptableDifficulty, true},
		{"at max", MaxAcceptableDifficulty, true},
		{"middle", 20, true},
		{"below min", 15, false},
		{"above max", 33, false},
		{"zero", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.ValidateDifficulty(tt.d); got != tt.expected {
				t.Errorf("ValidateDifficulty(%d) = %v, want %v", tt.d, got, tt.expected)
			}
		})
	}
}

func TestGlobalConfig(t *testing.T) {
	// Reset before test.
	ResetGlobalConfig()
	defer ResetGlobalConfig()

	cfg := GetGlobalConfig()
	if cfg == nil {
		t.Fatal("GetGlobalConfig() returned nil")
	}

	// Should return same instance.
	cfg2 := GetGlobalConfig()
	if cfg != cfg2 {
		t.Error("GetGlobalConfig() should return same instance")
	}

	// Modification should persist.
	cfg.SetStandard(22)
	if GetGlobalConfig().GetStandard() != 22 {
		t.Error("Global config modification did not persist")
	}
}

func TestConcurrentAccess(t *testing.T) {
	cfg := DefaultConfig()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			d := uint8(17 + (i % 10)) // 17-26 range
			cfg.SetStandard(d)
			_ = cfg.GetStandard()
			_ = cfg.ValidateDifficulty(d)
		}(i)
	}
	wg.Wait()

	// Just verify no panic occurred and value is in valid range.
	std := cfg.GetStandard()
	if std < MinAcceptableDifficulty || std > MaxAcceptableDifficulty {
		t.Errorf("Standard difficulty out of range: %d", std)
	}
}

func TestBoundaryValues(t *testing.T) {
	cfg := DefaultConfig()

	// Test exact boundaries.
	if !cfg.SetStandard(MinAcceptableDifficulty) {
		t.Error("Should accept min difficulty")
	}
	if !cfg.SetStandard(MaxAcceptableDifficulty) {
		t.Error("Should accept max difficulty")
	}

	// Test one below min.
	if cfg.SetStandard(MinAcceptableDifficulty - 1) {
		t.Error("Should reject below min")
	}

	// Test one above max.
	if cfg.SetStandard(MaxAcceptableDifficulty + 1) {
		t.Error("Should reject above max")
	}
}

func TestConfigIndependence(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	cfg1.SetStandard(25)
	if cfg2.GetStandard() != StandardDifficulty {
		t.Error("Config instances should be independent")
	}
}
