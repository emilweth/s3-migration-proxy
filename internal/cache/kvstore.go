package kvstore

import (
	"sync"
	"time"
)

type item struct {
	value      string
	expiryTime time.Time
}

// Store represents an in-memory cache of string key-values.
type Store struct {
	items map[string]*item
	mu    sync.RWMutex
}

// New creates a new Store.
func New() *Store {
	return &Store{
		items: make(map[string]*item),
	}
}

// Set adds an item to the cache for a specific duration.
func (s *Store) Set(key string, value string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiryTime := time.Now().Add(duration)
	s.items[key] = &item{
		value:      value,
		expiryTime: expiryTime,
	}
}

// Get retrieves an item from the cache.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, found := s.items[key]
	if !found || time.Now().After(item.expiryTime) {
		return "", false
	}
	return item.value, true
}

// Delete removes an item from the cache.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
}

// Cleanup periodically checks for expired items and removes them.
func (s *Store) Cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			for key, item := range s.items {
				if time.Now().After(item.expiryTime) {
					delete(s.items, key)
				}
			}
			s.mu.Unlock()
		}
	}
}
