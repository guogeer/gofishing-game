package main

import (
	"flag"
	_ "gofishing-game/demo/fingerguessing/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
