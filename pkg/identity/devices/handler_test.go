package devices

import (
	"context"
	"testing"
	"time"

	pb "github.com/opd-ai/murmur/proto"
)

func TestDeviceHandler_HandleDeviceAuthorization(t *testing.T) {
	mockDB := newMockDB()
	store := NewDeviceStore(mockDB)
	handler := NewDeviceHandler(store)

	masterKey := make([]byte, 32)
	masterKey[0] = 1
	deviceKey := make([]byte, 32)
	deviceKey[0] = 2

	auth := &pb.DeviceAuthorizationDeclaration{
		MasterPublicKey: masterKey,
		DevicePublicKey: deviceKey,
		DeviceLabel:     "Test Device",
		TimestampUnix:   time.Now().Unix(),
		ExpiresUnix:     0,
	}

	err := handler.HandleDeviceAuthorization(context.Background(), auth)
	if err != nil {
		t.Fatalf("HandleDeviceAuthorization failed: %v", err)
	}

	// Verify device was stored
	list, err := mockDB.GetDeviceList(masterKey)
	if err != nil {
		t.Fatalf("GetDeviceList failed: %v", err)
	}
	if len(list.Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(list.Devices))
	}
	if string(list.Devices[0].DevicePublicKey) != string(deviceKey) {
		t.Fatal("Device pubkey mismatch")
	}
}

func TestDeviceHandler_HandleDeviceRevocation(t *testing.T) {
	mockDB := newMockDB()
	store := NewDeviceStore(mockDB)
	handler := NewDeviceHandler(store)

	masterKey := make([]byte, 32)
	masterKey[0] = 1
	deviceKey := make([]byte, 32)
	deviceKey[0] = 2

	// First authorize the device
	auth := &pb.DeviceAuthorizationDeclaration{
		MasterPublicKey: masterKey,
		DevicePublicKey: deviceKey,
		DeviceLabel:     "Test Device",
		TimestampUnix:   time.Now().Unix(),
		ExpiresUnix:     0,
	}
	if err := store.AuthorizeDevice(auth); err != nil {
		t.Fatalf("AuthorizeDevice failed: %v", err)
	}

	// Now revoke it
	rev := &pb.DeviceRevocationDeclaration{
		MasterPublicKey:         masterKey,
		DevicePublicKeyToRevoke: deviceKey,
		RevocationReason:        "Test revocation",
		TimestampUnix:           time.Now().Unix(),
	}

	err := handler.HandleDeviceRevocation(context.Background(), rev)
	if err != nil {
		t.Fatalf("HandleDeviceRevocation failed: %v", err)
	}

	// Verify device was revoked
	list, err := mockDB.GetDeviceList(masterKey)
	if err != nil {
		t.Fatalf("GetDeviceList failed: %v", err)
	}
	if len(list.Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(list.Devices))
	}
	if !list.Devices[0].IsRevoked {
		t.Fatal("Device should be revoked")
	}
}

func TestDeviceHandler_HandleNilMessages(t *testing.T) {
	mockDB := newMockDB()
	store := NewDeviceStore(mockDB)
	handler := NewDeviceHandler(store)

	// Test nil authorization
	err := handler.HandleDeviceAuthorization(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil authorization")
	}

	// Test nil revocation
	err = handler.HandleDeviceRevocation(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error for nil revocation")
	}
}
