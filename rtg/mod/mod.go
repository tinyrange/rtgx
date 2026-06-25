package mod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Module struct {
	Root string
	Path string
}

func Find(start string) (Module, error) {
	if start == "" {
		start = "."
	}
	abs, err := filepath.Abs(start)
	if err != nil {
		return Module{}, err
	}
	info, err := os.Stat(abs)
	if err == nil && !info.IsDir() {
		abs = filepath.Dir(abs)
	}
	for {
		path := filepath.Join(abs, "go.mod")
		data, err := os.ReadFile(path)
		if err == nil {
			modulePath, err := ParseModulePath(string(data))
			if err != nil {
				return Module{}, fmt.Errorf("%s: %w", path, err)
			}
			return Module{Root: abs, Path: modulePath}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return Module{}, fmt.Errorf("go.mod not found from %s", start)
		}
		abs = parent
	}
}

func ParseModulePath(data string) (string, error) {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = stripLineComment(line)
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("module directive not found")
}

func stripLineComment(line string) string {
	for i := 0; i+1 < len(line); i++ {
		if line[i] == '/' && line[i+1] == '/' {
			return line[:i]
		}
	}
	return line
}
