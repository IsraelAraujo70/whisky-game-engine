package assets

import (
	"errors"
	"sync"
	"testing"
)

func TestGet_CacheMiss(t *testing.T) {
	c := NewCache(0)
	calls := 0
	val, err := Get(c, "hero.png", func(path string) (string, error) {
		calls++
		return "loaded:" + path, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "loaded:hero.png" {
		t.Fatalf("expected loaded:hero.png, got %s", val)
	}
	if calls != 1 {
		t.Fatalf("expected loader called once, got %d", calls)
	}
}

func TestGet_CacheHit(t *testing.T) {
	c := NewCache(0)
	calls := 0
	loader := func(path string) (int, error) {
		calls++
		return 42, nil
	}

	// First call — miss.
	v1, err := Get(c, "x", loader)
	if err != nil {
		t.Fatal(err)
	}
	// Second call — hit.
	v2, err := Get(c, "x", loader)
	if err != nil {
		t.Fatal(err)
	}

	if v1 != 42 || v2 != 42 {
		t.Fatalf("expected 42, got %d %d", v1, v2)
	}
	if calls != 1 {
		t.Fatalf("expected loader called once, got %d", calls)
	}
}

func TestGet_LoaderError(t *testing.T) {
	c := NewCache(0)
	_, err := Get(c, "bad", func(_ string) (int, error) {
		return 0, errors.New("boom")
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
	if c.Len() != 0 {
		t.Fatal("expected cache to remain empty on error")
	}
}

func TestInvalidate(t *testing.T) {
	c := NewCache(0)
	_, _ = Get(c, "a", func(_ string) (int, error) { return 1, nil })
	if c.Len() != 1 {
		t.Fatal("expected 1 entry")
	}
	if !c.Invalidate("a") {
		t.Fatal("expected Invalidate to return true")
	}
	if c.Len() != 0 {
		t.Fatal("expected 0 entries after invalidation")
	}
	if c.Invalidate("a") {
		t.Fatal("expected Invalidate to return false for missing key")
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewCache(2)
	loader := func(p string) (string, error) { return p, nil }

	_, _ = Get(c, "a", loader)
	_, _ = Get(c, "b", loader)
	_, _ = Get(c, "c", loader) // should evict "a"

	if c.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", c.Len())
	}

	// "a" should be gone.
	calls := 0
	_, _ = Get(c, "a", func(p string) (string, error) {
		calls++
		return p, nil
	})
	if calls != 1 {
		t.Fatal("expected cache miss for evicted entry 'a'")
	}
}

func TestLRUEviction_AccessBumps(t *testing.T) {
	c := NewCache(3)
	loader := func(p string) (string, error) { return p, nil }

	_, _ = Get(c, "a", loader) // order: [a]
	_, _ = Get(c, "b", loader) // order: [b, a]
	_, _ = Get(c, "c", loader) // order: [c, b, a]
	_, _ = Get(c, "a", loader) // bump "a" → [a, c, b]; "b" is LRU
	_, _ = Get(c, "d", loader) // evict "b" → [d, a, c]

	// "b" should be evicted.
	calls := 0
	_, _ = Get(c, "b", func(p string) (string, error) {
		calls++
		return p, nil
	})
	if calls != 1 {
		t.Fatal("expected cache miss for evicted entry 'b'")
	}

	// "a" should still be present (was bumped before eviction).
	calls = 0
	_, _ = Get(c, "a", func(p string) (string, error) {
		calls++
		return p, nil
	})
	if calls != 0 {
		t.Fatal("expected cache hit for 'a'")
	}
}

func TestClear(t *testing.T) {
	c := NewCache(0)
	loader := func(p string) (string, error) { return p, nil }
	_, _ = Get(c, "a", loader)
	_, _ = Get(c, "b", loader)
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", c.Len())
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "key"
			_, _ = Get(c, key, func(_ string) (int, error) {
				return n, nil
			})
		}(i)
	}
	wg.Wait()
	if c.Len() != 1 {
		t.Fatalf("expected 1 entry, got %d", c.Len())
	}
}
