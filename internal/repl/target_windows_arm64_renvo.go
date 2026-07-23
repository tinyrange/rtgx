//go:build renvo && windows && arm64

package repl

func replTarget() string { return "windows/arm64" }
func replTargetID() int  { return 10 }
