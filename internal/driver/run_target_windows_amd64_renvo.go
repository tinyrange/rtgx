//go:build renvo && windows && amd64

package driver

func renvoRunTarget() string              { return "windows/amd64" }
func renvoRunTargetID() int               { return 5 }
func renvoRunWindowsPointerSize() int     { return 8 }
func renvoRunWindowsStartupSize() int     { return 104 }
func renvoRunWindowsProcessInfoSize() int { return 24 }
func renvoRunStringWordStride() int       { return 2 }
