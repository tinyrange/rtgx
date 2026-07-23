//go:build renvo && linux && 386

package driver

func renvoRunTarget() string        { return "linux/386" }
func renvoRunTargetID() int         { return 2 }
func renvoRunMmapSyscall() int      { return 192 }
func renvoRunMprotectSyscall() int  { return 125 }
func renvoRunMunmapSyscall() int    { return 91 }
func renvoRunStringWordStride() int { return 4 }
