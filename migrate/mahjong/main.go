package main

import (
	"flag"
	_ "gofishing-game/migrate/mahjong/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
