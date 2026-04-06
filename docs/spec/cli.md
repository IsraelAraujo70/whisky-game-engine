# CLI

## Commands

### `whisky new <name>`

Creates a standalone starter project with:

- `go.mod`
- `whisky.json`
- `cmd/game/main.go`
- `game/game.go`
- `README.md`

When the CLI is run from the Whisky engine repository, the generated project receives a local `replace` directive back to the engine checkout.

### `whisky run`

Loads `whisky.json` from the current directory and executes:

```bash
go run <entry_point>
```

### `whisky doctor`

Reports availability of:

- `go`
- `git`
- `pkg-config`
- `vulkan`
- `x11`
- `wayland-client`

This command is intentionally pragmatic and shell-based.
