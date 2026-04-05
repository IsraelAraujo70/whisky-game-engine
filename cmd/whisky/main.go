package main

import (
	"os"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
