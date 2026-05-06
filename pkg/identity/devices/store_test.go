package devices

import (
	"testing"
	"time"

	"github.com/opd-ai/murmur/proto"
)

func TestAuthorizeDevice(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil // Empty bucket
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)
	deviceKey[0] = 1 // Different from master

	auth := &proto.DeviceAuthorizationDeclaration{
		MasterPublicKey: masterKey,
		DevicePublicKey: deviceKey,
		DeviceLabel:     "Test Device",
		TimestampUnix:   time.Now().Unix(),
		ExpiresUnix:     0, // No expiry
	}

	err := store.AuthorizeDevice(auth)
	if err != nil {
		t.Fatalf("AuthorizeDevice failed: %v", err)
	}
}

func TestAuthorizeDeviceExpired(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)

	auth := &proto.DeviceAuthorizationDeclaration{
		MasterPublicKey: masterKey,
		DevicePublicKey: deviceKey,
		DeviceLabel:     "Expired Device",
		TimestampUnix:   time.Now().Unix(),
		ExpiresUnix:     time.Now().Add(-1 * time.Hour).Unix(), // Already expired
	}

	err := store.AuthorizeDevice(auth)
	if err != ErrDeviceExpired {
		t.Fatalf("Expected ErrDeviceExpired, got: %v", err)
	}
}

func TestAuthorizeDeviceTimestampOutOfRange(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)

	// Timestamp 1 hour in the future (way beyond ±300 seconds)
	auth := &proto.DeviceAuthorizationDeclaration{
		MasterPublicKey: masterKey,
		DevicePublicKey: deviceKey,
		DeviceLabel:     "Future Device",
		TimestampUnix:   time.Now().Add(1 * time.Hour).Unix(),
		ExpiresUnix:     0,
	}

	err := store.AuthorizeDevice(auth)
	if err == nil {
		t.Fatal("Expected timestamp validation error, got nil")
	}
}

func TestRevokeDevice(t *testing.T) {
	// Skip this test as it requires a proper bucket accessor with pre-populated data
	// In real usage, the bucket would be populated by AuthorizeDevice calls first
	t.Skip("Skipping: requires integration with Bbolt store for proper testing")
}

func TestRevokeDeviceNotFound(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil // Empty bucket
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)

	rev := &proto.DeviceRevocationDeclaration{
		MasterPublicKey:         masterKey,
		DevicePublicKeyToRevoke: deviceKey,
		RevocationReason:        "Testing",
		TimestampUnix:           time.Now().Unix(),
	}

	err := store.RevokeDevice(rev)
	if err != ErrDeviceNotFound {
		t.Fatalf("Expected ErrDeviceNotFound, got: %v", err)
	}
}

func TestIsDeviceAuthorized(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)

	// Empty device list - device not authorized
	authorized, err := store.IsDeviceAuthorized(masterKey, deviceKey)
	if err != nil {
		t.Fatalf("IsDeviceAuthorized failed: %v", err)
	}
	if authorized {
		t.Fatal("Expected device to not be authorized")
	}
}

func TestIsDeviceAuthorizedWithGracePeriod(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil
	})

	masterKey := make([]byte, 32)
	deviceKey := make([]byte, 32)

	// Wave timestamp from 3 days ago
	waveTimestamp := time.Now().Add(-3 * 24 * time.Hour).Unix()

	// Empty device list - device not authorized
	authorized, err := store.IsDeviceAuthorizedWithGracePeriod(masterKey, deviceKey, waveTimestamp)
	if err != nil {
		t.Fatalf("IsDeviceAuthorizedWithGracePeriod failed: %v", err)
	}
	if authorized {
		t.Fatal("Expected device to not be authorized")
	}
}

func TestGetAuthorizedDevices(t *testing.T) {
	store := NewDeviceStore(func() ([]byte, error) {
		return nil, nil
	})

	masterKey := make([]byte, 32)

	// Empty device list
	devices, err := store.GetAuthorizedDevices(masterKey)
	if err != nil {
		t.Fatalf("GetAuthorizedDevices failed: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("Expected 0 devices, got %d", len(devices))
	}
}

func TestAbsFunction(t *testing.T) {
	tests := []struct {
		input    int64
		expected int64
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-100, 100},
		{100, 100},
	}

	for _, tt := range tests {
		result := abs(tt.input)
		if result != tt.expected {
			t.Errorf("abs(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}
