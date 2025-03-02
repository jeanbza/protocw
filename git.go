package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
	return nil
}
