# Package: `whisky`

The `whisky` package is the runtime entrypoint for games. It owns the high-level configuration contract and the execution loop that calls game lifecycle hooks.

## Files

| File | Purpose |
|------|---------|
| `runtime.go` | `Config`, `Context`, `Game`, and `Run()` |
| `internal/platform/sdl3/runtime.go` | SDL3 window bootstrap, event pump, input polling, and 2D rendering |

## Responsibilities

- normalize runtime configuration
- lock the main goroutine to the OS thread
- create a `Context` with Camera2D, Input, and Scene
- create a native SDL3 window with virtual resolution scaling
- poll keyboard state and feed it into the input system each frame
- execute `Load`, `Update`, and `Shutdown` in a predictable order
- advance the active scene each frame
- collect draw commands (`DrawRect`) and present them via SDL3
- provide a simple shutdown mechanism through `Context.Quit()`

## Current Loop

```text
Run()
  -> defaults
  -> Context (with Camera2D at virtual center)
  -> SDL3 window + renderer + SetLogicalPresentation
  -> Load()
  -> repeat until quit or max frames:
       UpdateInput (keyboard state -> input.State)
       Pump native events
       Scene.Update(dt)
       Game.Update(dt)     // game calls ctx.DrawRect() here
       DrawFrame (clear + filled rects + debug text)
       Reset draw queue
  -> Shutdown()
```

## Virtual Resolution

SDL3's `SetLogicalPresentation` handles the mapping from virtual coordinates (e.g. 320x180) to window pixels (e.g. 1280x720). When `Config.PixelPerfect` is true, integer scaling is used; otherwise letterboxing is applied. All draw calls and debug text operate in virtual coordinate space.

## Input Pipeline

The SDL3 platform reads the keyboard state every frame via `sdl.GetKeyboardState()` and maps scancodes to engine control names (e.g. `"w"`, `"a"`, `"space"`). Games bind actions to these control names via `ctx.Input.Bind("move_left", "a", "left")`.

> **Remark:** The scancode-to-control mapping is currently a hardcoded table inside `internal/platform/sdl3/runtime.go`. This must be made configurable so that games can define their own key mappings — for example via a `KeyMap` field on `Config` or a `RegisterKey` API.

## Rendering

Games queue filled rectangles via `ctx.DrawRect(worldRect, color)`. The Camera2D automatically transforms world coordinates to screen (virtual) coordinates. The SDL3 platform draws rectangles via `RenderFillRect` and renders the debug text overlay on top.
