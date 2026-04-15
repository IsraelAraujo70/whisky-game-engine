# Assets and Hot Reload

## Current state

The `assets` package provides a thread-safe, LRU-evictable cache for game
assets and an optional file-system watcher (`github.com/fsnotify/fsnotify`) for
hot-reload during development.

### Asset cache (`assets/cache.go`, `assets/loader.go`)

- `NewCache(maxSize)` creates a cache (0 = unlimited, default 256 in whisky.Config).
- `Get[T](cache, path, loadFn)` is the generic loader: cache hit returns
  immediately; cache miss calls `loadFn`, stores the result, and returns it.
- LRU eviction kicks in when maxSize is exceeded.
- All operations are goroutine-safe (sync.Mutex internally).

### File watcher (`assets/watcher.go`)

- `NewWatcher(assetsRoot, cache, logger)` recursively watches a directory tree.
- On Write/Create/Remove/Rename events the matching cache entry is invalidated
  and an optional `OnReload` callback is invoked with the relative path.
- New subdirectories are automatically added to the watch list.
- If `assetsRoot` does not exist, a warning is logged and hot-reload is
  silently disabled (no fatal error).

### Integration with whisky runtime

- `whisky.Config.HotReload` (default true) and `whisky.Config.AssetsRoot`
  control the feature.
- `whisky.Run` initializes cache and watcher; shutdown is deferred.
- `ctx.Assets()` exposes the cache to game code.
- `ctx.LoadTexture(path)` goes through the cache on each call; repeated
  loads of the same path are free.
- When a `.png` or `.jpg` file changes on disk, the watcher decodes the new
  image and calls `platform.ReuploadTexture` which destroys the old SDL
  texture and creates a new one under the same TextureID. Sprites that
  reference that handle update automatically on the next frame.

## Reasoning

It is better to lock the project shape first and attach the asset layer after the native runtime and renderer exist.
