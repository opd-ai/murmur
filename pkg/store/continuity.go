package store

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	pb "github.com/opd-ai/murmur/proto"
	"go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

var (
	// ErrChainNotFound indicates no continuity chain exists for the identity.
	ErrChainNotFound = errors.New("continuity chain not found")
	// ErrIdentityRootMismatch indicates identity root key doesn't match chain.
	ErrIdentityRootMismatch = errors.New("identity root key mismatch")
	// ErrChainLengthExceeded indicates chain exceeds 100 declarations.
	ErrChainLengthExceeded = errors.New("continuity chain length exceeded (max 100)")
)

const (
	// MaxChainLength is the maximum number of rotations allowed per identity.
	// Per KEY_ROTATION.md §Security Analysis, 100 rotations = reasonable for 100+ years.
	MaxChainLength = 100
)

// StoreContinuityDeclaration adds a rotation declaration to an identity's chain.
// If this is the first rotation for identityRoot, creates new chain.
// identityRoot must be 32 bytes (Ed25519 public key).
func (db *DB) StoreContinuityDeclaration(identityRoot []byte, decl *pb.ContinuityDeclaration) error {
	if len(identityRoot) != 32 {
		return fmt.Errorf("store: invalid identity root key size: %d (want 32)", len(identityRoot))
	}
	if decl == nil {
		return errors.New("store: nil continuity declaration")
	}

	return db.bolt.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketContinuityChains)
		if bucket == nil {
			return errors.New("store: continuity_chains bucket not found")
		}

		// Get existing chain or create new
		var chain *pb.ContinuityChain
		data := bucket.Get(identityRoot)
		if data == nil {
			// First rotation for this identity
			chain = &pb.ContinuityChain{
				IdentityRootKey:  identityRoot,
				Declarations:     nil,
				CurrentActiveKey: identityRoot, // Initially root key
				LastUpdatedUnix:  time.Now().Unix(),
			}
		} else {
			// Existing chain - unmarshal
			chain = &pb.ContinuityChain{}
			if err := proto.Unmarshal(data, chain); err != nil {
				return fmt.Errorf("store: unmarshal continuity chain: %w", err)
			}
		}

		// Validate identity root matches
		if !bytes.Equal(chain.IdentityRootKey, identityRoot) {
			return ErrIdentityRootMismatch
		}

		// Check chain length limit
		if len(chain.Declarations) >= MaxChainLength {
			return ErrChainLengthExceeded
		}

		// Check for duplicate declaration (same old_key + new_key pair)
		for _, existing := range chain.Declarations {
			if bytes.Equal(existing.OldPublicKey, decl.OldPublicKey) &&
				bytes.Equal(existing.NewPublicKey, decl.NewPublicKey) {
				// Already stored, idempotent operation
				return nil
			}
		}

		// Append declaration
		chain.Declarations = append(chain.Declarations, decl)

		// Update current active key
		chain.CurrentActiveKey = decl.NewPublicKey

		// Update timestamp
		chain.LastUpdatedUnix = time.Now().Unix()

		// Marshal and store
		data, err := proto.Marshal(chain)
		if err != nil {
			return fmt.Errorf("store: marshal continuity chain: %w", err)
		}

		return bucket.Put(identityRoot, data)
	})
}

// GetContinuityChain retrieves the full rotation history for an identity.
// Returns ErrChainNotFound if no chain exists.
func (db *DB) GetContinuityChain(identityRoot []byte) (*pb.ContinuityChain, error) {
	if len(identityRoot) != 32 {
		return nil, fmt.Errorf("store: invalid identity root key size: %d (want 32)", len(identityRoot))
	}

	var chain *pb.ContinuityChain

	err := db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketContinuityChains)
		if bucket == nil {
			return errors.New("store: continuity_chains bucket not found")
		}

		data := bucket.Get(identityRoot)
		if data == nil {
			return ErrChainNotFound
		}

		chain = &pb.ContinuityChain{}
		if err := proto.Unmarshal(data, chain); err != nil {
			return fmt.Errorf("store: unmarshal continuity chain: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// ResolveCurrentKey returns the current active key for an identity.
// This is a fast O(1) lookup using the cached CurrentActiveKey field.
// Returns ErrChainNotFound if no chain exists.
func (db *DB) ResolveCurrentKey(identityRoot []byte) ([]byte, error) {
	chain, err := db.GetContinuityChain(identityRoot)
	if err != nil {
		return nil, err
	}

	return chain.CurrentActiveKey, nil
}

// IsKeyValid checks if a signing key is valid for the given identity at the specified timestamp.
// Implements the chain resolution algorithm from KEY_ROTATION.md §Continuity Chain Management.
// Returns true if key is valid (current active key or old key within grace period).
func (db *DB) IsKeyValid(identityRoot, signingKey []byte, timestamp int64) (bool, error) {
	if len(identityRoot) != 32 {
		return false, fmt.Errorf("store: invalid identity root key size: %d (want 32)", len(identityRoot))
	}
	if len(signingKey) != 32 {
		return false, fmt.Errorf("store: invalid signing key size: %d (want 32)", len(signingKey))
	}

	chain, err := db.GetContinuityChain(identityRoot)
	if err == ErrChainNotFound {
		// No rotation history; signingKey must equal identityRoot
		return bytes.Equal(signingKey, identityRoot), nil
	}
	if err != nil {
		return false, err
	}

	// Fast path: check if signing key is current active key
	if bytes.Equal(signingKey, chain.CurrentActiveKey) {
		return true, nil
	}

	// Walk chain to find matching declaration
	for _, decl := range chain.Declarations {
		// Check if key is the new key (always valid after rotation)
		if bytes.Equal(signingKey, decl.NewPublicKey) {
			return true, nil
		}

		// Check if key is old key within grace period
		if bytes.Equal(signingKey, decl.OldPublicKey) {
			graceExpiry := decl.RotationTimestampUnix + (decl.GracePeriodDays * 86400)
			if timestamp <= graceExpiry {
				return true, nil // Old key still within grace period
			}
		}
	}

	return false, nil
}

// ResolveIdentityRoot attempts to find the identity root key for a given signing key.
// Walks all continuity chains to find one containing the signing key.
// Returns nil if key not found in any chain.
// This is O(N*M) where N=number of identities, M=chain length, so should be cached.
func (db *DB) ResolveIdentityRoot(signingKey []byte) ([]byte, error) {
	if len(signingKey) != 32 {
		return nil, fmt.Errorf("store: invalid signing key size: %d (want 32)", len(signingKey))
	}

	var identityRoot []byte

	err := db.bolt.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(BucketContinuityChains)
		if bucket == nil {
			return errors.New("store: continuity_chains bucket not found")
		}

		// Walk all chains
		return bucket.ForEach(func(root, data []byte) error {
			chain := &pb.ContinuityChain{}
			if err := proto.Unmarshal(data, chain); err != nil {
				return fmt.Errorf("store: unmarshal continuity chain: %w", err)
			}

			// Check if signingKey is identity root
			if bytes.Equal(signingKey, chain.IdentityRootKey) {
				identityRoot = chain.IdentityRootKey
				return nil // Stop iteration
			}

			// Check if signingKey is current active key
			if bytes.Equal(signingKey, chain.CurrentActiveKey) {
				identityRoot = chain.IdentityRootKey
				return nil // Stop iteration
			}

			// Walk declarations
			for _, decl := range chain.Declarations {
				if bytes.Equal(signingKey, decl.OldPublicKey) || bytes.Equal(signingKey, decl.NewPublicKey) {
					identityRoot = chain.IdentityRootKey
					return nil // Stop iteration
				}
			}

			return nil // Continue to next chain
		})
	})
	if err != nil {
		return nil, err
	}

	return identityRoot, nil
}
