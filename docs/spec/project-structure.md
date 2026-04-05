# Project Structure

## Repository layout

- `cmd/whisky`: command-line entrypoint
- `whisky`: public runtime bootstrap
- `geom`: math and color primitives
- `scene`: node hierarchy and components
- `input`: action-oriented input state
- `physics`: collision queries
- `internal/cli`: command dispatch and environment checks
- `internal/project`: project file and scaffolding
- `internal/template`: embedded template files
- `examples/pixel-quest`: sample project
- `docs/spec`: implementation-oriented docs
- `docs/packages`: package-oriented docs

## Why this shape

This keeps the public surface small and the tooling internals isolated, while matching the internal docs pattern already used in `crit-ide`.

