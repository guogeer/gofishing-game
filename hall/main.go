package main

import (
	// "fmt"
	"gofishing-game/service"
	_ "gofishing-game/service/system"
)

func main() {
	service.Start()
}
