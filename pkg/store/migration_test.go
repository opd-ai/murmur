package store

import (
	"path/filepath"
	"testing"
)

func TestGetSetSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Initially should be 0.
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion() error: %v", err)
	}
	if version != 0 {
		t.Errorf("initial GetSchemaVersion() = %d, want 0", version)
	}

	// Set version.
	if err := db.SetSchemaVersion(42); err != nil {
		t.Fatalf("SetSchemaVersion() error: %v", err)
	}

	// Read it back.
	version, err = db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion() error: %v", err)
	}
	if version != 42 {
		t.Errorf("GetSchemaVersion() = %d, want 42", version)
	}
}

func TestMigrate(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Run migrations.
	applied, err := db.Migrate()
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Should apply at least the initial migration.
	if applied < 1 {
		t.Errorf("Migrate() applied %d migrations, want >= 1", applied)
	}

	// Version should now be CurrentSchemaVersion.
	version, err := db.GetSchemaVersion()
	if err != nil {
		t.Fatalf("GetSchemaVersion() error: %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("GetSchemaVersion() = %d, want %d", version, CurrentSchemaVersion)
	}

	// Running again should apply 0 migrations.
	applied, err = db.Migrate()
	if err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if applied != 0 {
		t.Errorf("second Migrate() applied %d migrations, want 0", applied)
	}
}

func TestNeedsMigration(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Fresh database needs migration.
	needs, err := db.NeedsMigration()
	if err != nil {
		t.Fatalf("NeedsMigration() error: %v", err)
	}
	if !needs {
		t.Error("fresh database NeedsMigration() = false, want true")
	}

	// After migration, should not need migration.
	if _, err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	needs, err = db.NeedsMigration()
	if err != nil {
		t.Fatalf("NeedsMigration() error: %v", err)
	}
	if needs {
		t.Error("after Migrate() NeedsMigration() = true, want false")
	}
}

func TestListPendingMigrations(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	// Fresh database should have pending migrations.
	pending, err := db.ListPendingMigrations()
	if err != nil {
		t.Fatalf("ListPendingMigrations() error: %v", err)
	}
	if len(pending) < 1 {
		t.Error("fresh database ListPendingMigrations() returned empty list")
	}

	// After migration, should have no pending.
	if _, err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	pending, err = db.ListPendingMigrations()
	if err != nil {
		t.Fatalf("ListPendingMigrations() error: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("after Migrate() ListPendingMigrations() = %v, want empty", pending)
	}
}
