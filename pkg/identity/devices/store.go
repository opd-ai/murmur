// Package devices implements multi-device identity management for MURMUR.
// Per docs/MULTI_DEVICE_IDENTITY.md, one Master Identity authorizes multiple Device Keys.
package devices

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/opd-ai/murmur/proto"
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
	db interface {
		GetDeviceList(masterPubkey []byte) (*proto.DeviceList, error)
		PutDeviceList(masterPubkey []byte, list *proto.DeviceList) error
	}
}

// NewDeviceStore creates a new device store with the given database.
func NewDeviceStore(db interface {
	GetDeviceList(masterPubkey []byte) (*proto.DeviceList, error)
	PutDeviceList(masterPubkey []byte, list *proto.DeviceList) error
},
) *DeviceStore {
	return &DeviceStore{db: db}
}

// AuthorizeDevice adds a new device to an identity's authorized device list.
// Validates the authorization declaration signature and enforces device limits.
func (s *DeviceStore) AuthorizeDevice(auth *proto.DeviceAuthorizationDeclaration) error {
	if auth == nil {
		return errors.New("nil authorization declaration")
	}

	now := time.Now().Unix()
	if err := s.validateAuthorizationTiming(auth, now); err != nil {
		return err
	}

	list, err := s.getDeviceList(auth.MasterPublicKey)
	if err != nil {
		return fmt.Errorf("failed to load device list: %w", err)
	}

	if err := s.checkDeviceLimit(list, now); err != nil {
		return err
	}

	s.updateOrAddDevice(list, auth)
	return s.saveDeviceList(auth.MasterPublicKey, list)
}

// validateAuthorizationTiming checks timestamp and expiry per spec (±300 seconds).
func (s *DeviceStore) validateAuthorizationTiming(auth *proto.DeviceAuthorizationDeclaration, now int64) error {
	if abs(now-auth.TimestampUnix) > 300 {
		return fmt.Errorf("authorization timestamp out of range: %d (now: %d)", auth.TimestampUnix, now)
	}
	if auth.ExpiresUnix > 0 && auth.ExpiresUnix < now {
		return ErrDeviceExpired
	}
	return nil
}

// checkDeviceLimit enforces MaxDevicesPerIdentity limit.
func (s *DeviceStore) checkDeviceLimit(list *proto.DeviceList, now int64) error {
	activeCount := s.countActiveDevices(list, now)
	if activeCount >= MaxDevicesPerIdentity {
		return ErrDeviceLimitExceeded
	}
	return nil
}

// countActiveDevices returns number of non-revoked, non-expired devices.
func (s *DeviceStore) countActiveDevices(list *proto.DeviceList, now int64) int {
	count := 0
	for _, dev := range list.Devices {
		if s.isDeviceActive(dev, now) {
			count++
		}
	}
	return count
}

// isDeviceActive checks if device is not revoked and not expired.
func (s *DeviceStore) isDeviceActive(dev *proto.AuthorizedDevice, now int64) bool {
	return !dev.IsRevoked && (dev.ExpiresAtUnix == 0 || dev.ExpiresAtUnix > now)
}

// updateOrAddDevice renews existing device or adds new one.
func (s *DeviceStore) updateOrAddDevice(list *proto.DeviceList, auth *proto.DeviceAuthorizationDeclaration) {
	newDevice := s.createAuthorizedDevice(auth)

	for i, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, auth.DevicePublicKey) {
			list.Devices[i] = newDevice
			return
		}
	}

	list.Devices = append(list.Devices, newDevice)
}

// createAuthorizedDevice builds an AuthorizedDevice from declaration.
func (s *DeviceStore) createAuthorizedDevice(auth *proto.DeviceAuthorizationDeclaration) *proto.AuthorizedDevice {
	return &proto.AuthorizedDevice{
		DevicePublicKey:  auth.DevicePublicKey,
		DeviceLabel:      auth.DeviceLabel,
		AuthorizedAtUnix: auth.TimestampUnix,
		ExpiresAtUnix:    auth.ExpiresUnix,
		IsRevoked:        false,
		RevokedAtUnix:    0,
	}
}

// RevokeDevice marks a device as revoked in the device list.
func (s *DeviceStore) RevokeDevice(rev *proto.DeviceRevocationDeclaration) error {
	if err := s.validateRevocation(rev); err != nil {
		return err
	}

	list, err := s.getDeviceList(rev.MasterPublicKey)
	if err != nil {
		return fmt.Errorf("failed to load device list: %w", err)
	}

	if err := s.markDeviceAsRevoked(list, rev); err != nil {
		return err
	}

	return s.saveDeviceList(rev.MasterPublicKey, list)
}

func (s *DeviceStore) validateRevocation(rev *proto.DeviceRevocationDeclaration) error {
	if rev == nil {
		return errors.New("nil revocation declaration")
	}

	now := time.Now().Unix()
	if abs(now-rev.TimestampUnix) > 300 {
		return fmt.Errorf("revocation timestamp out of range: %d (now: %d)", rev.TimestampUnix, now)
	}
	return nil
}

func (s *DeviceStore) markDeviceAsRevoked(list *proto.DeviceList, rev *proto.DeviceRevocationDeclaration) error {
	for i, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, rev.DevicePublicKeyToRevoke) {
			list.Devices[i].IsRevoked = true
			list.Devices[i].RevokedAtUnix = rev.TimestampUnix
			return nil
		}
	}
	return ErrDeviceNotFound
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

	dev := s.findDevice(list, devicePubKey)
	if dev == nil {
		return false, nil
	}

	return s.isDeviceValidAtTimestamp(dev, waveTimestamp), nil
}

// findDevice locates device by public key in list.
func (s *DeviceStore) findDevice(list *proto.DeviceList, devicePubKey []byte) *proto.AuthorizedDevice {
	for _, dev := range list.Devices {
		if bytes.Equal(dev.DevicePublicKey, devicePubKey) {
			return dev
		}
	}
	return nil
}

// isDeviceValidAtTimestamp checks if device was valid at Wave creation time.
func (s *DeviceStore) isDeviceValidAtTimestamp(dev *proto.AuthorizedDevice, waveTimestamp int64) bool {
	if s.isDeviceExpiredAtTime(dev, waveTimestamp) {
		return false
	}
	if dev.IsRevoked {
		return s.isWithinGracePeriod(dev, waveTimestamp)
	}
	return true
}

// isDeviceExpiredAtTime checks if device was expired at given timestamp.
func (s *DeviceStore) isDeviceExpiredAtTime(dev *proto.AuthorizedDevice, timestamp int64) bool {
	return dev.ExpiresAtUnix > 0 && dev.ExpiresAtUnix < timestamp
}

// isWithinGracePeriod checks if timestamp is within revocation grace period.
func (s *DeviceStore) isWithinGracePeriod(dev *proto.AuthorizedDevice, timestamp int64) bool {
	gracePeriodEnd := dev.RevokedAtUnix + int64(DefaultGracePeriod.Seconds())
	return timestamp < gracePeriodEnd
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
	return s.db.GetDeviceList(masterPubKey)
}

// saveDeviceList persists the device list for a master identity to storage.
func (s *DeviceStore) saveDeviceList(masterPubKey []byte, list *proto.DeviceList) error {
	return s.db.PutDeviceList(masterPubKey, list)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
