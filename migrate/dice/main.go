package main

import (
	"flag"
	_ "gofishing-game/migrate/dice/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
