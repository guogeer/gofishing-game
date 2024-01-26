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
	Version     int    `yaml:"version"`
	Environment string `yaml:"environment"`
	DB          struct {
		Game   DataSource `yaml:"game"`
		Manage DataSource `yaml:"manage"`
	} `yaml:"db"`
	ProductName string `yaml:"productName"`
	ScriptPath  string `yaml:"scriptPath"`
	TablePath   string `yaml:"tablePath"`
}

var defaultConfig Env

func init() {
	config.LoadFile(config.Config().Path(), &defaultConfig)
}

func Config() *Env {
	return &defaultConfig
}
