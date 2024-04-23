package main

import (
	"flag"
	_ "gofishing-game/migrate/entertainment/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
