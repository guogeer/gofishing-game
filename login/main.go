package main

import (
	"flag"
	"fmt"

	"github.com/guogeer/quasar/v2/api"
	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/log"
)

var port = flag.Int("port", 9501, "the server port")

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Infof("start login server, listen %s", addr)
	cmd.RegisterService(&cmd.ServiceConfig{
		Name: "login", Addr: addr,
	})
	log.Debugf("register service login addr %s", addr)
	api.Run(addr)
}
