# Runtime and Main Loop

## Requirements

- single high-level engine entrypoint
- deterministic lifecycle order
- graceful shutdown
- config defaults that make window bootstrap simple
- future compatibility with a native platform backend

## Current shape

The runtime now creates a basic SDL3 window and renderer. It still exists mainly to lock the API and lifecycle before the GL33 renderer arrives.

### Lifecycle

```text
Config -> Context -> SDL3 init -> Game.Load() -> loop(Poll + Update + Present) -> Game.Shutdown()
```

### Threading

`whisky.Run()` locks the goroutine to the current OS thread. That keeps the contract aligned with future graphics APIs that require main-thread ownership.

### Timing

The runtime uses a target FPS and a ticker-backed frame cadence. It is good enough for the bootstrap slice and easy to replace with a more precise frame scheduler later.

### Bootstrap rendering

The current frame presentation uses SDL's built-in renderer and debug text:

- clear the window with `Config.ClearColor`
- draw a small overlay with title, frame count, and optional debug text
- close on `Esc` or native window close

This is not the long-term rendering architecture. It is only enough to make the engine visibly alive while the renderer stack is still being built.
