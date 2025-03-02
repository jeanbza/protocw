package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("protocw <config file> <out dir>")
		os.Exit(1)
	}
	ctx := context.Background()
	configFile := os.Args[1]
	outDir := os.Args[2]
	if err := run(ctx, configFile, outDir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configFile, outDir string) error {
	c, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	tmpRoot, err := os.MkdirTemp("", "protocw")
	if err != nil {
		return err
	}

	var b protocBuilder
	for _, d := range c.Includes {
		if err := cloneInto(ctx, tmpRoot, d.Repo); err != nil {
			return fmt.Errorf("error cloning %s into %s: %v", d.Repo, tmpRoot, err)
		}
		for _, proto := range d.Protos {
			path, err := searchDirForProto(tmpRoot, proto)
			if err != nil {
				return fmt.Errorf("error searching for %s in %s: %v", proto, tmpRoot, err)
			}
			fmt.Printf("Found proto %s in %s at %s\n", proto, d.Repo, path)
			if err := b.addInclude(proto, path, tmpRoot); err != nil {
				return err
			}
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("error creating %s: %v", outDir, err)
	}

	cmd := b.build(ctx, outDir)
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s:\n%s%s: %v", cmd.Args, o.String(), e.String(), err)
	}

	mr, err := modRoot(ctx)
	if err != nil {
		return err
	}
	for _, i := range c.Includes {
		oldImportRoot := i.Repo
		newImportRoot := filepath.Join(mr, outDir)
		fmt.Printf("replacing %s with %s in %s\n", oldImportRoot, newImportRoot, outDir)
		if err := replaceImports(outDir, oldImportRoot, newImportRoot); err != nil {
			return err
		}
	}

	return nil
}

func modRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "mod", "graph")
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error running %s:\n%s%s: %v", cmd.Args, o.String(), e.String(), err)
	}
	scanner := bufio.NewScanner(&o)
	if scanner.Scan() {
		t := scanner.Text()
		s := strings.Split(t, " ")
		if len(s) == 0 {
			return "", fmt.Errorf("error running %s: output %s doesn't look like it contains a go module", cmd.Args, s)
		}
		return s[0], nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error running %s: %v", cmd.Args, err)
	}
	return "", fmt.Errorf("error running %s: got not output", cmd.Args)
}

type protocTriplet struct {
	// ex: buf/validate/validate.proto.
	protoImportPath string
	// The -I.
	//
	// ex: /path/to/protovalidate/proto/protovalidate.
	includeDir string
	// Where this gets placed in the protogen dir.
	// Concretely: the path that goes after --go_opt=M<proto>=cosmoscommons/.
	//
	// ex: buf/validate (which translates to <out dir>/buf/validate).
	outDir string
}

type protocBuilder struct {
	includes []*protocTriplet
}

func (p *protocBuilder) addInclude(protoImportPath, includeDir, tmpRoot string) error {
	outDir := strings.TrimPrefix(includeDir, tmpRoot)
	if outDir == "" {
		return fmt.Errorf("unable to determine a directory for generation of %s", protoImportPath)
	}
	p.includes = append(p.includes, &protocTriplet{
		protoImportPath: protoImportPath,
		includeDir:      includeDir,
		outDir:          outDir,
	})
	return nil
}

func (p *protocBuilder) build(ctx context.Context, outDir string) exec.Cmd {
	cmd := exec.CommandContext(ctx, "protoc")
	for _, t := range p.includes {
		cmd.Args = append(cmd.Args, t.protoImportPath)
	}
	for _, t := range p.includes {
		cmd.Args = append(cmd.Args, "-I", t.includeDir)
	}
	for _, t := range p.includes {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--go_opt=M%s=%s", t.protoImportPath, t.outDir))
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("--go_out=:%s", outDir))
	return *cmd
}
