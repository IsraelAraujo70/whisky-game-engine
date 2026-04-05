# Package: `input`

The `input` package models actions instead of hardware events. Games bind one or more controls to named actions, then query state through a shared input object.

## Files

| File | Purpose |
|------|---------|
| `state.go` | action bindings, button state, and digital axis helpers |

## Current capabilities

- bind multiple controls to a single action
- mark controls as pressed or released
- query `Pressed(action)`
- query `JustPressed(action)`
- derive a digital axis with `Axis(negativeAction, positiveAction)`

This package is intentionally backend-agnostic. The SDL3 platform feeds it every frame by reading `sdl.GetKeyboardState()` and calling `SetPressed` for each mapped scancode.

## Available control names

The SDL3 platform currently maps these scancodes to control names:

| Control | Key |
|---------|-----|
| `w` | W |
| `a` | A |
| `s` | S |
| `d` | D |
| `up` | Arrow Up |
| `down` | Arrow Down |
| `left` | Arrow Left |
| `right` | Arrow Right |
| `space` | Space |
| `lshift` | Left Shift |
| `enter` | Enter/Return |

> **Remark:** This mapping is hardcoded in `internal/platform/sdl3/runtime.go` and must be made configurable in a future iteration so games can register custom scancode-to-control mappings.

## Example usage

```go
// In Load:
ctx.Input.Bind("move_left", "a", "left")
ctx.Input.Bind("move_right", "d", "right")

// In Update:
dx := ctx.Input.Axis("move_left", "move_right") // returns -1, 0, or +1
```

