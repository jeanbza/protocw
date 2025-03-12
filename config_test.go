package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "config.yml")
	defer func() { os.RemoveAll(dir) }()

	p := filepath.Join(dir, "some.proto")
	if err := os.WriteFile(p, []byte(`
syntax = "proto3";
package test;
message Foo { string hello = 1; }`), 0755); err != nil {
		t.Fatal(err)
	}

	configContents := fmt.Sprintf(`
- repo: https://github.com/jeanbza/example1.git
  protos:
    - foo.proto
- repo: https://github.com/jeanbza/example2.git
  protos:
    - bar.proto
- localpath: %s
`, p)
	if err := os.WriteFile(f, []byte(configContents), 0755); err != nil {
		t.Fatal(err)
	}

	// SUT.
	got, err := loadConfig(f)
	if err != nil {
		t.Fatalf("error loading config: %v\n\n%s", err, configContents)
	}

	// Expect.
	want := []Include{
		{Repo: "https://github.com/jeanbza/example1.git", Protos: []string{"foo.proto"}},
		{Repo: "https://github.com/jeanbza/example2.git", Protos: []string{"bar.proto"}},
		{LocalPath: p},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("want -, got +: %s", diff)
	}
}
