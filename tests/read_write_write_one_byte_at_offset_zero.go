package main

func appMain(args []string) int {
	fd := open("rtg_0751_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0751 open failed\n")
		return 1
	}
	b := []byte("A")
	if write(fd, b, 0) != 1 {
		print("RTG-0751 write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0751 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
