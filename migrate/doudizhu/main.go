package main

import (
	"flag"
	_ "gofishing-game/migrate/doudizhu/internal"
	"gofishing-game/service"
)

func main() {
	flag.Parse()

	service.Start()
}
