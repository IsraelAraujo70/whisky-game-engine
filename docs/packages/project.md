# Package: `internal/project`

The project package handles Whisky project metadata and scaffolding. It is the backbone for the CLI commands that create and run projects.

## Files

| File | Purpose |
|------|---------|
| `config.go` | load and validate `whisky.json` |
| `scaffold.go` | generate starter projects from embedded templates |

## Current responsibilities

- parse project metadata from `whisky.json`
- validate essential defaults such as entrypoint and target FPS
- locate a local engine checkout for `replace` directives
- write a standalone starter project structure

This package is intentionally internal because the user-facing contract is the CLI and the `whisky.json` file, not the scaffolding API itself.

