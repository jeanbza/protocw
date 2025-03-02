package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func cloneInto(ctx context.Context, dir, repo string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repo)
	cmd.Dir = dir
	var o, e bytes.Buffer
	cmd.Stdout = &o
	cmd.Stderr = &e
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %s:\n%s%s: %v", cmd.Args, o.String(), e.String(), err)
	}
	return removeGoPackageOptions(dir)
}

var goPackageRe = regexp.MustCompile(`option go_package = ".*";\n`)

// Removes any occurrences of "go_package [...]" in .proto files.
func removeGoPackageOptions(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(filepath.Base(path), ".") {
			// Ignore hidden dirs, like ".git".
			return filepath.SkipDir
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".proto" {
			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			output := goPackageRe.ReplaceAll(input, []byte(""))

			if err := os.WriteFile(path, output, info.Mode().Perm()); err != nil {
				return err
			}
		}
		return nil
	})
}

func searchDirForProto(dir, proto string) (string, error) {
	var results []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if _, err := os.Lstat(filepath.Join(path, proto)); err == nil {
			results = append(results, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", os.ErrNotExist
	}
	if len(results) > 1 {
		return "", fmt.Errorf("found multiple protos named %s: %v", proto, results)
	}
	return results[0], nil
}
