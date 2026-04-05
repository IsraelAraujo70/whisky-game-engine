# Runtime and Main Loop

## Requirements

- single high-level engine entrypoint
- deterministic lifecycle order
- graceful shutdown
- config defaults that make window bootstrap simple
- virtual resolution scaling with letterbox / pixel-perfect modes
- keyboard input fed into the action-based input system

## Current shape

The runtime creates an SDL3 window with virtual resolution scaling via `SetLogicalPresentation`. It polls keyboard state, runs the game loop, collects draw commands, and presents 2D rectangles plus a debug text overlay.

### Lifecycle

```text
Config -> Context (+ Camera2D) -> SDL3 init + SetLogicalPresentation -> Game.Load()
  -> loop:
       UpdateInput (keyboard -> input.State)
       Poll native events
       Scene.Update(dt)
       Game.Update(dt)       // game calls ctx.DrawRect() here
       DrawFrame (clear + rects + debug overlay)
       Reset draw queue
  -> Game.Shutdown()
```

### Threading

`whisky.Run()` locks the goroutine to the current OS thread. That keeps the contract aligned with future graphics APIs that require main-thread ownership.

### Timing

The runtime uses a target FPS and a ticker-backed frame cadence. It is good enough for the bootstrap slice and easy to replace with a more precise frame scheduler later.

### Virtual resolution

SDL3's `SetLogicalPresentation` maps virtual coordinates (default 320x180) to window pixels (default 1280x720). When `Config.PixelPerfect` is true, `LOGICAL_PRESENTATION_INTEGER_SCALE` is used; otherwise `LOGICAL_PRESENTATION_LETTERBOX`. All rendering (rectangles and debug text) operates in the virtual coordinate space.

### 2D rendering

Games queue colored rectangles via `ctx.DrawRect(worldRect, color)`. The `Camera2D` on `Context` transforms world coordinates to virtual screen coordinates automatically. The SDL3 platform draws the queued rectangles via `RenderFillRect`, then renders the debug text overlay on top of everything, and finally calls `Present`.

### Input pipeline

Every frame the SDL3 platform reads `sdl.GetKeyboardState()` and maps scancodes to control names using the `Config.KeyMap` provided by the game. Games bind actions to controls via `ctx.Input.Bind()` and query them with `Pressed`, `JustPressed`, or `Axis`.

`Config.KeyMap` is a `whisky.KeyMap` (`map[string]string`) where keys are human-readable key names and values are control names. If nil, a built-in default set is used. Unknown key names are silently ignored. The full mapping is resolved once at startup via `buildKeyBindings` in the SDL3 platform layer.
