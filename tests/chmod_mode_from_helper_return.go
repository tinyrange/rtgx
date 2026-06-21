package main

func rtg0783Mode() int {
	return 420
}

func appMain(args []string) int {
	fd := open("rtg_0783_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0783 open failed\n")
		return 1
	}
	if chmod(fd, rtg0783Mode()) != 0 {
		print("RTG-0783 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0783 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
