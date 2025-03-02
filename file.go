package main

import (
	"fmt"
	"os"
	"path/filepath"
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
