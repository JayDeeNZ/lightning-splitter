package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Host         string `yaml:"host"`
	Port         int64  `yaml:"port"`
	TLSCertPath  string `yaml:"tlsCertPath"`
	MacaroonPath string `yaml:"macaroonPath"`
}

var (
	Config Configuration
)

func LoadConfig(configPath string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	var c *Configuration
	if err = yaml.Unmarshal(data, &c); err != nil {
		log.Fatal(err)
	}

	Config = *c
}
