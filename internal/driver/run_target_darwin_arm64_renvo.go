//go:build renvo && darwin && arm64

package driver

func renvoRunTarget() string        { return "darwin/arm64" }
func renvoRunTargetID() int         { return 8 }
func renvoRunStringWordStride() int { return 2 }
