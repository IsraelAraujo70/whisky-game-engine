package assets

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher observes a directory tree for file changes and invalidates
// corresponding entries in the asset Cache. An optional OnReload callback is
// invoked after invalidation with the relative path of the changed file.
type Watcher struct {
	assetsRoot string
	cache      *Cache
	watcher    *fsnotify.Watcher
	done       chan struct{}
	logger     *log.Logger

	mu       sync.Mutex
	onReload func(relPath string)
}

// SetOnReload sets the callback invoked when a watched file changes. It is
// safe to call from any goroutine.
func (w *Watcher) SetOnReload(fn func(relPath string)) {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.onReload = fn
	w.mu.Unlock()
}

// NewWatcher creates a Watcher that recursively watches assetsRoot. If the
// directory does not exist a warning is logged and a nil Watcher is returned
// (hot-reload is silently disabled).
func NewWatcher(assetsRoot string, cache *Cache, logger *log.Logger) (*Watcher, error) {
	absRoot, err := filepath.Abs(assetsRoot)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absRoot)
	if err != nil || !info.IsDir() {
		logger.Printf("[assets] warning: assets_root %q not found, hot-reload disabled", absRoot)
		return nil, nil //nolint:nilerr // intentional: missing dir is not fatal
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Walk and add all directories.
	if err := addDirsRecursive(fsw, absRoot); err != nil {
		_ = fsw.Close()
		return nil, err
	}

	w := &Watcher{
		assetsRoot: absRoot,
		cache:      cache,
		watcher:    fsw,
		done:       make(chan struct{}),
		logger:     logger,
	}

	go w.loop()
	return w, nil
}

// Close stops the watcher goroutine and releases resources.
func (w *Watcher) Close() error {
	if w == nil {
		return nil
	}
	err := w.watcher.Close()
	<-w.done // wait for loop to exit
	return err
}

// loop processes fsnotify events until the watcher is closed.
func (w *Watcher) loop() {
	defer close(w.done)
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Printf("[assets] watcher error: %v", err)
		}
	}
}

// handleEvent processes a single fsnotify event.
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only care about writes, creates, removes, and renames.
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}

	absPath, err := filepath.Abs(event.Name)
	if err != nil {
		return
	}

	// If a new directory was created, add it to the watcher.
	if event.Op&fsnotify.Create != 0 {
		if info, sErr := os.Stat(absPath); sErr == nil && info.IsDir() {
			_ = addDirsRecursive(w.watcher, absPath)
			return
		}
	}

	relPath, err := filepath.Rel(w.assetsRoot, absPath)
	if err != nil {
		return
	}
	// Normalize to forward slashes for consistent cache keys.
	relPath = filepath.ToSlash(relPath)

	// Skip paths outside the root (shouldn't happen, but be safe).
	if strings.HasPrefix(relPath, "..") {
		return
	}

	w.cache.Invalidate(relPath)

	w.mu.Lock()
	fn := w.onReload
	w.mu.Unlock()
	if fn != nil {
		fn(relPath)
	}
}

// addDirsRecursive walks root and adds every directory to the fsnotify watcher.
func addDirsRecursive(fsw *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			return fsw.Add(path)
		}
		return nil
	})
}
