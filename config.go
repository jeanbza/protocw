package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type dep struct {
	repo string
}

type config struct {
	deps []dep
}

func loadConfig(configFile string) (*config, error) {
	b, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var c config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
