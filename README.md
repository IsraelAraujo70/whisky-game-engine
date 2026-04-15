# whisky game engine

Whisky is a 2D game engine for Go, built for desktop-first games and structured to grow into a reusable open source engine.

## Current status

This repository is bootstrapped with:

- a native-window + Vulkan-first runtime architecture in [`whisky`](./whisky)
- foundational packages for [`geom`](./geom), [`scene`](./scene), [`input`](./input), and [`physics`](./physics)
- reusable gameplay primitives in [`gameplay`](./gameplay) for health, damage, and basic patrol AI
- a CLI in [`cmd/whisky`](./cmd/whisky) with `new`, `run`, and `doctor`
- an internal project template system
- a sample game in [`examples/pixel-quest`](./examples/pixel-quest)
- internal product and package documentation in the same structure used by `../crit-ide`

## Quick start

Run the engine checks:

```bash
go run ./cmd/whisky doctor
```

Create a new game project:

```bash
go run ./cmd/whisky new my-game
```

Run a Whisky project from its root:

```bash
go run ./cmd/whisky run
```

Run the bundled example:

```bash
go run ./examples/pixel-quest/cmd/game
```

## Near-term direction

- finish Vulkan device, swapchain, and 2D renderer integration
- add a Metal-backed renderer path for macOS after the RHI stabilizes
- load real assets from `whisky.json`
- evolve the sample into a small playable pixel-art game

See [`docs/prd.md`](./docs/prd.md), [`docs/progress.json`](./docs/progress.json), and the files in [`docs/spec`](./docs/spec) for the current internal planning baseline.
