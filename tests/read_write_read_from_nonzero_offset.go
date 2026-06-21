package main

func appMain(args []string) int {
	fd := open("rtg_0759_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0759 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RTG-0759 seed write failed\n")
		return 1
	}
	r := []byte(" ")
	if read(fd, r, 2) != 1 {
		print("RTG-0759 read failed\n")
		return 1
	}
	if r[0] != 'c' {
		print("RTG-0759 value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0759 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
