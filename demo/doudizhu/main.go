package main

import (
	"flag"
	_ "gofishing-game/demo/doudizhu/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
