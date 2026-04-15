package assets

import (
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// eventually retries fn every interval up to timeout. Returns false if fn never
// returned true within the deadline.
func eventually(timeout, interval time.Duration, fn func() bool) bool {
	deadline := time.After(timeout)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if fn() {
			return true
		}
		select {
		case <-deadline:
			return fn()
		case <-tick.C:
		}
	}
}

func TestWatcher_InvalidateOnWrite(t *testing.T) {
	dir := t.TempDir()

	// Create a file before starting the watcher.
	fpath := filepath.Join(dir, "sprite.png")
	if err := os.WriteFile(fpath, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewCache(0)
	// Pre-populate cache with a relative path.
	cache.mu.Lock()
	cache.set("sprite.png", "original")
	cache.mu.Unlock()

	logger := log.New(os.Stderr, "[test] ", 0)

	var reloaded atomic.Int32
	w, err := NewWatcher(dir, cache, logger)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	w.SetOnReload(func(relPath string) {
		reloaded.Add(1)
	})
	defer func() {
		if cErr := w.Close(); cErr != nil {
			t.Logf("watcher close: %v", cErr)
		}
	}()

	// Give the watcher a moment to settle.
	time.Sleep(50 * time.Millisecond)

	// Modify the file.
	if err := os.WriteFile(fpath, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Assert that the callback fired and the cache was invalidated.
	if !eventually(2*time.Second, 20*time.Millisecond, func() bool {
		return reloaded.Load() > 0
	}) {
		t.Fatal("expected OnReload callback to fire after file write")
	}

	if cache.Len() != 0 {
		t.Fatalf("expected cache to be empty after invalidation, got %d entries", cache.Len())
	}
}

func TestWatcher_NewSubdirectory(t *testing.T) {
	dir := t.TempDir()
	cache := NewCache(0)
	logger := log.New(os.Stderr, "[test] ", 0)

	var reloaded atomic.Int32
	w, err := NewWatcher(dir, cache, logger)
	if err != nil {
		t.Fatal(err)
	}
	if w == nil {
		t.Fatal("expected non-nil watcher")
	}
	w.SetOnReload(func(relPath string) {
		reloaded.Add(1)
	})
	defer func() { _ = w.Close() }()

	time.Sleep(50 * time.Millisecond)

	// Create a subdirectory and write a file into it.
	sub := filepath.Join(dir, "sprites")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// Give watcher time to pick up the new directory.
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(filepath.Join(sub, "hero.png"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !eventually(2*time.Second, 20*time.Millisecond, func() bool {
		return reloaded.Load() > 0
	}) {
		t.Fatal("expected OnReload callback for file in new subdirectory")
	}
}

func TestWatcher_MissingDirReturnsNil(t *testing.T) {
	cache := NewCache(0)
	logger := log.New(os.Stderr, "[test] ", 0)
	w, err := NewWatcher("/nonexistent/path/that/does/not/exist", cache, logger)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if w != nil {
		_ = w.Close()
		t.Fatal("expected nil watcher for missing directory")
	}
}

func TestWatcher_CloseNil(t *testing.T) {
	var w *Watcher
	if err := w.Close(); err != nil {
		t.Fatalf("Close on nil watcher should not error, got %v", err)
	}
}
