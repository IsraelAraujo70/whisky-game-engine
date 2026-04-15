package assets

// Get retrieves a cached asset of type T at the given path. On cache miss, the
// load function is called to produce the value, which is then stored in the
// cache for future hits.
//
// The generic signature avoids type assertions at the call site while keeping
// the cache itself type-erased (any).
func Get[T any](cache *Cache, path string, load func(string) (T, error)) (T, error) {
	cache.mu.Lock()

	// Cache hit: bump LRU and return.
	if v, ok := cache.get(path); ok {
		cache.mu.Unlock()
		return v.(T), nil
	}

	// Cache miss: release the lock while calling the (potentially slow) loader
	// so other goroutines are not blocked.
	cache.mu.Unlock()

	val, err := load(path)
	if err != nil {
		var zero T
		return zero, err
	}

	cache.mu.Lock()
	// Another goroutine may have loaded the same path while we were loading.
	// If so, prefer the already-cached value for consistency.
	if v, ok := cache.get(path); ok {
		cache.mu.Unlock()
		return v.(T), nil
	}
	cache.set(path, val)
	cache.mu.Unlock()
	return val, nil
}
