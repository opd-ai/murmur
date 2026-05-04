package pulsemap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookmarkManager_AddAndList(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Add bookmarks
	err = bm.Add("node1", "Alice", 100.0, 200.0)
	require.NoError(t, err)

	err = bm.Add("node2", "Bob", 150.0, 250.0)
	require.NoError(t, err)

	// List bookmarks
	bookmarks := bm.List()
	assert.Len(t, bookmarks, 2)

	// Verify bookmark data
	assert.Equal(t, "node1", bookmarks[0].NodeID)
	assert.Equal(t, "Alice", bookmarks[0].Label)
	assert.Equal(t, 100.0, bookmarks[0].X)
	assert.Equal(t, 200.0, bookmarks[0].Y)
}

func TestBookmarkManager_UpdateExisting(t *testing.T) {
	tmpDir := t.TempDir()
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Add bookmark
	err = bm.Add("node1", "Alice", 100.0, 200.0)
	require.NoError(t, err)

	// Update same node with different label and position
	err = bm.Add("node1", "Alice Updated", 150.0, 250.0)
	require.NoError(t, err)

	// Verify only one bookmark exists with updated data
	bookmarks := bm.List()
	assert.Len(t, bookmarks, 1)
	assert.Equal(t, "Alice Updated", bookmarks[0].Label)
	assert.Equal(t, 150.0, bookmarks[0].X)
	assert.Equal(t, 250.0, bookmarks[0].Y)
}

func TestBookmarkManager_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Add bookmarks
	bm.Add("node1", "Alice", 100.0, 200.0)
	bm.Add("node2", "Bob", 150.0, 250.0)

	// Remove one bookmark
	err = bm.Remove("node1")
	require.NoError(t, err)

	// Verify only one remains
	bookmarks := bm.List()
	assert.Len(t, bookmarks, 1)
	assert.Equal(t, "node2", bookmarks[0].NodeID)

	// Remove non-existent bookmark (should not error)
	err = bm.Remove("nonexistent")
	assert.NoError(t, err)
}

func TestBookmarkManager_Get(t *testing.T) {
	tmpDir := t.TempDir()
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Add bookmark
	bm.Add("node1", "Alice", 100.0, 200.0)

	// Get existing bookmark
	bookmark, found := bm.Get("node1")
	assert.True(t, found)
	assert.Equal(t, "Alice", bookmark.Label)

	// Get non-existent bookmark
	_, found = bm.Get("nonexistent")
	assert.False(t, found)
}

func TestBookmarkManager_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manager and add bookmarks
	bm1, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)
	bm1.Add("node1", "Alice", 100.0, 200.0)
	bm1.Add("node2", "Bob", 150.0, 250.0)

	// Create new manager with same directory (should load persisted bookmarks)
	bm2, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Verify bookmarks were loaded
	bookmarks := bm2.List()
	assert.Len(t, bookmarks, 2)

	// Verify data integrity
	assert.Equal(t, "node1", bookmarks[0].NodeID)
	assert.Equal(t, "node2", bookmarks[1].NodeID)
}

func TestBookmarkManager_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manager in directory with no existing bookmarks file
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Should start with empty list
	bookmarks := bm.List()
	assert.Len(t, bookmarks, 0)
}

func TestBookmarkManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	bm, err := NewBookmarkManager(tmpDir)
	require.NoError(t, err)

	// Add initial bookmark
	bm.Add("node1", "Alice", 100.0, 200.0)

	// Concurrent reads and writes
	done := make(chan bool, 10)
	for i := 0; i < 5; i++ {
		go func(id int) {
			bm.List() // Read
			done <- true
		}(i)
	}
	for i := 0; i < 5; i++ {
		go func(id int) {
			bm.Add("node2", "Bob", 150.0, 250.0) // Write
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify state is consistent
	bookmarks := bm.List()
	assert.Len(t, bookmarks, 2)
}

func TestBookmarkManager_InvalidPath(t *testing.T) {
	// Try to create manager with invalid directory
	_, err := NewBookmarkManager("/nonexistent/invalid/path")
	// Should not error during construction (only on first save)
	assert.NoError(t, err)
}
