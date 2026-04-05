# whisky game engine — Product Requirements Document

## Vision

Whisky is a desktop-first 2D game engine written in Go. It targets developers who want a code-first workflow, a clean runtime architecture, and a path from personal experimentation to reusable open source tooling.

The first implementation prioritizes a strong foundation over flashy surface area. The engine starts with a minimal runtime loop, scene graph, input actions, collision primitives, and project tooling, while keeping the architecture aligned with the future SDL3 + OpenGL runtime.

## Core Principles

1. **Code-first** — Games are written in Go against engine APIs. No editor is required for V1.
2. **Desktop-first** — Linux, macOS, and Windows are the initial targets.
3. **Pixel-art aware** — Virtual resolution and predictable scaling are first-class defaults.
4. **Stable conceptual core** — A small set of public packages should carry most of the mental model.
5. **Pragmatic evolution** — Start with a headless bootstrap that compiles and tests cleanly, then add native runtime layers without resetting the API.
6. **Open source ready** — Repository shape, docs, and package boundaries should make sense to external contributors from day one.

## Target Users

### Primary

Developers who:
- Write Go comfortably
- Want to build small-to-medium 2D games
- Prefer code APIs over heavy visual editors
- Care about a clean internal architecture

### Secondary

- Hobbyist engine builders
- Open source contributors interested in runtime architecture
- Developers prototyping gameplay systems in Go

## Product Scope — Initial Foundation

### In Scope

| Area | Features |
|------|----------|
| **Runtime** | `Game`, `Config`, `Context`, fixed-target loop, graceful shutdown |
| **Scene** | Node tree, local/world position, component lifecycle |
| **Geometry** | `Vec2`, `Rect`, `Color` |
| **Input** | Action map, pressed/just-pressed, digital axis |
| **Physics** | AABB overlap, layers/masks, point/rect queries |
| **Tooling** | `whisky new`, `whisky run`, `whisky doctor` |
| **Project format** | `whisky.json` parsing and validation |
| **Templates** | Embedded starter project template |
| **Examples** | `pixel-quest` sample using only public APIs |
| **Docs** | Product, spec, package, and progress docs |

### Out of Scope (Current Slice)

- SDL3 window creation
- OpenGL rendering
- Sprite batching
- Real asset loading pipeline
- Audio playback
- Tilemaps
- Editor tooling
- Advanced 2D physics

### Next Slice

- SDL3 platform layer
- GL33 renderer shell
- Asset handles and cache
- Camera and sprite components
- Real sample gameplay content

## Architecture Overview

### Layered Model

```text
Layer 4 — Games
  example games, generated projects, gameplay code

Layer 3 — Public Engine Packages
  whisky, scene, input, physics, geom

Layer 2 — Internal Tooling
  internal/cli, internal/project, internal/template

Layer 1 — Platform / Native Backends
  planned: SDL3, OpenGL 3.3, asset watchers, audio
```

### Runtime Flow

```text
Run(game, cfg)
  -> normalize config
  -> create Context
  -> call game.Load()
  -> loop:
       scene.Update(dt)
       game.Update(dt)
       stop when Context.Quit() or MaxFrames reached
  -> call game.Shutdown()
```

### Tooling Flow

```text
whisky new <name>
  -> create standalone project
  -> write go.mod, whisky.json, cmd/game, game package
  -> wire local replace directive when running from engine source

whisky run
  -> load whisky.json
  -> execute "go run <entry_point>" in the current project

whisky doctor
  -> inspect Go, git, pkg-config, SDL3 and OpenGL availability
```

## Success Criteria

- The repository builds and tests on a clean Go install.
- `whisky doctor` reports meaningful status for local prerequisites.
- `whisky new` generates a runnable project.
- The generated project can run through `whisky run`.
- The example project demonstrates the expected public API shape.

## Development Phases

| Phase | Focus | Key Deliverables |
|-------|-------|-----------------|
| **Sprint 1** | Foundation | runtime loop, config, CLI, templates, docs, tests |
| **Sprint 2** | Native runtime | SDL3 init, thread model, platform input shell |
| **Sprint 3** | Rendering | GL33 context, sprite batch, camera, clear/present |
| **Sprint 4** | Content | assets, tilemap ingestion, animation, sample game |
| **Sprint 5** | Gameplay polish | audio, collision workflow, editor-facing metadata |

