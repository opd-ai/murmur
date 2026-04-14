package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if db.Path() != path {
		t.Errorf("Path() = %q, want %q", db.Path(), path)
	}

	if err := db.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify file was created.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestPutGet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	key := []byte("test-key")
	value := []byte("test-value")

	// Put a value.
	if err := db.Put(BucketConfig, key, value); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Get the value.
	got, err := db.Get(BucketConfig, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if string(got) != string(value) {
		t.Errorf("Get() = %q, want %q", got, value)
	}
}

func TestGetNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	got, err := db.Get(BucketConfig, []byte("nonexistent"))
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != nil {
		t.Errorf("Get() = %q, want nil for nonexistent key", got)
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	key := []byte("delete-key")
	value := []byte("delete-value")

	// Put a value.
	if err := db.Put(BucketConfig, key, value); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// Delete the value.
	if err := db.Delete(BucketConfig, key); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone.
	got, err := db.Get(BucketConfig, key)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got != nil {
		t.Errorf("Get() = %q after Delete(), want nil", got)
	}
}

func TestAllBucketsCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	buckets := [][]byte{
		BucketIdentity,
		BucketPeers,
		BucketWaves,
		BucketThreads,
		BucketShroud,
		BucketResonance,
		BucketConfig,
	}

	// Verify each bucket exists by putting a test value.
	for _, bucket := range buckets {
		err := db.Put(bucket, []byte("test"), []byte("value"))
		if err != nil {
			t.Errorf("Put to bucket %s failed: %v", bucket, err)
		}
	}
}

func TestPrefixScan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Populate test data with different prefixes.
	testData := map[string]string{
		"user:alice":   "alice-data",
		"user:bob":     "bob-data",
		"user:charlie": "charlie-data",
		"peer:node1":   "node1-data",
		"peer:node2":   "node2-data",
	}

	for k, v := range testData {
		if err := db.Put(BucketConfig, []byte(k), []byte(v)); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	t.Run("scan user prefix", func(t *testing.T) {
		var keys []string
		err := db.PrefixScan(BucketConfig, []byte("user:"), func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			t.Fatalf("PrefixScan() error = %v", err)
		}

		if len(keys) != 3 {
			t.Errorf("PrefixScan() returned %d keys, want 3", len(keys))
		}
	})

	t.Run("scan peer prefix", func(t *testing.T) {
		var keys []string
		err := db.PrefixScan(BucketConfig, []byte("peer:"), func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			t.Fatalf("PrefixScan() error = %v", err)
		}

		if len(keys) != 2 {
			t.Errorf("PrefixScan() returned %d keys, want 2", len(keys))
		}
	})

	t.Run("scan nonexistent prefix", func(t *testing.T) {
		var keys []string
		err := db.PrefixScan(BucketConfig, []byte("nonexistent:"), func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			t.Fatalf("PrefixScan() error = %v", err)
		}

		if len(keys) != 0 {
			t.Errorf("PrefixScan() returned %d keys, want 0", len(keys))
		}
	})
}

func TestRangeScan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Populate test data with ordered keys.
	testData := []struct {
		key   string
		value string
	}{
		{"a", "1"},
		{"b", "2"},
		{"c", "3"},
		{"d", "4"},
		{"e", "5"},
	}

	for _, td := range testData {
		if err := db.Put(BucketConfig, []byte(td.key), []byte(td.value)); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	t.Run("range b to d", func(t *testing.T) {
		var keys []string
		err := db.RangeScan(BucketConfig, []byte("b"), []byte("d"), func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			t.Fatalf("RangeScan() error = %v", err)
		}

		// Should include b, c but not d (exclusive end)
		if len(keys) != 2 {
			t.Errorf("RangeScan() returned %d keys, want 2 (b,c)", len(keys))
		}
	})

	t.Run("range from c to end", func(t *testing.T) {
		var keys []string
		err := db.RangeScan(BucketConfig, []byte("c"), nil, func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		if err != nil {
			t.Fatalf("RangeScan() error = %v", err)
		}

		// Should include c, d, e
		if len(keys) != 3 {
			t.Errorf("RangeScan() returned %d keys, want 3 (c,d,e)", len(keys))
		}
	})
}

func TestCompareBytes(t *testing.T) {
	tests := []struct {
		a, b []byte
		want int
	}{
		{[]byte("a"), []byte("b"), -1},
		{[]byte("b"), []byte("a"), 1},
		{[]byte("a"), []byte("a"), 0},
		{[]byte("ab"), []byte("a"), 1},
		{[]byte("a"), []byte("ab"), -1},
		{nil, nil, 0},
		{[]byte{}, []byte{}, 0},
	}

	for _, tt := range tests {
		got := compareBytes(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareBytes(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	// Empty bucket stats.
	stats, err := db.Stats(BucketConfig)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.KeyCount != 0 {
		t.Errorf("Empty bucket KeyCount = %d, want 0", stats.KeyCount)
	}

	// Add some data.
	testData := map[string]string{
		"key1": "value1",
		"key2": "longer-value-2",
		"key3": "v3",
	}
	for k, v := range testData {
		if err := db.Put(BucketConfig, []byte(k), []byte(v)); err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	}

	// Get stats again.
	stats, err = db.Stats(BucketConfig)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if stats.KeyCount != 3 {
		t.Errorf("KeyCount = %d, want 3", stats.KeyCount)
	}

	// key1+key2+key3 = 4+4+4 = 12 bytes
	expectedKeySize := int64(12)
	if stats.TotalKeySize != expectedKeySize {
		t.Errorf("TotalKeySize = %d, want %d", stats.TotalKeySize, expectedKeySize)
	}

	// value1+longer-value-2+v3 = 6+14+2 = 22 bytes
	expectedValueSize := int64(22)
	if stats.TotalValueSize != expectedValueSize {
		t.Errorf("TotalValueSize = %d, want %d", stats.TotalValueSize, expectedValueSize)
	}

	if stats.TotalSize != expectedKeySize+expectedValueSize {
		t.Errorf("TotalSize = %d, want %d", stats.TotalSize, expectedKeySize+expectedValueSize)
	}
}

func TestAllStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	stats, err := db.AllStats()
	if err != nil {
		t.Fatalf("AllStats() error = %v", err)
	}

	// Check that we get stats for all expected buckets.
	expectedBuckets := []string{
		"identity", "peers", "waves", "threads", "shroud",
		"resonance", "config", "gifts", "marks", "puzzles",
		"hunts", "territories", "oracles", "forge", "shadowplay",
		"councils", "daily_limits",
	}

	for _, bucket := range expectedBuckets {
		if _, ok := stats[bucket]; !ok {
			t.Errorf("AllStats() missing bucket %q", bucket)
		}
	}
}

func TestDatabaseSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	size, err := db.DatabaseSize()
	if err != nil {
		t.Fatalf("DatabaseSize() error = %v", err)
	}

	if size <= 0 {
		t.Errorf("DatabaseSize() = %d, want > 0", size)
	}
}
