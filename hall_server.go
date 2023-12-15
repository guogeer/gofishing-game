//go:build ignore
// +build ignore

package main

import (
	// "fmt"
	_ "gofishing-game/hall"
	"gofishing-game/service"
)

func main() {
	service.Start()
}
