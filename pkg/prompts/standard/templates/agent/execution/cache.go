package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// PromptCache caches generated prompts for performance
type PromptCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	prompt    string
	timestamp time.Time
}

// NewPromptCache creates a new prompt cache
func NewPromptCache(maxSize int, ttl time.Duration) *PromptCache {
	return &PromptCache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a prompt from cache if it exists and is not expired
func (c *PromptCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	// Check if entry is expired
	if time.Since(entry.timestamp) > c.ttl {
		return "", false
	}

	return entry.prompt, true
}

// Set stores a prompt in the cache
func (c *PromptCache) Set(key string, prompt string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entry if cache is full
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &cacheEntry{
		prompt:    prompt,
		timestamp: time.Now(),
	}
}

// GenerateKey creates a cache key from layers and data
func GenerateKey(layers []Layer, data interface{}) (string, error) {
	// Create a deterministic key from layers and data
	keyData := struct {
		Layers []Layer
		Data   interface{}
	}{
		Layers: layers,
		Data:   data,
	}

	jsonBytes, err := json.Marshal(keyData)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// Clear removes all entries from the cache
func (c *PromptCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// evictOldest removes the oldest cache entry
func (c *PromptCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Stats returns cache statistics
func (c *PromptCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	validCount := 0
	now := time.Now()

	for _, entry := range c.entries {
		if now.Sub(entry.timestamp) <= c.ttl {
			validCount++
		}
	}

	return CacheStats{
		TotalEntries: len(c.entries),
		ValidEntries: validCount,
		MaxSize:      c.maxSize,
		TTL:          c.ttl,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	TotalEntries int
	ValidEntries int
	MaxSize      int
	TTL          time.Duration
}
