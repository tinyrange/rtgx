//go:build renvo && linux && (aarch64 || arm64)

package driver

func renvoRunTarget() string        { return "linux/aarch64" }
func renvoRunTargetID() int         { return 3 }
func renvoRunMmapSyscall() int      { return 222 }
func renvoRunMprotectSyscall() int  { return 226 }
func renvoRunMunmapSyscall() int    { return 215 }
func renvoRunStringWordStride() int { return 2 }
