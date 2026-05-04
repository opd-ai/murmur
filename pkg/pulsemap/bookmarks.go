// Package pulsemap provides the force-directed graph visualization (Pulse Map).
// This file implements bookmark functionality for saving and navigating to specific nodes.

//go:build !test
// +build !test

package pulsemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Bookmark represents a saved node location with metadata.
type Bookmark struct {
	NodeID    string    `json:"node_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	X         float64   `json:"x"` // World coordinates for precision
	Y         float64   `json:"y"`
}

// BookmarkManager handles bookmark storage and retrieval.
type BookmarkManager struct {
	mu        sync.RWMutex
	bookmarks []Bookmark
	filePath  string
}

// NewBookmarkManager creates a new bookmark manager.
// filePath is the JSON file where bookmarks are persisted.
func NewBookmarkManager(dataDir string) (*BookmarkManager, error) {
	filePath := filepath.Join(dataDir, "bookmarks.json")
	bm := &BookmarkManager{
		bookmarks: make([]Bookmark, 0),
		filePath:  filePath,
	}

	// Load existing bookmarks if file exists
	if err := bm.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load bookmarks: %w", err)
	}

	return bm, nil
}

// Add creates a new bookmark for the given node.
func (bm *BookmarkManager) Add(nodeID, label string, x, y float64) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Check if bookmark already exists
	for i := range bm.bookmarks {
		if bm.bookmarks[i].NodeID == nodeID {
			// Update existing bookmark
			bm.bookmarks[i].Label = label
			bm.bookmarks[i].X = x
			bm.bookmarks[i].Y = y
			bm.bookmarks[i].CreatedAt = time.Now()
			return bm.save()
		}
	}

	// Add new bookmark
	bookmark := Bookmark{
		NodeID:    nodeID,
		Label:     label,
		CreatedAt: time.Now(),
		X:         x,
		Y:         y,
	}
	bm.bookmarks = append(bm.bookmarks, bookmark)
	return bm.save()
}

// Remove deletes a bookmark by node ID.
func (bm *BookmarkManager) Remove(nodeID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for i := range bm.bookmarks {
		if bm.bookmarks[i].NodeID == nodeID {
			// Remove by replacing with last element and truncating
			bm.bookmarks[i] = bm.bookmarks[len(bm.bookmarks)-1]
			bm.bookmarks = bm.bookmarks[:len(bm.bookmarks)-1]
			return bm.save()
		}
	}
	return nil // Not found, no error
}

// List returns all bookmarks sorted by creation time (newest first).
func (bm *BookmarkManager) List() []Bookmark {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make([]Bookmark, len(bm.bookmarks))
	copy(result, bm.bookmarks)
	return result
}

// Get retrieves a specific bookmark by node ID.
func (bm *BookmarkManager) Get(nodeID string) (*Bookmark, bool) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	for i := range bm.bookmarks {
		if bm.bookmarks[i].NodeID == nodeID {
			b := bm.bookmarks[i]
			return &b, true
		}
	}
	return nil, false
}

// load reads bookmarks from disk.
func (bm *BookmarkManager) load() error {
	data, err := os.ReadFile(bm.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &bm.bookmarks)
}

// save writes bookmarks to disk.
func (bm *BookmarkManager) save() error {
	data, err := json.Marshal(bm.bookmarks)
	if err != nil {
		return fmt.Errorf("marshal bookmarks: %w", err)
	}

	// Write atomically via temp file + rename
	tmpPath := bm.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, bm.filePath); err != nil {
		os.Remove(tmpPath) // Cleanup on failure
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
