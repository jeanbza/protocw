package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Include struct {
	LocalPath string   `yaml:"localpath"`
	Repo      string   `yaml:"repo"`
	Protos    []string `yaml:"protos"`
}

func loadConfig(configFile string) ([]Include, error) {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var i []Include
	if err := yaml.Unmarshal(b, &i); err != nil {
		return nil, err
	}

	return i, nil
}
