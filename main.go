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
var outDirPath = flag.String("o", "internal/protogen", "out dir path")
var verbose = flag.Bool("v", false, "verbose output")
var grpc = flag.Bool("grpc", false, "generate grpc outputs too")

func main() {
	flag.Parse()

	if *configFilePath == "" || *outDirPath == "" {
		fmt.Println("protocw -c=<config file> [-o=<out dir>] [-v]")
		os.Exit(1)
	}

	logger := log.New(io.Discard, "", log.LstdFlags)
	if *verbose {
		logger = log.Default()
	}

	if err := run(context.Background(), logger, *configFilePath, *outDirPath, *grpc); err != nil {
		fmt.Println("error during run:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *log.Logger, configFile, outDir string, grpc bool) error {
	logger.Println("Starting")
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
	b := newProtocBuilder(mr, outDir, grpc)

	for _, d := range i {
		if d.LocalPath != "" {
			// We don't need to clone, files are local.
			logger.Printf("Local proto %s included\n", d.LocalPath)
			b.addLocalInclude(d.LocalPath)
			continue
		}
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
	logger.Println("Done")
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
	// The import path that other protos use to import this.
	//
	// ex: buf/validate/validate.proto.
	protoImportPath string
	// The -I: where to find it on local disk.
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
	grpc            bool

	mu       sync.Mutex
	includes []*protocTriplet
}

func newProtocBuilder(modRoot, outDir string, grpc bool) *protocBuilder {
	return &protocBuilder{modRoot: modRoot, outDir: outDir, grpc: grpc}
}

func (b *protocBuilder) addLocalInclude(pathToProto string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	dir := filepath.Dir(pathToProto)
	if dir == "" {
		// It's a file without a leading dir.
		dir = strings.TrimSuffix(pathToProto, filepath.Ext(pathToProto))
	}
	b.includes = append(b.includes, &protocTriplet{
		protoImportPath: pathToProto,
		includeDir:      "",
		newImportPath:   filepath.Join(b.modRoot, b.outDir, dir),
	})
}

func (b *protocBuilder) addInclude(protoImportPath, includeDir, tmpRoot string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	relativeToOutDir := strings.TrimPrefix(includeDir, tmpRoot)
	if relativeToOutDir == "" {
		return fmt.Errorf("unable to determine a directory for generation of %s", protoImportPath)
	}
	newImportPath := filepath.Join(b.modRoot, b.outDir, stripInternal(relativeToOutDir))
	b.includes = append(b.includes, &protocTriplet{
		protoImportPath: protoImportPath,
		includeDir:      includeDir,
		newImportPath:   newImportPath,
	})
	return nil
}

// Anything after an internal/ directory can't be imported from outside the
// internal/ directory. https://go.dev/doc/go1.4#internalpackages
func stripInternal(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	var newParts []string
	for _, part := range parts {
		if part != "internal" {
			newParts = append(newParts, part)
		}
	}
	return strings.Join(newParts, string(filepath.Separator))
}

func (b *protocBuilder) build(ctx context.Context) exec.Cmd {
	b.mu.Lock()
	defer b.mu.Unlock()
	cmd := exec.CommandContext(ctx, "protoc")
	for _, t := range b.includes {
		cmd.Args = append(cmd.Args, t.protoImportPath)
	}
	for _, t := range b.includes {
		if t.includeDir == "" {
			continue
		}
		cmd.Args = append(cmd.Args, "-I", t.includeDir)
	}
	for _, t := range b.includes {
		if t.newImportPath == "" {
			continue
		}
		cmd.Args = append(cmd.Args, fmt.Sprintf("--go_opt=M%s=%s", t.protoImportPath, t.newImportPath))
		if b.grpc {
			cmd.Args = append(cmd.Args, fmt.Sprintf("--go-grpc_opt=M%s=%s", t.protoImportPath, t.newImportPath))
		}
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("--go_opt=module=%s", b.modRoot))
	if b.grpc {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--go-grpc_opt=module=%s", b.modRoot))
	}
	cmd.Args = append(cmd.Args, "--go_out=.")
	if b.grpc {
		// --go-grpc_out=. --go-grpc_opt=paths=source_relative
		cmd.Args = append(cmd.Args, "--go-grpc_out=.")
	}
	return *cmd
}
