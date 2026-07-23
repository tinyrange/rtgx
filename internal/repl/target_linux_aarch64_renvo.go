//go:build renvo && linux && (aarch64 || arm64)

package repl

func replTarget() string { return "linux/aarch64" }
func replTargetID() int  { return 3 }
