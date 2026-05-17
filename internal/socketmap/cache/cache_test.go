package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ttlSeconds := 60
	c := New(ttlSeconds)

	if c == nil {
		t.Fatal("Expected New to return a valid Cache instance, got nil")
	}
	if c.store == nil {
		t.Error("Expected store to be initialized, but it was nil")
	}
	expectedTTL := time.Duration(ttlSeconds) * time.Second
	if c.ttl != expectedTTL {
		t.Errorf("Expected TTL to be %v, got %v", expectedTTL, c.ttl)
	}
}

func TestGetTTL(t *testing.T) {
	ttlSeconds := 30
	c := New(ttlSeconds)

	expectedTTL := time.Duration(ttlSeconds) * time.Second
	if ttl := c.GetTTL(); ttl != expectedTTL {
		t.Errorf("Expected GetTTL to return %v, got %v", expectedTTL, ttl)
	}
}

func TestSetAndGet(t *testing.T) {
	c := New(60)

	key := "test-key"
	entry := Entry{
		Exists:     true,
		Data:       "test-data",
		Expires:    time.Now().Add(1 * time.Hour),
		LastUpdate: time.Now(),
	}

	// Test Get on empty cache
	_, found := c.Get(key)
	if found {
		t.Error("Expected found to be false for empty cache")
	}

	// Test Set
	c.Set(key, entry)

	// Test Get on populated cache
	retrievedEntry, found := c.Get(key)
	if !found {
		t.Error("Expected found to be true for existing key")
	}
	if retrievedEntry.Data != entry.Data {
		t.Errorf("Expected retrieved data to be %v, got %v", entry.Data, retrievedEntry.Data)
	}
	if retrievedEntry.Exists != entry.Exists {
		t.Errorf("Expected retrieved Exists to be %v, got %v", entry.Exists, retrievedEntry.Exists)
	}
}

func TestIsExpired(t *testing.T) {
	c := New(60)

	now := time.Now()

	// Test non-expired entry
	futureEntry := Entry{
		Expires: now.Add(1 * time.Hour),
	}
	if c.IsExpired(futureEntry) {
		t.Error("Expected future entry to not be expired")
	}

	// Test expired entry
	pastEntry := Entry{
		Expires: now.Add(-1 * time.Hour),
	}
	if !c.IsExpired(pastEntry) {
		t.Error("Expected past entry to be expired")
	}
}

func TestCacheConcurrency(t *testing.T) {
	c := New(60)
	var wg sync.WaitGroup

	numGoroutines := 100
	numOperations := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", goroutineID, j)
				entry := Entry{
					Data: fmt.Sprintf("data-%d-%d", goroutineID, j),
				}
				c.Set(key, entry)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				// We might read keys that haven't been written yet, which is fine,
				// we just want to ensure there are no data races or deadlocks.
				key := fmt.Sprintf("key-%d-%d", goroutineID, j)
				c.Get(key)
			}
		}(i)
	}

	wg.Wait()
}
