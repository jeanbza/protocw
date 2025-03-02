package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
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
		return fmt.Errorf("error loading config: %v", err)
	}

	dir, err := os.MkdirTemp("", "protocw")
	if err != nil {
		return err
	}

	var b protocBuilder

	for _, d := range c.Includes {
		if err := cloneInto(ctx, dir, d.Repo); err != nil {
			return fmt.Errorf("error cloning %s into %s: %v", d.Repo, dir, err)
		}
		for _, proto := range d.Protos {
			path, err := searchDirForProto(dir, proto)
			if err != nil {
				return fmt.Errorf("error searching for %s in %s: %v", proto, dir, err)
			}
			fmt.Printf("Found proto %s in %s at %s\n", proto, d.Repo, path)
			b.addInclude(proto, path)
		}
	}

	cmd := b.build(ctx, "protogen")
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s:\n%s%s: %v", cmd.Args, o.String(), e.String(), err)
	}

	return nil
}

type protocBuilder struct {
	includes map[string]string
}

func (p *protocBuilder) addInclude(proto, pathToProto string) {
	if p.includes == nil {
		p.includes = make(map[string]string)
	}
	p.includes[proto] = pathToProto
}

func (p *protocBuilder) build(ctx context.Context, outDir string) exec.Cmd {
	cmd := exec.CommandContext(ctx, "protoc")
	for proto := range p.includes {
		cmd.Args = append(cmd.Args, proto)
	}
	for _, path := range p.includes {
		cmd.Args = append(cmd.Args, "-I", path)
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("--go_out=:%s", outDir))
	return *cmd
}
