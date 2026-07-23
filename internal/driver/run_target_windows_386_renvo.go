//go:build renvo && windows && 386

package driver

func renvoRunTarget() string              { return "windows/386" }
func renvoRunTargetID() int               { return 6 }
func renvoRunWindowsPointerSize() int     { return 4 }
func renvoRunWindowsStartupSize() int     { return 68 }
func renvoRunWindowsProcessInfoSize() int { return 16 }
func renvoRunStringWordStride() int       { return 4 }
