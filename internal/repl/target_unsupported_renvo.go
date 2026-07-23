//go:build renvo && !((linux && (amd64 || 386 || aarch64 || arm64 || arm)) || (windows && (amd64 || 386 || arm64)) || (darwin && arm64))

package repl

func replTarget() string { return "" }
func replTargetID() int  { return 0 }
