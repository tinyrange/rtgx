//go:build !rtg

package driver

import (
	"os"
	"os/exec"
	"path/filepath"
)

type CommandBackend struct {
	Path string
	Args []string
	Env  []string
}

func (b CommandBackend) CompileUnit(unit []byte, target string, strip bool) ([]byte, bool) {
	if b.Path == "" || target == "" || len(unit) == 0 {
		return nil, false
	}
	dir, err := os.MkdirTemp("", "rtg-backend-*")
	if err != nil {
		return nil, false
	}
	defer os.RemoveAll(dir)

	unitPath := filepath.Join(dir, "input.rtgu")
	outputPath := filepath.Join(dir, "output")
	if err := os.WriteFile(unitPath, unit, 0o644); err != nil {
		return nil, false
	}

	args := make([]string, 0, len(b.Args)+7)
	args = append(args, b.Args...)
	args = append(args, "-t", target)
	if strip {
		args = append(args, "-s")
	}
	args = append(args, "-o", outputPath, unitPath)
	cmd := exec.Command(b.Path, args...)
	if len(b.Env) > 0 {
		cmd.Env = append(os.Environ(), b.Env...)
	}
	if err := cmd.Run(); err != nil {
		return nil, false
	}
	data, err := os.ReadFile(outputPath)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
