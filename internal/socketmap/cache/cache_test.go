package cache

import (
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	ttlSeconds := 60
	c := New(ttlSeconds)

	if c == nil {
		t.Fatal("Expected cache instance, got nil")
	}

	if c.store == nil {
		t.Fatal("Expected cache store to be initialized")
	}

	expectedTTL := time.Duration(ttlSeconds) * time.Second
	if c.GetTTL() != expectedTTL {
		t.Errorf("Expected TTL to be %v, got %v", expectedTTL, c.GetTTL())
	}
}

func TestSetAndGet(t *testing.T) {
	c := New(60)

	key := "test-key"
	entry := Entry{
		Exists:     true,
		Data:       "test-data",
		Expires:    time.Now().Add(time.Minute),
		LastUpdate: time.Now(),
	}

	// Test Set
	c.Set(key, entry)

	// Test Get
	retrieved, found := c.Get(key)
	if !found {
		t.Fatalf("Expected to find key %q in cache", key)
	}

	if retrieved.Data != entry.Data {
		t.Errorf("Expected data %q, got %q", entry.Data, retrieved.Data)
	}
	if retrieved.Exists != entry.Exists {
		t.Errorf("Expected Exists %v, got %v", entry.Exists, retrieved.Exists)
	}

	// Test missing key
	_, foundMissing := c.Get("missing-key")
	if foundMissing {
		t.Error("Expected not to find missing-key")
	}
}

func TestIsExpired(t *testing.T) {
	c := New(60)

	now := time.Now()

	// Not expired entry
	validEntry := Entry{
		Expires: now.Add(1 * time.Minute),
	}
	if c.IsExpired(validEntry) {
		t.Error("Expected entry to not be expired")
	}

	// Expired entry
	expiredEntry := Entry{
		Expires: now.Add(-1 * time.Minute),
	}
	if !c.IsExpired(expiredEntry) {
		t.Error("Expected entry to be expired")
	}
}

func TestConcurrency(t *testing.T) {
	c := New(60)
	var wg sync.WaitGroup

	// Test concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			c.Set("key", Entry{
				Exists:  true,
				Data:    "concurrent-data",
				Expires: time.Now().Add(time.Minute),
			})
		}(i)
	}

	// Test concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get("key")
		}()
	}

	wg.Wait()

	// Ensure at least it is readable without panic and data race.
	_, found := c.Get("key")
	if !found {
		t.Error("Expected 'key' to be present after concurrent writes")
	}
}
