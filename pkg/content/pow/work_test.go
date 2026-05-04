package pow

import (
	"bytes"
	"testing"
)

func TestCompute(t *testing.T) {
	data := []byte("test data for proof of work")

	// Use lower difficulty for faster tests.
	work, err := Compute(data, 8)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if work.Difficulty != 8 {
		t.Errorf("expected difficulty 8, got %d", work.Difficulty)
	}

	// Verify the computed work.
	if !VerifyWork(data, work) {
		t.Error("computed work failed verification")
	}
}

func TestVerify(t *testing.T) {
	data := []byte("verification test data")

	work, err := Compute(data, 10)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	// Verify should succeed with correct nonce.
	if !Verify(data, work.Nonce, 10) {
		t.Error("verification failed for valid proof")
	}

	// Verify should fail with wrong nonce.
	if Verify(data, work.Nonce+1, 10) {
		t.Error("verification passed for invalid nonce")
	}

	// Verify should fail with different data.
	if Verify([]byte("different data"), work.Nonce, 10) {
		t.Error("verification passed for different data")
	}
}

func TestVerifyWork(t *testing.T) {
	data := []byte("work verification test")

	work, err := Compute(data, 8)
	if err != nil {
		t.Fatalf("Compute failed: %v", err)
	}

	if !VerifyWork(data, work) {
		t.Error("VerifyWork failed for valid work")
	}

	// Nil work should fail.
	if VerifyWork(data, nil) {
		t.Error("VerifyWork passed for nil work")
	}

	// Modified work should fail.
	modifiedWork := &Work{
		Nonce:      work.Nonce + 1,
		Hash:       work.Hash,
		Difficulty: work.Difficulty,
	}
	if VerifyWork(data, modifiedWork) {
		t.Error("VerifyWork passed for modified work")
	}
}

func TestCheckDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		hash       []byte
		difficulty uint8
		expected   bool
	}{
		{
			name:       "zero difficulty always passes",
			hash:       []byte{0xFF, 0xFF, 0xFF, 0xFF},
			difficulty: 0,
			expected:   true,
		},
		{
			name:       "8 zeros with 8 difficulty",
			hash:       []byte{0x00, 0xFF, 0xFF, 0xFF},
			difficulty: 8,
			expected:   true,
		},
		{
			name:       "8 zeros with 9 difficulty fails",
			hash:       []byte{0x00, 0xFF, 0xFF, 0xFF},
			difficulty: 9,
			expected:   false,
		},
		{
			name:       "12 zeros with 10 difficulty",
			hash:       []byte{0x00, 0x0F, 0xFF, 0xFF},
			difficulty: 10,
			expected:   true,
		},
		{
			name:       "16 zeros with 16 difficulty",
			hash:       []byte{0x00, 0x00, 0xFF, 0xFF},
			difficulty: 16,
			expected:   true,
		},
		{
			name:       "all zeros passes any difficulty",
			hash:       []byte{0x00, 0x00, 0x00, 0x00},
			difficulty: 32,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkDifficulty(tt.hash, tt.difficulty)
			if result != tt.expected {
				t.Errorf("checkDifficulty(%v, %d) = %v, want %v",
					tt.hash, tt.difficulty, result, tt.expected)
			}
		})
	}
}

func TestLeadingZeros(t *testing.T) {
	tests := []struct {
		name     string
		hash     []byte
		expected int
	}{
		{
			name:     "no zeros",
			hash:     []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expected: 0,
		},
		{
			name:     "8 leading zeros",
			hash:     []byte{0x00, 0xFF, 0xFF, 0xFF},
			expected: 8,
		},
		{
			name:     "12 leading zeros",
			hash:     []byte{0x00, 0x0F, 0xFF, 0xFF},
			expected: 12,
		},
		{
			name:     "16 leading zeros",
			hash:     []byte{0x00, 0x00, 0xFF, 0xFF},
			expected: 16,
		},
		{
			name:     "24 leading zeros",
			hash:     []byte{0x00, 0x00, 0x00, 0xFF},
			expected: 24,
		},
		{
			name:     "all zeros",
			hash:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LeadingZeros(tt.hash)
			if result != tt.expected {
				t.Errorf("LeadingZeros(%v) = %d, want %d",
					tt.hash, result, tt.expected)
			}
		})
	}
}

func TestComputeDeterminism(t *testing.T) {
	data := []byte("determinism test")

	work1, err := Compute(data, 8)
	if err != nil {
		t.Fatalf("first Compute failed: %v", err)
	}

	work2, err := Compute(data, 8)
	if err != nil {
		t.Fatalf("second Compute failed: %v", err)
	}

	// Same data should produce same nonce.
	if work1.Nonce != work2.Nonce {
		t.Errorf("nonces differ: %d != %d", work1.Nonce, work2.Nonce)
	}

	if !bytes.Equal(work1.Hash[:], work2.Hash[:]) {
		t.Error("hashes differ for same computation")
	}
}

func TestComputeUniqueness(t *testing.T) {
	data1 := []byte("data one")
	data2 := []byte("data two")

	work1, err := Compute(data1, 8)
	if err != nil {
		t.Fatalf("first Compute failed: %v", err)
	}

	work2, err := Compute(data2, 8)
	if err != nil {
		t.Fatalf("second Compute failed: %v", err)
	}

	// Different data should produce different hashes.
	if bytes.Equal(work1.Hash[:], work2.Hash[:]) {
		t.Error("different data produced same hash")
	}
}

// BenchmarkCompute measures PoW computation time.
// Per TECHNICAL_IMPLEMENTATION.md, default difficulty 20 targets 2-5 seconds.
func BenchmarkCompute(b *testing.B) {
	data := []byte("benchmark test data")

	b.Run("difficulty_8", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compute(data, 8)
		}
	})

	b.Run("difficulty_12", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compute(data, 12)
		}
	})

	b.Run("difficulty_15", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compute(data, 15)
		}
	})

	b.Run("difficulty_20_default", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			Compute(data, DefaultDifficulty)
		}
	})
}

// BenchmarkVerify measures PoW verification performance.
func BenchmarkVerify(b *testing.B) {
	data := []byte("benchmark test data")
	work, _ := Compute(data, DefaultDifficulty)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(data, work.Nonce, DefaultDifficulty)
	}
}
