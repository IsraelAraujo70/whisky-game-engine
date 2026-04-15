// Package assets provides a thread-safe, LRU-evictable cache for game assets
// and an optional file-system watcher for hot-reload during development.
package assets

import (
	"container/list"
	"sync"
)

// entry is one cached value together with its LRU tracking element.
type entry struct {
	path    string
	value   any
	element *list.Element // pointer into Cache.order
}

// Cache is a thread-safe asset cache keyed by path strings that are relative to
// the project's assets_root. An optional maximum size enables LRU eviction.
//
// Zero value is NOT usable; create one with NewCache.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*entry
	order   *list.List // front = most recently used
	maxSize int        // 0 means unlimited

	// AssetsRoot is the absolute path of the watched asset directory.
	// It is used to compute relative cache keys from absolute file paths.
	AssetsRoot string
}

// NewCache creates a new Cache. maxSize <= 0 means unlimited entries.
func NewCache(maxSize int) *Cache {
	if maxSize < 0 {
		maxSize = 0
	}
	return &Cache{
		items:   make(map[string]*entry),
		order:   list.New(),
		maxSize: maxSize,
	}
}

// get returns the cached value for path and promotes it to the front of the LRU
// list. Caller must hold the write lock.
func (c *Cache) get(path string) (any, bool) {
	e, ok := c.items[path]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(e.element)
	return e.value, true
}

// set stores value under path, evicting the LRU entry if the cache is full.
func (c *Cache) set(path string, value any) {
	if e, ok := c.items[path]; ok {
		e.value = value
		c.order.MoveToFront(e.element)
		return
	}

	// Evict oldest if at capacity.
	if c.maxSize > 0 && c.order.Len() >= c.maxSize {
		c.evictOldest()
	}

	e := &entry{path: path, value: value}
	e.element = c.order.PushFront(path)
	c.items[path] = e
}

// evictOldest removes the least-recently-used entry.
func (c *Cache) evictOldest() {
	back := c.order.Back()
	if back == nil {
		return
	}
	p := back.Value.(string)
	c.order.Remove(back)
	delete(c.items, p)
}

// Invalidate removes the entry for path, returning true if it existed.
func (c *Cache) Invalidate(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[path]
	if !ok {
		return false
	}
	c.order.Remove(e.element)
	delete(c.items, e.path)
	return true
}

// Len returns the number of cached entries.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Clear removes all entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*entry)
	c.order.Init()
}
