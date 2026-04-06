# Package: `whisky`

The `whisky` package is the runtime entrypoint for games. It owns the high-level configuration contract and the execution loop that calls game lifecycle hooks.

## Files

| File | Purpose |
|------|---------|
| `runtime.go` | `Config`, `Context`, `Game`, and `Run()` |
| `internal/backend/desktop_*.go` | OS-selected desktop backend factories used by the runtime |
| `internal/backend/vulkan.go` | native desktop backend that boots native windows plus the Vulkan instance/surface/device/swapchain stack |
| `internal/nativewindow/desktop_*.go` | OS-selected native window factories for future Vulkan/D3D12/Metal integration |
| `internal/gfx/rhi/*.go` | graphics abstraction contracts that bind future render backends to native window handles |
| `internal/gfx/vulkan/*.go` | Vulkan backend loader, instance/device/swapchain creation, and native surface creation behind the RHI |
| `internal/platform/platform.go` | backend contracts for native platform and renderer integration |

## Responsibilities

- normalize runtime configuration
- lock the main goroutine to the OS thread
- create a `Context` with Camera2D, Input, and Scene
- bind the runtime to an internal backend contract rather than a concrete native implementation
- create a native window and, once the Vulkan renderer is complete, bind it to the RHI stack
- poll native input state and feed it into the input system each frame
- execute `Load`, `Update`, and `Shutdown` in a predictable order
- advance the active scene each frame
- collect draw commands (`DrawRect`) and present them via the active renderer backend
- provide a simple shutdown mechanism through `Context.Quit()`

## Current Loop

```text
Run()
  -> defaults
  -> Context (with Camera2D at virtual center)
  -> create backend
  -> backend.SetLogicalSize(...)
  -> Load()
  -> repeat until quit or max frames:
       backend.UpdateInput(...)
       Pump native events
       Scene.Update(dt)
       Game.Update(dt)     // game calls ctx.DrawRect() here
       backend.DrawFrame(...)
       Reset draw queue
  -> Shutdown()
```

The SDL3 renderer path has been removed from `whisky.Run`. Desktop now boots through the native window layer plus Vulkan instance, surface, device, and swapchain creation for Win32, X11, and Wayland. The remaining gap is the actual 2D renderer implementation: the backend is alive, but `DrawFrame` does not yet rasterize or present the queued draw commands.

## Virtual Resolution

Virtual resolution remains part of the runtime contract, but the concrete mapping from virtual coordinates to window pixels still needs to move into the Vulkan renderer path.

## Input Pipeline

The native Win32, X11, and Wayland backends mirror the same `Config.KeyMap` contract. Wayland currently uses a minimal evdev-based keyboard map and does not yet include full `xkbcommon` layout handling.

The scancode-to-control mapping is defined by `Config.KeyMap` (`whisky.KeyMap`, a `map[string]string`). Keys are human-readable names (e.g. `"w"`, `"space"`, `"f1"`); values are the control names fed into the input system. If `KeyMap` is nil, a built-in default set is used (w/a/s/d, arrows, space, lshift, enter).

```go
whisky.Run(&game{}, whisky.Config{
    KeyMap: whisky.KeyMap{
        "space": "jump",
        "z":     "attack",
        "lctrl": "crouch",
    },
})
```

Supported key names: letters (`"a"`–`"z"`), digits (`"0"`–`"9"`), arrow keys (`"up"`, `"down"`, `"left"`, `"right"`), named keys (`"space"`, `"enter"`, `"escape"`, `"backspace"`, `"tab"`, `"lshift"`, `"rshift"`, `"lctrl"`, `"rctrl"`, `"lalt"`, `"ralt"`), and function keys (`"f1"`–`"f12"`). Unknown key names are silently ignored.

## Rendering

Games queue filled rectangles via `ctx.DrawRect(worldRect, color)`. The Camera2D automatically transforms world coordinates to screen (virtual) coordinates. The backend already owns a live Vulkan swapchain; the next renderer milestone is to consume that draw queue with command buffers and present it.
