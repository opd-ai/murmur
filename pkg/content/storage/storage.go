// Package storage provides local Wave caching and garbage collection.
// Per TECHNICAL_IMPLEMENTATION.md §1.5, Waves are stored in Bbolt
// with TTL metadata for expiration.
package storage

import "time"

// GCInterval is the interval between garbage collection runs.
const GCInterval = 60 * time.Second

// TODO: Implement Wave storage per PLAN.md Step 6.
