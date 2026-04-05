# Assets and Hot Reload

## Current state

The initial bootstrap does not load real assets yet. The repository only establishes the project file and the sample structure that future asset systems will consume.

## Planned direction

- `whisky.json` remains the root metadata file
- the generated project reserves an `assets/` directory
- hot reload will first target file-backed content such as textures, tilemaps, and audio
- code reload is explicitly out of scope for the early engine slices

## Reasoning

It is better to lock the project shape first and attach the asset layer after the native runtime and renderer exist.

