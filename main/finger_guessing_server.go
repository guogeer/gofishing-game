//go:build ignore
// +build ignore

package main

import (
	"flag"
	"gofishing-game/service"
	"net/http"

	"github.com/guogeer/quasar/log"
)

var serverName = flag.String("name", "fingerguessing", "server name, default demo")

func main() {
	flag.Parse()
	log.Infof("start bingo game: %s", *serverName)
	go func() { http.ListenAndServe(":8087", nil) }()

	// fingerguessing.InitWorld(*serverName)
	service.Start()
}
