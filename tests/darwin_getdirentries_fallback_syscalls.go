package main

func rtgTargetIsDarwin() bool { return false }

func syscall(num int, fd int, buf []byte, size int) int { return -1 }

func exerciseDarwinDirectoryFallbacks() {
	buf := make([]byte, 8)
	if rtgTargetIsDarwin() {
		n := syscall(217, -1, buf, len(buf))
		if n < 0 {
			n = syscall(61, -1, buf, len(buf))
		}
		if n < 0 {
			syscall(220, -1, buf, len(buf))
		}
	}
}

func appMain(args []string, env []string) int {
	exerciseDarwinDirectoryFallbacks()
	print("PASS\n")
	return 0
}
