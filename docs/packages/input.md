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

This package is intentionally backend-agnostic. Native platform backends feed it every frame by translating OS keyboard state into semantic control names and calling `SetPressed`.

## Available control names

Control names are defined by `Config.KeyMap` in the `whisky` package. Games map key names to control names of their choice. The default `KeyMap` (used when nil) provides:

| Key name | Default control |
|----------|----------------|
| `w` | `w` |
| `a` | `a` |
| `s` | `s` |
| `d` | `d` |
| `up` | `up` |
| `down` | `down` |
| `left` | `left` |
| `right` | `right` |
| `space` | `space` |
| `lshift` | `lshift` |
| `enter` | `enter` |

Games can override this entirely or extend it with custom controls:

```go
whisky.Config{
    KeyMap: whisky.KeyMap{
        "space": "jump",
        "z":     "attack",
        "up":    "jump", // remap arrow up to jump too
    },
}
```

Supported key names: letters (`"a"`–`"z"`), digits (`"0"`–`"9"`), arrow keys, named keys (`"space"`, `"enter"`, `"escape"`, `"backspace"`, `"tab"`, `"lshift"`, `"rshift"`, `"lctrl"`, `"rctrl"`, `"lalt"`, `"ralt"`), and function keys (`"f1"`–`"f12"`).

## Example usage

```go
// In Load:
ctx.Input.Bind("move_left", "a", "left")
ctx.Input.Bind("move_right", "d", "right")

// In Update:
dx := ctx.Input.Axis("move_left", "move_right") // returns -1, 0, or +1
```
