package env

import (
	"github.com/guogeer/quasar/config"
)

type DataSource struct {
	User         string `xml:"User" yaml:"user"`
	Password     string `xml:"Password" yaml:"password"`
	Addr         string `xml:"Address" yaml:"address"`
	Name         string `xml:"Name" yaml:"name"`
	MaxIdleConns int    `yaml:"maxIdleConns"`
	MaxOpenConns int    `yaml:"maxOpenConns"`
}

type Env struct {
	Version          int        `yaml:"version"`
	Environment      string     `yaml:"environment"`
	DataSource       DataSource `yaml:"dataSource"`
	ManageDataSource DataSource `yaml:"manageDataSource"`
	SlaveDataSource  DataSource `yaml:"slaveDataSource"`
	ProductName      string     `yaml:"productName"`
	Sign             string     `yaml:"sign"`
}

var defaultConfig Env

func init() {
	config.LoadFile(config.Config().Path(), &defaultConfig)
}

func Config() *Env {
	return &defaultConfig
}
