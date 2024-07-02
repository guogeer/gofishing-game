package main

import (
	"flag"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/guogeer/quasar/v2/api"
	"github.com/guogeer/quasar/v2/cmd"
	"github.com/guogeer/quasar/v2/log"
)

var port = flag.Int("port", 9501, "the server port")
var rootpath = flag.String("rootpath", "", "root path")

func main() {
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)
	log.Infof("start login server, listen %s", addr)
	cmd.RegisterService(&cmd.ServiceConfig{
		Name: "login", Addr: addr,
	})
	log.Debugf("register service login addr %s", addr)

	r := gin.Default()
	if *rootpath != "" {
		r.StaticFile("/", *rootpath)
	}
	api.RunWithEngine(r, addr)
}
