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
	if err := validateContinuityInput(identityRoot, decl); err != nil {
		return err
	}

	return db.bolt.Update(func(tx *bbolt.Tx) error {
		return db.storeContinuityInTransaction(tx, identityRoot, decl)
	})
}

// validateContinuityInput validates the identity root and declaration.
func validateContinuityInput(identityRoot []byte, decl *pb.ContinuityDeclaration) error {
	if len(identityRoot) != 32 {
		return fmt.Errorf("store: invalid identity root key size: %d (want 32)", len(identityRoot))
	}
	if decl == nil {
		return errors.New("store: nil continuity declaration")
	}
	return nil
}

// storeContinuityInTransaction stores a continuity declaration within a transaction.
func (db *DB) storeContinuityInTransaction(tx *bbolt.Tx, identityRoot []byte, decl *pb.ContinuityDeclaration) error {
	bucket := tx.Bucket(BucketContinuityChains)
	if bucket == nil {
		return errors.New("store: continuity_chains bucket not found")
	}

	chain, err := db.loadOrCreateChain(bucket, identityRoot)
	if err != nil {
		return err
	}

	isDuplicate, err := db.validateChainForDeclaration(chain, identityRoot, decl)
	if err != nil {
		return err
	}
	if isDuplicate {
		return nil
	}

	db.updateChainWithDeclaration(chain, decl)

	return db.marshalAndStoreChain(bucket, identityRoot, chain)
}

// loadOrCreateChain retrieves existing chain or creates new one for first rotation.
func (db *DB) loadOrCreateChain(bucket *bbolt.Bucket, identityRoot []byte) (*pb.ContinuityChain, error) {
	data := bucket.Get(identityRoot)
	if data == nil {
		return &pb.ContinuityChain{
			IdentityRootKey:  identityRoot,
			Declarations:     nil,
			CurrentActiveKey: identityRoot,
			LastUpdatedUnix:  time.Now().Unix(),
		}, nil
	}

	chain := &pb.ContinuityChain{}
	if err := proto.Unmarshal(data, chain); err != nil {
		return nil, fmt.Errorf("store: unmarshal continuity chain: %w", err)
	}
	return chain, nil
}

// validateChainForDeclaration checks chain invariants before adding declaration.
// Returns (isDuplicate, error).
func (db *DB) validateChainForDeclaration(chain *pb.ContinuityChain, identityRoot []byte, decl *pb.ContinuityDeclaration) (bool, error) {
	if !bytes.Equal(chain.IdentityRootKey, identityRoot) {
		return false, ErrIdentityRootMismatch
	}
	if len(chain.Declarations) >= MaxChainLength {
		return false, ErrChainLengthExceeded
	}
	if db.isDuplicateDeclaration(chain, decl) {
		return true, nil
	}
	return false, nil
}

// isDuplicateDeclaration checks if declaration already exists in chain.
func (db *DB) isDuplicateDeclaration(chain *pb.ContinuityChain, decl *pb.ContinuityDeclaration) bool {
	for _, existing := range chain.Declarations {
		if bytes.Equal(existing.OldPublicKey, decl.OldPublicKey) &&
			bytes.Equal(existing.NewPublicKey, decl.NewPublicKey) {
			return true
		}
	}
	return false
}

// updateChainWithDeclaration appends declaration and updates chain metadata.
func (db *DB) updateChainWithDeclaration(chain *pb.ContinuityChain, decl *pb.ContinuityDeclaration) {
	chain.Declarations = append(chain.Declarations, decl)
	chain.CurrentActiveKey = decl.NewPublicKey
	chain.LastUpdatedUnix = time.Now().Unix()
}

// marshalAndStoreChain serializes chain and writes to bucket.
func (db *DB) marshalAndStoreChain(bucket *bbolt.Bucket, identityRoot []byte, chain *pb.ContinuityChain) error {
	data, err := proto.Marshal(chain)
	if err != nil {
		return fmt.Errorf("store: marshal continuity chain: %w", err)
	}
	return bucket.Put(identityRoot, data)
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
	if err := db.validateKeyParameters(identityRoot, signingKey); err != nil {
		return false, err
	}

	chain, err := db.GetContinuityChain(identityRoot)
	if err == ErrChainNotFound {
		return bytes.Equal(signingKey, identityRoot), nil
	}
	if err != nil {
		return false, err
	}

	if bytes.Equal(signingKey, chain.CurrentActiveKey) {
		return true, nil
	}

	return db.isKeyValidInChain(signingKey, timestamp, chain), nil
}

// validateKeyParameters checks key size requirements.
func (db *DB) validateKeyParameters(identityRoot, signingKey []byte) error {
	if len(identityRoot) != 32 {
		return fmt.Errorf("store: invalid identity root key size: %d (want 32)", len(identityRoot))
	}
	if len(signingKey) != 32 {
		return fmt.Errorf("store: invalid signing key size: %d (want 32)", len(signingKey))
	}
	return nil
}

// isKeyValidInChain checks if key exists in chain declarations.
func (db *DB) isKeyValidInChain(signingKey []byte, timestamp int64, chain *pb.ContinuityChain) bool {
	for _, decl := range chain.Declarations {
		if bytes.Equal(signingKey, decl.NewPublicKey) {
			return true
		}
		if db.isOldKeyInGracePeriod(signingKey, timestamp, decl) {
			return true
		}
	}
	return false
}

// isOldKeyInGracePeriod checks if old key is still within grace period.
func (db *DB) isOldKeyInGracePeriod(signingKey []byte, timestamp int64, decl *pb.ContinuityDeclaration) bool {
	if !bytes.Equal(signingKey, decl.OldPublicKey) {
		return false
	}
	graceExpiry := decl.RotationTimestampUnix + (decl.GracePeriodDays * 86400)
	return timestamp <= graceExpiry
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

		return bucket.ForEach(func(root, data []byte) error {
			chain := &pb.ContinuityChain{}
			if err := proto.Unmarshal(data, chain); err != nil {
				return fmt.Errorf("store: unmarshal continuity chain: %w", err)
			}

			if db.signingKeyMatchesChain(signingKey, chain) {
				identityRoot = chain.IdentityRootKey
				return nil
			}

			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	return identityRoot, nil
}

// signingKeyMatchesChain checks if signing key belongs to chain.
func (db *DB) signingKeyMatchesChain(signingKey []byte, chain *pb.ContinuityChain) bool {
	if bytes.Equal(signingKey, chain.IdentityRootKey) {
		return true
	}
	if bytes.Equal(signingKey, chain.CurrentActiveKey) {
		return true
	}
	return db.signingKeyInDeclarations(signingKey, chain.Declarations)
}

// signingKeyInDeclarations checks if key appears in any declaration.
func (db *DB) signingKeyInDeclarations(signingKey []byte, declarations []*pb.ContinuityDeclaration) bool {
	for _, decl := range declarations {
		if bytes.Equal(signingKey, decl.OldPublicKey) || bytes.Equal(signingKey, decl.NewPublicKey) {
			return true
		}
	}
	return false
}
