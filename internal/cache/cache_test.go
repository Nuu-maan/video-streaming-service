package cache

import (
	"testing"
	"time"
)

// newTestLocalCache builds a LocalCache with no background sweeper, so the tests
// observe only what Get/Set/Delete do. No Redis client is involved.
func newTestLocalCache(max int) *LocalCache {
	return &LocalCache{
		items: make(map[string]*cacheItem, max),
		max:   max,
		stop:  make(chan struct{}),
	}
}

func TestLocalCacheSetGet(t *testing.T) {
	tests := []struct {
		name      string
		seed      map[string]string
		lookup    string
		wantValue string
		wantFound bool
	}{
		{
			name:      "round trip",
			seed:      map[string]string{"alpha": "one"},
			lookup:    "alpha",
			wantValue: "one",
			wantFound: true,
		},
		{
			name:      "absent key is a miss",
			seed:      map[string]string{"alpha": "one"},
			lookup:    "beta",
			wantFound: false,
		},
		{
			name:      "miss on an empty cache",
			seed:      nil,
			lookup:    "alpha",
			wantFound: false,
		},
		{
			name:      "empty key is a distinct key",
			seed:      map[string]string{"": "empty-key-value"},
			lookup:    "",
			wantValue: "empty-key-value",
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestLocalCache(16)
			for k, v := range tt.seed {
				c.Set(k, []byte(v), time.Minute)
			}

			got, found := c.Get(tt.lookup)
			if found != tt.wantFound {
				t.Fatalf("Get(%q) found = %v, want %v", tt.lookup, found, tt.wantFound)
			}
			if !tt.wantFound {
				if got != nil {
					t.Errorf("Get(%q) returned %q on a miss, want nil", tt.lookup, got)
				}
				return
			}
			if string(got) != tt.wantValue {
				t.Errorf("Get(%q) = %q, want %q", tt.lookup, got, tt.wantValue)
			}
		})
	}
}

func TestLocalCacheOverwriteReplacesValue(t *testing.T) {
	c := newTestLocalCache(16)

	c.Set("key", []byte("first"), time.Minute)
	c.Set("key", []byte("second"), time.Minute)

	got, found := c.Get("key")
	if !found {
		t.Fatal("Get() found = false after an overwrite, want true")
	}
	if string(got) != "second" {
		t.Errorf("Get() = %q, want the overwritten value %q", got, "second")
	}
}

func TestLocalCacheExpiry(t *testing.T) {
	c := newTestLocalCache(16)

	c.Set("ephemeral", []byte("value"), time.Millisecond)
	c.Set("durable", []byte("value"), time.Minute)

	// Confirm the entry is live before it expires, so this test cannot pass
	// merely because the Set never landed.
	if _, found := c.Get("ephemeral"); !found {
		t.Fatal("Get() found = false immediately after Set, want true")
	}

	time.Sleep(5 * time.Millisecond)

	if got, found := c.Get("ephemeral"); found {
		t.Errorf("Get() returned an expired entry %q, want a miss", got)
	}
	if _, found := c.Get("durable"); !found {
		t.Error("Get() missed a live entry; only the expired one should have been dropped")
	}

	// The expired entry is dropped on read, not merely hidden, so it cannot sit
	// in a slot and be chosen as an eviction victim ahead of live entries.
	if got := c.Len(); got != 1 {
		t.Errorf("Len() = %d after reading an expired entry, want 1", got)
	}
}

// TestLocalCacheOverwriteDoesNotEvict is the regression test for the size-counter
// bug: occupancy was tracked in a hand-maintained field that was incremented on
// every Set, including overwrites of an existing key. The count drifted above the
// real item count and the cache began evicting live entries while genuinely under
// its limit.
func TestLocalCacheOverwriteDoesNotEvict(t *testing.T) {
	c := newTestLocalCache(2)

	c.Set("same", []byte("v1"), time.Minute)
	c.Set("same", []byte("v2"), time.Minute)
	c.Set("same", []byte("v3"), time.Minute)

	if got := c.Len(); got != 1 {
		t.Fatalf("Len() = %d after three Sets of the same key, want 1", got)
	}

	got, found := c.Get("same")
	if !found {
		t.Fatal("Get(\"same\") found = false; the only live entry was evicted")
	}
	if string(got) != "v3" {
		t.Errorf("Get(\"same\") = %q, want the most recent value %q", got, "v3")
	}

	// The cache is still genuinely at one of two slots, so a second distinct key
	// must fit alongside it without evicting anything.
	c.Set("other", []byte("w1"), time.Minute)

	if got := c.Len(); got != 2 {
		t.Fatalf("Len() = %d, want 2", got)
	}
	if _, found := c.Get("same"); !found {
		t.Error("Get(\"same\") found = false; a live entry was evicted while the cache was under its limit")
	}
	if _, found := c.Get("other"); !found {
		t.Error("Get(\"other\") found = false")
	}
}

func TestLocalCacheEvictsWhenFull(t *testing.T) {
	c := newTestLocalCache(2)

	// Staggered TTLs so the eviction victim (the entry closest to expiry) is
	// deterministic: "first" expires soonest.
	c.Set("first", []byte("1"), 1*time.Hour)
	c.Set("second", []byte("2"), 2*time.Hour)

	if got := c.Len(); got != 2 {
		t.Fatalf("Len() = %d after filling the cache, want 2", got)
	}

	c.Set("third", []byte("3"), 3*time.Hour)

	if got := c.Len(); got != 2 {
		t.Fatalf("Len() = %d after inserting a third distinct key into a cache of max 2, want 2", got)
	}

	if _, found := c.Get("first"); found {
		t.Error("Get(\"first\") found = true; the entry closest to expiry should have been evicted")
	}
	for _, key := range []string{"second", "third"} {
		if _, found := c.Get(key); !found {
			t.Errorf("Get(%q) found = false; it should have survived eviction", key)
		}
	}
}

func TestLocalCacheDelete(t *testing.T) {
	c := newTestLocalCache(4)

	c.Set("key", []byte("value"), time.Minute)
	if got := c.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}

	c.Delete("key")

	if _, found := c.Get("key"); found {
		t.Error("Get() found = true after Delete(), want false")
	}
	if got := c.Len(); got != 0 {
		t.Errorf("Len() = %d after Delete(), want 0", got)
	}

	// Deleting an absent key is a no-op, not a panic or an underflow.
	c.Delete("never-existed")
	if got := c.Len(); got != 0 {
		t.Errorf("Len() = %d after deleting an absent key, want 0", got)
	}
}

func TestLocalCacheLen(t *testing.T) {
	c := newTestLocalCache(8)

	if got := c.Len(); got != 0 {
		t.Fatalf("Len() = %d on a fresh cache, want 0", got)
	}

	for _, key := range []string{"a", "b", "c"} {
		c.Set(key, []byte(key), time.Minute)
	}

	if got := c.Len(); got != 3 {
		t.Errorf("Len() = %d, want 3", got)
	}
}
