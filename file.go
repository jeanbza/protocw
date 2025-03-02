package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

// Walks outDir, replacing oldImportRoot with newImportRoot in every .go file it
// sees.
func replaceImports(outDir, oldImportRoot, newImportRoot string) error {
	return filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		fmt.Println("Inspecting", path)
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			output := bytes.Replace(input, []byte(oldImportRoot), []byte(newImportRoot), -1)
			if !bytes.Equal(input, output) {
				if err = os.WriteFile(path, output, info.Mode().Perm()); err != nil {
					return err
				}
				fmt.Printf("Replaced %s with %s in %s\n", oldImportRoot, newImportRoot, path)
			}
		}
		return nil
	})
}
