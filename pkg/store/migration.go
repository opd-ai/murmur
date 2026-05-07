//go:build !js

// Package store provides Bbolt-based persistent storage for MURMUR.
// This file implements the schema migration system for version upgrades.
package store

import (
	"fmt"
	"sort"
)

// SchemaVersion represents a database schema version.
type SchemaVersion int

const (
	// SchemaVersionKey is the key used to store the schema version in the config bucket.
	SchemaVersionKey = "schema_version"

	// CurrentSchemaVersion is the latest schema version.
	// Increment this when adding new migrations.
	CurrentSchemaVersion SchemaVersion = 1
)

// Migration represents a database migration.
type Migration struct {
	// Version is the schema version this migration upgrades to.
	Version SchemaVersion
	// Description describes what this migration does.
	Description string
	// Up performs the migration upgrade.
	Up func(db *DB) error
}

// migrationRegistry holds all registered migrations.
var migrationRegistry = []Migration{
	{
		Version:     1,
		Description: "Initial schema - create all buckets",
		Up: func(db *DB) error {
			// Buckets are already created in initBuckets().
			// This migration just marks schema version 1.
			return nil
		},
	},
}

// GetSchemaVersion returns the current schema version of the database.
func (db *DB) GetSchemaVersion() (SchemaVersion, error) {
	data, err := db.Get(BucketConfig, []byte(SchemaVersionKey))
	if err != nil {
		return 0, fmt.Errorf("getting schema version: %w", err)
	}
	if data == nil {
		return 0, nil // No version stored = version 0 (pre-migration)
	}
	if len(data) < 4 {
		return 0, fmt.Errorf("invalid schema version data")
	}
	// Decode as little-endian int32.
	version := SchemaVersion(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24)
	return version, nil
}

// SetSchemaVersion sets the schema version of the database.
func (db *DB) SetSchemaVersion(version SchemaVersion) error {
	// Encode as little-endian int32.
	data := make([]byte, 4)
	v := uint32(version)
	data[0] = byte(v)
	data[1] = byte(v >> 8)
	data[2] = byte(v >> 16)
	data[3] = byte(v >> 24)
	return db.Put(BucketConfig, []byte(SchemaVersionKey), data)
}

// Migrate runs all pending migrations to bring the database up to CurrentSchemaVersion.
// Returns the number of migrations applied.
func (db *DB) Migrate() (int, error) {
	currentVersion, err := db.GetSchemaVersion()
	if err != nil {
		return 0, fmt.Errorf("getting current schema version: %w", err)
	}

	// Sort migrations by version.
	migrations := make([]Migration, len(migrationRegistry))
	copy(migrations, migrationRegistry)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	applied := 0
	for _, m := range migrations {
		if m.Version <= currentVersion {
			continue // Already applied
		}

		// Run migration.
		if err := m.Up(db); err != nil {
			return applied, fmt.Errorf("migration to v%d failed: %w", m.Version, err)
		}

		// Update schema version.
		if err := db.SetSchemaVersion(m.Version); err != nil {
			return applied, fmt.Errorf("setting schema version to %d: %w", m.Version, err)
		}

		applied++
	}

	return applied, nil
}

// NeedsMigration returns true if the database needs migration.
func (db *DB) NeedsMigration() (bool, error) {
	currentVersion, err := db.GetSchemaVersion()
	if err != nil {
		return false, err
	}
	return currentVersion < CurrentSchemaVersion, nil
}

// RegisterMigration adds a migration to the registry.
// This should be called during init() in packages that define migrations.
func RegisterMigration(m Migration) {
	migrationRegistry = append(migrationRegistry, m)
}

// ListPendingMigrations returns descriptions of all migrations that haven't been applied yet.
func (db *DB) ListPendingMigrations() ([]string, error) {
	currentVersion, err := db.GetSchemaVersion()
	if err != nil {
		return nil, err
	}

	var pending []string
	for _, m := range migrationRegistry {
		if m.Version > currentVersion {
			pending = append(pending, fmt.Sprintf("v%d: %s", m.Version, m.Description))
		}
	}
	return pending, nil
}
