# pixel-quest

`pixel-quest` is the first bundled Whisky sample. It is still a bootstrap sample, not the final gameplay milestone.

Run it with:

```bash
go run ./examples/pixel-quest/cmd/game
```

## Hot-reload

While the game is running, edit any `.png` or `.jpg` file under
`examples/pixel-quest/assets/` and save. The engine detects the change
automatically and reuploads the texture to the GPU -- the sprite updates on
screen without restarting the game.

To test manually:

```bash
# In one terminal, start the game:
go run ./examples/pixel-quest/cmd/game

# In another terminal, replace the player sprite:
cp examples/pixel-quest/assets/player.png /tmp/player-backup.png
convert -flop examples/pixel-quest/assets/player.png examples/pixel-quest/assets/player.png
# (or simply open the PNG in GIMP, edit, and save)
```

Hot-reload is enabled by default. To disable it, set `HotReload` to `false` in
the game config (or remove `AssetsRoot`).
