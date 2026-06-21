package main

func rtg0799Inner(fd int) int {
	return chmod(fd, 420)
}

func rtg0799Outer(fd int) int {
	return rtg0799Inner(fd)
}

func appMain(args []string) int {
	fd := open("rtg_0799_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0799 open failed\n")
		return 1
	}
	if rtg0799Outer(fd) != 0 {
		print("RTG-0799 nested chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0799 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
