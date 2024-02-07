package main

import (
	"flag"
	_ "gofishing-game/fingerguessing/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
