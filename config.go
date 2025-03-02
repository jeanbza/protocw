package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Include struct {
	Repo   string   `yaml:"repo"`
	Protos []string `yaml:"protos"`
}

type Config struct {
	Includes []Include `yaml:"includes"`
}

func loadConfig(configFile string) (*Config, error) {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
