//go:build renvo && linux && arm

package driver

func renvoRunTarget() string        { return "linux/arm" }
func renvoRunTargetID() int         { return 4 }
func renvoRunMmapSyscall() int      { return 192 }
func renvoRunMprotectSyscall() int  { return 125 }
func renvoRunMunmapSyscall() int    { return 91 }
func renvoRunStringWordStride() int { return 4 }
