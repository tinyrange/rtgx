//go:build renvo && linux && amd64

package driver

func renvoRunTarget() string        { return "linux/amd64" }
func renvoRunTargetID() int         { return 1 }
func renvoRunMmapSyscall() int      { return 9 }
func renvoRunMprotectSyscall() int  { return 10 }
func renvoRunMunmapSyscall() int    { return 11 }
func renvoRunStringWordStride() int { return 2 }
