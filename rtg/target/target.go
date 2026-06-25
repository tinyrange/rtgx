package target

import (
	"runtime"
	"strings"
)

var supported = []string{
	"linux/amd64",
	"linux/386",
	"linux/aarch64",
	"linux/arm",
	"windows/amd64",
	"windows/386",
	"wasi/wasm32",
}

func Default() string {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if host == "linux/arm64" {
		return "linux/aarch64"
	}
	if Supported(host) {
		return host
	}
	return "linux/amd64"
}

func Supported(name string) bool {
	for _, supportedName := range supported {
		if name == supportedName {
			return true
		}
	}
	return false
}

func List() string {
	return strings.Join(supported, ", ")
}
