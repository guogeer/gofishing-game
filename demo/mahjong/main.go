package main

import (
	"flag"
	_ "gofishing-game/demo/mahjong/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
