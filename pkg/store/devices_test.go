package store

import (
	"os"
	"path/filepath"
	"testing"

	pb "github.com/opd-ai/murmur/proto"
)

func TestDeviceListAccessors(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	masterPubkey := make([]byte, 32)
	masterPubkey[0] = 1

	// Test GetDeviceList on empty database
	list, err := db.GetDeviceList(masterPubkey)
	if err != nil {
		t.Fatalf("GetDeviceList failed: %v", err)
	}
	if list == nil {
		t.Fatal("Expected non-nil device list")
	}
	if len(list.Devices) != 0 {
		t.Fatalf("Expected empty device list, got %d devices", len(list.Devices))
	}

	// Add a device
	devicePubkey := make([]byte, 32)
	devicePubkey[0] = 2
	list.Devices = append(list.Devices, &pb.AuthorizedDevice{
		DevicePublicKey:  devicePubkey,
		DeviceLabel:      "Test Device",
		AuthorizedAtUnix: 1234567890,
		ExpiresAtUnix:    0,
		IsRevoked:        false,
		RevokedAtUnix:    0,
	})

	// Test PutDeviceList
	err = db.PutDeviceList(masterPubkey, list)
	if err != nil {
		t.Fatalf("PutDeviceList failed: %v", err)
	}

	// Test GetDeviceList after put
	retrievedList, err := db.GetDeviceList(masterPubkey)
	if err != nil {
		t.Fatalf("GetDeviceList after put failed: %v", err)
	}
	if len(retrievedList.Devices) != 1 {
		t.Fatalf("Expected 1 device, got %d", len(retrievedList.Devices))
	}
	if string(retrievedList.Devices[0].DevicePublicKey) != string(devicePubkey) {
		t.Fatal("Device pubkey mismatch")
	}
	if retrievedList.Devices[0].DeviceLabel != "Test Device" {
		t.Fatalf("Expected label 'Test Device', got '%s'", retrievedList.Devices[0].DeviceLabel)
	}

	// Test DeleteDeviceList
	err = db.DeleteDeviceList(masterPubkey)
	if err != nil {
		t.Fatalf("DeleteDeviceList failed: %v", err)
	}

	// Verify deletion
	deletedList, err := db.GetDeviceList(masterPubkey)
	if err != nil {
		t.Fatalf("GetDeviceList after delete failed: %v", err)
	}
	if len(deletedList.Devices) != 0 {
		t.Fatalf("Expected empty list after delete, got %d devices", len(deletedList.Devices))
	}
}

func TestDeviceListNilHandling(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	defer os.Remove(dbPath)

	// Test PutDeviceList with nil list
	err = db.PutDeviceList([]byte("key"), nil)
	if err == nil {
		t.Fatal("Expected error for nil device list")
	}

	// Test PutDeviceList with empty master pubkey
	list := &pb.DeviceList{Devices: []*pb.AuthorizedDevice{}}
	err = db.PutDeviceList([]byte{}, list)
	if err == nil {
		t.Fatal("Expected error for empty master pubkey")
	}
}
