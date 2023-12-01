//go:build ignore
// +build ignore

package main

import (
	// "fmt"
	"gofishing-game/service"
	_ "gofishing-game/service/hall"
)

func main() {
	service.Start()
}
