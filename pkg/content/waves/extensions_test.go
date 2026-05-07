package waves

import (
	"errors"
	"testing"

	pb "github.com/opd-ai/murmur/proto"
)

type testExtensionHandler struct {
	err error
}

func (h testExtensionHandler) Validate(_ *pb.Wave) error {
	return h.err
}

func (h testExtensionHandler) Render(_ *pb.Wave) ([]byte, error) {
	return []byte("ok"), nil
}

func TestRegisterWaveType(t *testing.T) {
	resetWaveTypeRegistry()
	t.Cleanup(resetWaveTypeRegistry)

	if err := RegisterWaveType(TypeExtensionStart, testExtensionHandler{}); err != nil {
		t.Fatalf("RegisterWaveType failed: %v", err)
	}

	registered := RegisteredWaveTypes()
	if len(registered) != 1 || registered[0] != TypeExtensionStart {
		t.Fatalf("unexpected registered types: %#v", registered)
	}

	if err := RegisterWaveType(TypeExtensionStart, testExtensionHandler{}); err == nil {
		t.Fatal("expected duplicate registration error")
	}
}

func TestRegisterWaveTypeRejectsInvalidInputs(t *testing.T) {
	resetWaveTypeRegistry()
	t.Cleanup(resetWaveTypeRegistry)

	if err := RegisterWaveType(TypeSurface, testExtensionHandler{}); !errors.Is(err, errReservedWaveType) {
		t.Fatalf("expected reserved type error, got %v", err)
	}
	if err := RegisterWaveType(TypeExtensionStart, nil); !errors.Is(err, errNilExtensionHandle) {
		t.Fatalf("expected nil handler error, got %v", err)
	}
}

func TestValidateExtensionWave(t *testing.T) {
	resetWaveTypeRegistry()
	t.Cleanup(resetWaveTypeRegistry)

	wave := &pb.Wave{WaveType: pb.WaveType(TypeExtensionStart)}
	if err := validateExtensionWave(wave); err == nil {
		t.Fatal("expected missing handler error")
	}

	wantErr := errors.New("boom")
	if err := RegisterWaveType(TypeExtensionStart, testExtensionHandler{err: wantErr}); err != nil {
		t.Fatalf("RegisterWaveType failed: %v", err)
	}
	if err := validateExtensionWave(wave); !errors.Is(err, wantErr) {
		t.Fatalf("expected handler error, got %v", err)
	}
}
