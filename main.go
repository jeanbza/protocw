package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("protocw <config file>")
		os.Exit(1)
	}
	ctx := context.Background()
	configFile := os.Args[1]
	if err := run(ctx, configFile); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configFile string) error {
	c, err := loadConfig(configFile)
	if err != nil {
		return err
	}

	for _, d := range c.deps {
		fmt.Println(d)
	}

	return nil
}
