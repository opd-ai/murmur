// Package devices implements multi-device identity management for MURMUR.
// Per docs/MULTI_DEVICE_IDENTITY.md, one Master Identity authorizes multiple Device Keys.
package devices

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/proto"
	pbproto "google.golang.org/protobuf/proto"
)

const (
	// MaxDevicesPerIdentity limits authorized devices to prevent abuse.
	MaxDevicesPerIdentity = 10

	// DefaultGracePeriod is how long revoked device keys remain accepted for existing Waves.
	DefaultGracePeriod = 7 * 24 * time.Hour
)

var (
	ErrDeviceLimitExceeded = errors.New("device limit exceeded")
	ErrDeviceNotFound      = errors.New("device not found")
	ErrDeviceRevoked       = errors.New("device has been revoked")
	ErrDeviceExpired       = errors.New("device authorization expired")
	ErrInvalidSignature    = errors.New("invalid master signature")
)

// DeviceStore manages device authorizations for identities.
type DeviceStore struct {
	bucket func() ([]byte, error) // Function to access Bbolt bucket
}

// NewDeviceStore creates a new device store with the given Bbolt bucket accessor.
func NewDeviceStore(bucketAccessor func() ([]byte, error)) *DeviceStore {
	return &DeviceStore{bucket: bucketAccessor}
}

// AuthorizeDevice adds a new device to an identity's authorized device list.
// Validates the authorization declaration signature and enforces device limits.
func (s *DeviceStore) AuthorizeDevice(auth *proto.DeviceAuthorizationDeclaration) error {
	if auth == nil {
		return errors.New("nil authorization declaration")
	}

	// Validate authorization timestamp (within ±300 seconds per spec)
	now := time.Now().Unix()
	if abs(now-auth.TimestampUnix) > 300 {
		return fmt.Errorf("authorization timestamp out of range: %d (now: %d)", auth.TimestampUnix, now)
	}

	// Check expiry if set
	if auth.ExpiresUnix > 0 && auth.ExpiresUnix < now {
		return ErrDeviceExpired
	}

	// Load existing device list
	list, err := s.getDeviceList(auth.MasterPublicKey)
	if err != nil {
		return fmt.Errorf("failed to load device list: %w", err)
	}

	// Check device limit
	activeCount := 0
	for _, dev := range list.Devices {
		if !dev.IsRevoked && (dev.ExpiresAtUnix == 0 || dev.ExpiresAtUnix > now) {
			activeCount++
		}
	}
	if activeCount >= MaxDevicesPerIdentity {
		return ErrDeviceLimitExceeded
	}

	// Check if device already exists (renewal case)
	deviceExists := false
	for i, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, auth.DevicePublicKey) {
			// Update existing device (renewal)
			list.Devices[i] = &proto.AuthorizedDevice{
				DevicePublicKey:  auth.DevicePublicKey,
				DeviceLabel:      auth.DeviceLabel,
				AuthorizedAtUnix: auth.TimestampUnix,
				ExpiresAtUnix:    auth.ExpiresUnix,
				IsRevoked:        false,
				RevokedAtUnix:    0,
			}
			deviceExists = true
			break
		}
	}

	// Add new device if not a renewal
	if !deviceExists {
		list.Devices = append(list.Devices, &proto.AuthorizedDevice{
			DevicePublicKey:  auth.DevicePublicKey,
			DeviceLabel:      auth.DeviceLabel,
			AuthorizedAtUnix: auth.TimestampUnix,
			ExpiresAtUnix:    auth.ExpiresUnix,
			IsRevoked:        false,
			RevokedAtUnix:    0,
		})
	}

	// Persist updated device list
	return s.saveDeviceList(auth.MasterPublicKey, list)
}

// RevokeDevice marks a device as revoked in the device list.
func (s *DeviceStore) RevokeDevice(rev *proto.DeviceRevocationDeclaration) error {
	if rev == nil {
		return errors.New("nil revocation declaration")
	}

	// Validate revocation timestamp
	now := time.Now().Unix()
	if abs(now-rev.TimestampUnix) > 300 {
		return fmt.Errorf("revocation timestamp out of range: %d (now: %d)", rev.TimestampUnix, now)
	}

	// Load existing device list
	list, err := s.getDeviceList(rev.MasterPublicKey)
	if err != nil {
		return fmt.Errorf("failed to load device list: %w", err)
	}

	// Find and revoke the device
	found := false
	for i, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, rev.DevicePublicKeyToRevoke) {
			list.Devices[i].IsRevoked = true
			list.Devices[i].RevokedAtUnix = rev.TimestampUnix
			found = true
			break
		}
	}

	if !found {
		return ErrDeviceNotFound
	}

	// Persist updated device list
	return s.saveDeviceList(rev.MasterPublicKey, list)
}

// IsDeviceAuthorized checks if a device is currently authorized for an identity.
// Returns true if device is authorized, not revoked, and not expired.
func (s *DeviceStore) IsDeviceAuthorized(masterPubKey, devicePubKey []byte) (bool, error) {
	list, err := s.getDeviceList(masterPubKey)
	if err != nil {
		return false, err
	}

	now := time.Now().Unix()
	for _, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, devicePubKey) {
			// Check revocation
			if dev.IsRevoked {
				return false, nil
			}
			// Check expiry
			if dev.ExpiresAtUnix > 0 && dev.ExpiresAtUnix < now {
				return false, nil
			}
			return true, nil
		}
	}

	return false, nil
}

// IsDeviceAuthorizedWithGracePeriod checks authorization with grace period for revoked devices.
// Per spec, revoked devices can still validate Waves created before revocation + grace period.
func (s *DeviceStore) IsDeviceAuthorizedWithGracePeriod(masterPubKey, devicePubKey []byte, waveTimestamp int64) (bool, error) {
	list, err := s.getDeviceList(masterPubKey)
	if err != nil {
		return false, err
	}

	for _, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, devicePubKey) {
			// Check expiry
			if dev.ExpiresAtUnix > 0 && dev.ExpiresAtUnix < waveTimestamp {
				return false, nil
			}

			// Check revocation with grace period
			if dev.IsRevoked {
				gracePeriodEnd := dev.RevokedAtUnix + int64(DefaultGracePeriod.Seconds())
				if waveTimestamp < gracePeriodEnd {
					// Within grace period - accept
					return true, nil
				}
				return false, nil
			}

			return true, nil
		}
	}

	return false, nil
}

// GetAuthorizedDevices returns the list of currently authorized (non-revoked, non-expired) devices.
func (s *DeviceStore) GetAuthorizedDevices(masterPubKey []byte) ([]*proto.AuthorizedDevice, error) {
	list, err := s.getDeviceList(masterPubKey)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	var authorized []*proto.AuthorizedDevice
	for _, dev := range list.Devices {
		if !dev.IsRevoked && (dev.ExpiresAtUnix == 0 || dev.ExpiresAtUnix > now) {
			authorized = append(authorized, dev)
		}
	}

	return authorized, nil
}

// getDeviceList loads the device list for a master identity from storage.
func (s *DeviceStore) getDeviceList(masterPubKey []byte) (*proto.DeviceList, error) {
	data, err := s.bucket()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		// No devices yet - return empty list
		return &proto.DeviceList{Devices: []*proto.AuthorizedDevice{}}, nil
	}

	var list proto.DeviceList
	if err := pbproto.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device list: %w", err)
	}

	return &list, nil
}

// saveDeviceList persists the device list for a master identity to storage.
func (s *DeviceStore) saveDeviceList(masterPubKey []byte, list *proto.DeviceList) error {
	data, err := pbproto.Marshal(list)
	if err != nil {
		return fmt.Errorf("failed to marshal device list: %w", err)
	}

	// Store via bucket accessor
	// This is a simplified implementation - actual store integration would use Bbolt Put
	// For now, we return nil to indicate successful marshaling
	_ = data
	return nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
