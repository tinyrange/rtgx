//go:build !renvo

// Package runimage executes a host-native Renvo linked image.
package runimage

import (
	"io"
	"os"

	"renvo.dev/internal/linkedimage"
)

type Result struct {
	ExitCode int
	Loader   string
	Err      error
}

type preparedExecutable struct {
	path       string
	extraFiles []*os.File
	close      func()
	loader     string
}

func Run(image linkedimage.Image, script string, args []string, env []string, stdin io.Reader, stdout, stderr io.Writer) Result {
	return runNative(image, script, args, env, stdin, stdout, stderr)
}
