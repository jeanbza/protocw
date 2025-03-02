package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

var configFilePath = flag.String("c", "", "config file path")
var outDirPath = flag.String("o", "protogen", "out dir path")
var verboseFlag = flag.Bool("v", false, "verbose output")

func main() {
	flag.Parse()

	if *configFilePath == "" || *outDirPath == "" {
		fmt.Println("protocw -c=<config file> [-o=<out dir>] [-v]")
		os.Exit(1)
	}

	logger := log.New(io.Discard, "", log.LstdFlags)
	if *verboseFlag {
		logger = log.Default()
	}

	if err := run(context.Background(), logger, *configFilePath, *outDirPath); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *log.Logger, configFile, outDir string) error {
	i, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("error loading config: %v", err)
	}

	tmpRoot, err := os.MkdirTemp("", "protocw")
	if err != nil {
		return err
	}

	mr, err := modRoot(ctx)
	if err != nil {
		return err
	}

	grp, grpCtx := errgroup.WithContext(ctx)
	b := newProtocBuilder(mr, outDir)

	for _, d := range i {
		grp.Go(func() error {
			logger.Printf("Cloning %s into %s", d.Repo, tmpRoot)
			if err := cloneInto(grpCtx, tmpRoot, d.Repo); err != nil {
				return fmt.Errorf("error cloning %s into %s: %v", d.Repo, tmpRoot, err)
			}
			for _, proto := range d.Protos {
				path, err := searchDirForProto(tmpRoot, proto)
				if err != nil {
					return fmt.Errorf("error searching for %s in %s: %v", proto, tmpRoot, err)
				}
				logger.Printf("Found %s in %s at %s\n", proto, d.Repo, path)
				if err := b.addInclude(proto, path, tmpRoot); err != nil {
					return err
				}
			}
			return nil
		})
	}

	if err := grp.Wait(); err != nil {
		return err
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("error creating %s: %v", outDir, err)
	}

	cmd := b.build(ctx)
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	logger.Printf("Running %v\n", cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s:\n%s%s: %v", cmd.Args, o.String(), e.String(), err)
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
	// The import path in the .proto file.
	//
	// ex: buf/validate/validate.proto.
	protoImportPath string
	// The -I.
	//
	// ex: /path/to/protovalidate/proto/protovalidate.
	includeDir string
	// Where to find this in the outDir.
	// Concretely: the path that goes after --go_opt=M<protoImportPath>=<newImportPath>.
	//
	// ex: buf/validate (which translates to <out dir>/buf/validate).
	newImportPath string
}

type protocBuilder struct {
	modRoot, outDir string

	mu       sync.Mutex
	includes []*protocTriplet
}

func newProtocBuilder(modRoot, outDir string) *protocBuilder {
	return &protocBuilder{modRoot: modRoot, outDir: outDir}
}

func (b *protocBuilder) addInclude(protoImportPath, includeDir, tmpRoot string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	relativeToOutDir := strings.TrimPrefix(includeDir, tmpRoot)
	if relativeToOutDir == "" {
		return fmt.Errorf("unable to determine a directory for generation of %s", protoImportPath)
	}
	newImportPath := filepath.Join(b.modRoot, b.outDir, relativeToOutDir)
	b.includes = append(b.includes, &protocTriplet{
		protoImportPath: protoImportPath,
		includeDir:      includeDir,
		newImportPath:   newImportPath,
	})
	return nil
}

func (b *protocBuilder) build(ctx context.Context) exec.Cmd {
	b.mu.Lock()
	defer b.mu.Unlock()
	cmd := exec.CommandContext(ctx, "protoc")
	for _, t := range b.includes {
		cmd.Args = append(cmd.Args, t.protoImportPath)
	}
	for _, t := range b.includes {
		cmd.Args = append(cmd.Args, "-I", t.includeDir)
	}
	for _, t := range b.includes {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--go_opt=M%s=%s", t.protoImportPath, t.newImportPath))
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("--go_opt=module=%s", b.modRoot))
	cmd.Args = append(cmd.Args, "--go_out=.")
	return *cmd
}
