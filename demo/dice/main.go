package main

import (
	"flag"
	_ "gofishing-game/demo/dice/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
