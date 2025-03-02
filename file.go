package main

import (
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
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
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if err := replaceImport(path, oldImportRoot, newImportRoot); err != nil {
				return err
			}
		}
		return nil
	})
}

// Reads goFilePath, a go file, and parses its AST. Replaces any imports that
// begin with a forward slash + oldImportPath with newImportPath.
//
// ex: "protogen/cosmos-stratum-integtest/stratum-integtest-proto-definition/src/main/proto/stratumintegtest.pb.go",
// "/cosmos-stratum-function-contract",
// "github.com/jeanbza/protocw/protogen/cosmos-stratum-function-contract"
func replaceImport(goFilePath, oldImportRoot, newImportRoot string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, goFilePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	changed := false
	for _, imp := range node.Imports {
		if strings.HasPrefix(imp.Path.Value, `"`+oldImportRoot) {
			imp.Path.Value = `"` + newImportRoot + strings.TrimPrefix(imp.Path.Value, `"`+oldImportRoot)
			changed = true
		}
	}

	if changed {
		file, err := os.Create(goFilePath)
		if err != nil {
			return err
		}
		defer file.Close()

		if err := printer.Fprint(file, fset, node); err != nil {
			return err
		}
	}

	return nil
}
