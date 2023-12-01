//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"net/http"

	_ "gofishing-game/login"

	"github.com/guogeer/quasar/cmd"
	"github.com/guogeer/quasar/log"
)

var port = flag.Int("port", 9501, "the server port")

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Infof("start login server, listen %s", addr)
	cmd.RegisterService(&cmd.ServiceConfig{
		Name: "login", Addr: addr,
	})
	http.ListenAndServe(addr, nil)
}
