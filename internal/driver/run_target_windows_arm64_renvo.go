//go:build renvo && windows && arm64

package driver

func renvoRunTarget() string              { return "windows/arm64" }
func renvoRunTargetID() int               { return 10 }
func renvoRunWindowsPointerSize() int     { return 8 }
func renvoRunWindowsStartupSize() int     { return 104 }
func renvoRunWindowsProcessInfoSize() int { return 24 }
func renvoRunStringWordStride() int       { return 2 }
