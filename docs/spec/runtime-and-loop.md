# Runtime and Main Loop

## Requirements

- single high-level engine entrypoint
- deterministic lifecycle order
- graceful shutdown
- config defaults that make window bootstrap simple
- virtual resolution scaling with letterbox / pixel-perfect modes
- keyboard input fed into the action-based input system

## Current shape

The runtime no longer falls back to SDL3. Its desktop path is now explicitly native-window + Vulkan-first: native Win32/X11/Wayland windows exist, Vulkan instance/surface creation exists, and the remaining gap is device, swapchain, and frame presentation.

### Lifecycle

```text
Config -> Context (+ Camera2D) -> native window + Vulkan backend bootstrap -> Game.Load()
  -> loop:
       UpdateInput (keyboard -> input.State)
       Poll native events
       Scene.Update(dt)
       Game.Update(dt)       // game calls ctx.DrawRect() here
       DrawFrame (future Vulkan clear + rects + debug overlay)
       Reset draw queue
  -> Game.Shutdown()
```

### Threading

`whisky.Run()` locks the goroutine to the current OS thread. That keeps the contract aligned with future graphics APIs that require main-thread ownership.

### Timing

The runtime uses a target FPS and a ticker-backed frame cadence. It is good enough for the bootstrap slice and easy to replace with a more precise frame scheduler later.

### Virtual resolution

Virtual coordinates remain part of the runtime contract. The concrete pixel-perfect or letterboxed mapping will be implemented in the Vulkan renderer path instead of SDL's logical-presentation API.

### 2D rendering

Games queue colored rectangles via `ctx.DrawRect(worldRect, color)`. The `Camera2D` on `Context` transforms world coordinates to virtual screen coordinates automatically. The next renderer milestone is to consume that queue from Vulkan and present through a swapchain.

### Input pipeline

Every frame the native platform layer maps OS keyboard state to control names using the `Config.KeyMap` provided by the game. Games bind actions to controls via `ctx.Input.Bind()` and query them with `Pressed`, `JustPressed`, or `Axis`.

`Config.KeyMap` is a `whisky.KeyMap` (`map[string]string`) where keys are human-readable key names and values are control names. If nil, a built-in default set is used. Unknown key names are silently ignored. The full mapping is resolved once at startup in the active native backend.
