package security

import (
	"testing"
)

func TestCryptoAuditorEd25519(t *testing.T) {
	auditor := NewCryptoAuditor()
	if err := auditor.AuditEd25519(); err != nil {
		t.Fatalf("AuditEd25519 failed: %v", err)
	}

	results := auditor.Results()
	if len(results) < 4 {
		t.Errorf("expected at least 4 results, got %d", len(results))
	}

	for _, r := range results {
		if !r.Passed {
			t.Errorf("Ed25519 audit failed: %s - %s", r.Description, r.Details)
		}
	}
}

func TestCryptoAuditorCurve25519(t *testing.T) {
	auditor := NewCryptoAuditor()
	if err := auditor.AuditCurve25519(); err != nil {
		t.Fatalf("AuditCurve25519 failed: %v", err)
	}

	for _, r := range auditor.Results() {
		if !r.Passed {
			t.Errorf("Curve25519 audit failed: %s - %s", r.Description, r.Details)
		}
	}
}

func TestCryptoAuditorChaCha20Poly1305(t *testing.T) {
	auditor := NewCryptoAuditor()
	if err := auditor.AuditChaCha20Poly1305(); err != nil {
		t.Fatalf("AuditChaCha20Poly1305 failed: %v", err)
	}

	for _, r := range auditor.Results() {
		if !r.Passed {
			t.Errorf("ChaCha20-Poly1305 audit failed: %s - %s", r.Description, r.Details)
		}
	}
}

func TestCryptoAuditorRandom(t *testing.T) {
	auditor := NewCryptoAuditor()
	if err := auditor.AuditRandom(); err != nil {
		t.Fatalf("AuditRandom failed: %v", err)
	}

	for _, r := range auditor.Results() {
		if !r.Passed {
			t.Errorf("Random audit failed: %s - %s", r.Description, r.Details)
		}
	}
}

func TestCryptoAuditorFullAudit(t *testing.T) {
	auditor := NewCryptoAuditor()
	results, err := auditor.RunFullAudit()
	if err != nil {
		t.Fatalf("RunFullAudit failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected audit results")
	}

	passed := auditor.PassedCount()
	failed := auditor.FailedCount()

	if failed > 0 {
		t.Errorf("audit had %d failures", failed)
		for _, r := range results {
			if !r.Passed {
				t.Logf("FAILED: %s - %s: %s", r.Category, r.Description, r.Details)
			}
		}
	}

	if passed == 0 {
		t.Error("expected at least some passing audits")
	}

	t.Logf("Audit complete: %d passed, %d failed", passed, failed)
}

func TestKeyStorageAuditor(t *testing.T) {
	auditor := NewKeyStorageAuditor()
	if err := auditor.AuditKeyZeroization(); err != nil {
		t.Fatalf("AuditKeyZeroization failed: %v", err)
	}

	for _, r := range auditor.Results() {
		if !r.Passed {
			t.Errorf("Key storage audit failed: %s - %s", r.Description, r.Details)
		}
	}
}

func TestZeroBytes(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	ZeroBytes(data)

	for i, b := range data {
		if b != 0 {
			t.Errorf("byte %d not zeroed: %d", i, b)
		}
	}
}

func TestZeroKey(t *testing.T) {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}

	ZeroKey(&key)

	for i, b := range key {
		if b != 0 {
			t.Errorf("byte %d not zeroed: %d", i, b)
		}
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		t.Fatalf("GenerateRandomBytes failed: %v", err)
	}

	if len(b) != 32 {
		t.Errorf("expected 32 bytes, got %d", len(b))
	}

	// Check it's not all zeros.
	allZero := true
	for _, v := range b {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("random bytes should not be all zeros")
	}
}

func TestSecureReader(t *testing.T) {
	reader := NewSecureReader()
	buf := make([]byte, 16)

	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 16 {
		t.Errorf("expected 16 bytes, got %d", n)
	}
}

func TestAuditResult(t *testing.T) {
	r := AuditResult{
		Category:    "Test",
		Passed:      true,
		Description: "Test description",
		Details:     "Test details",
	}

	if r.Category != "Test" {
		t.Errorf("Category = %s, want Test", r.Category)
	}
	if !r.Passed {
		t.Error("Passed should be true")
	}
}
