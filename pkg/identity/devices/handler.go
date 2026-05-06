// Package devices implements multi-device identity management for MURMUR.
// This file implements multi-device identity gossip message handling.
package devices

import (
	"context"
	"fmt"

	pb "github.com/opd-ai/murmur/proto"
)

// Store defines the storage interface for device authorizations.
type Store interface {
	AuthorizeDevice(auth *pb.DeviceAuthorizationDeclaration) error
	RevokeDevice(rev *pb.DeviceRevocationDeclaration) error
}

// DeviceHandler processes device authorization and revocation messages.
type DeviceHandler struct {
	store Store
}

// NewDeviceHandler creates a new device handler with the given store.
func NewDeviceHandler(store Store) *DeviceHandler {
	return &DeviceHandler{store: store}
}

// HandleDeviceAuthorization processes a device authorization declaration.
// Per docs/MULTI_DEVICE_IDENTITY.md, this adds a device to an identity's authorized list.
func (h *DeviceHandler) HandleDeviceAuthorization(ctx context.Context, auth *pb.DeviceAuthorizationDeclaration) error {
	if auth == nil {
		return fmt.Errorf("nil device authorization")
	}

	// Store the authorization
	if err := h.store.AuthorizeDevice(auth); err != nil {
		return fmt.Errorf("failed to authorize device: %w", err)
	}

	return nil
}

// HandleDeviceRevocation processes a device revocation declaration.
// Per docs/MULTI_DEVICE_IDENTITY.md, this marks a device as revoked with a grace period.
func (h *DeviceHandler) HandleDeviceRevocation(ctx context.Context, rev *pb.DeviceRevocationDeclaration) error {
	if rev == nil {
		return fmt.Errorf("nil device revocation")
	}

	// Store the revocation
	if err := h.store.RevokeDevice(rev); err != nil {
		return fmt.Errorf("failed to revoke device: %w", err)
	}

	return nil
}
