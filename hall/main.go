package main

import (
	// "fmt"
	_ "gofishing-game/hall/internal"
	"gofishing-game/service"
	_ "gofishing-game/service/system"
)

func main() {
	service.Start()
}
