# Package: `input`

The `input` package currently models actions instead of hardware events. Games bind one or more controls to named actions, then query state through a shared input object.

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

This package is intentionally backend-agnostic so it can later be fed by SDL3 events without changing game-facing APIs.

