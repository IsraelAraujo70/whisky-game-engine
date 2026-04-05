# Package: `whisky`

The `whisky` package is the runtime entrypoint for games. It owns the high-level configuration contract and the execution loop that calls game lifecycle hooks.

## Files

| File | Purpose |
|------|---------|
| `runtime.go` | `Config`, `Context`, `Game`, and `Run()` |
| `internal/platform/sdl3/runtime.go` | SDL3 window bootstrap, event pump, and debug-text frame presentation |

## Responsibilities

- normalize runtime configuration
- lock the main goroutine to the OS thread
- create a `Context`
- create a native SDL3 window by default
- execute `Load`, `Update`, and `Shutdown` in a predictable order
- advance the active scene each frame
- provide a simple shutdown mechanism through `Context.Quit()`

## Current Loop

```text
Run()
  -> defaults
  -> Context
  -> SDL3 window + renderer
  -> Load()
  -> repeat until quit or max frames:
       Pump native events
       Scene.Update(dt)
       Game.Update(dt)
       Draw debug overlay
  -> Shutdown()
```

This is intentionally small. The package now opens a real window through SDL3, but the drawing path is still only a bootstrap overlay until the dedicated renderer lands.
