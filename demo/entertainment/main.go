package main

import (
	"flag"
	_ "gofishing-game/demo/entertainment/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
