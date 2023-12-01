package env

import (
	"github.com/guogeer/quasar/config"
)

type DataSource struct {
	User         string `xml:"User"`
	Password     string `xml:"Password"`
	Addr         string `xml:"Address"`
	Name         string `xml:"Name"`
	MaxIdleConns int
	MaxOpenConns int
}

type Env struct {
	Version          int
	Environment      string
	DataSource       DataSource
	ManageDataSource DataSource
	SlaveDataSource  DataSource
	ProductName      string
	Sign             string

	config.Env
}

var defaultConfig Env

func init() {
	config.LoadFile(config.Config().Path(), &defaultConfig)
}

func Config() *Env {
	return &defaultConfig
}
