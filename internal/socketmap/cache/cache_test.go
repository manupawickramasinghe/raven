package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ttlSeconds := 5
	c := New(ttlSeconds)

	if c == nil {
		t.Fatal("Expected New to return a valid Cache instance, got nil")
	}

	if c.store == nil {
		t.Error("Expected internal map to be initialized, got nil")
	}

	expectedTTL := time.Duration(ttlSeconds) * time.Second
	if c.ttl != expectedTTL {
		t.Errorf("Expected TTL %v, got %v", expectedTTL, c.ttl)
	}
}

func TestGetSet(t *testing.T) {
	c := New(10)

	// Test Get on empty cache
	_, found := c.Get("missing_key")
	if found {
		t.Error("Expected Get to return false for missing key")
	}

	// Test Set and Get
	now := time.Now()
	entry := Entry{
		Exists:     true,
		Data:       "test_data",
		Expires:    now.Add(10 * time.Second),
		LastUpdate: now,
	}

	c.Set("key1", entry)

	retrieved, found := c.Get("key1")
	if !found {
		t.Fatal("Expected Get to return true for existing key")
	}
	if retrieved.Data != entry.Data {
		t.Errorf("Expected data %q, got %q", entry.Data, retrieved.Data)
	}
	if retrieved.Exists != entry.Exists {
		t.Errorf("Expected exists %v, got %v", entry.Exists, retrieved.Exists)
	}
	if !retrieved.Expires.Equal(entry.Expires) {
		t.Errorf("Expected expires %v, got %v", entry.Expires, retrieved.Expires)
	}

	// Test Update
	entry.Data = "updated_data"
	c.Set("key1", entry)

	retrieved, found = c.Get("key1")
	if !found {
		t.Fatal("Expected Get to return true for existing key")
	}
	if retrieved.Data != "updated_data" {
		t.Errorf("Expected data %q, got %q", "updated_data", retrieved.Data)
	}
}

func TestIsExpired(t *testing.T) {
	c := New(10)
	now := time.Now()

	expiredEntry := Entry{
		Expires: now.Add(-1 * time.Second),
	}
	if !c.IsExpired(expiredEntry) {
		t.Error("Expected IsExpired to return true for an entry with a past expiration time")
	}

	validEntry := Entry{
		Expires: now.Add(1 * time.Second),
	}
	if c.IsExpired(validEntry) {
		t.Error("Expected IsExpired to return false for an entry with a future expiration time")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New(10)

	// Pre-populate some keys
	keys := []string{"key1", "key2", "key3", "key4"}
	for _, k := range keys {
		c.Set(k, Entry{Data: "initial"})
	}

	var wg sync.WaitGroup
	workers := 100
	iterations := 50

	// Concurrent writers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := keys[j%len(keys)]
				c.Set(key, Entry{Data: "updated"})
				time.Sleep(time.Millisecond) // Slight pause to increase interleaving
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := keys[j%len(keys)]
				c.Get(key)
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	// Wait for all goroutines to finish. If there's a race condition or deadlock, this will hang or crash.
	wg.Wait()
}
