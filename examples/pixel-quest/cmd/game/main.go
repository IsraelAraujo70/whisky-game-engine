package main

import (
	"log"

	"github.com/IsraelAraujo70/whisky-game-engine/examples/pixel-quest/game"
)

func main() {
	if err := game.Run(); err != nil {
		log.Fatal(err)
	}
}
