package storage

import (
	"strings"
)

// NewInMemoryStorage creates a new InMemory instance.
func NewInMemoryStorage() InMemory {
	return InMemory{
		store: make(map[string][]byte),
	}
}

// InMemory represents an in memory storage for thumbnails
// Can be used during development
type InMemory struct {
	store map[string][]byte
}

func (s InMemory) Stat(key string) bool {
	_, exists := s.store[key]
	return exists
}

// Get loads the thumbnail from memory.
func (s InMemory) Get(key string) ([]byte, error) {
	return s.store[key], nil
}

// Set stores the thumbnail in memory.
func (s InMemory) Put(key string, thumbnail []byte) error {
	s.store[key] = thumbnail
	return nil
}

// BuildKey generates a unique key to store and retrieve the thumbnail.
func (s InMemory) BuildKey(r Request) string {
	parts := []string{
		r.Checksum,
		r.Resolution.String(),
	}

	if r.Characteristic != "" {
		parts = append(parts, r.Characteristic)
	}

	parts = append(parts, strings.Join(r.Types, ","))

	return strings.Join(parts, "+")
}
