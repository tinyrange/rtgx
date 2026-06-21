package main

func appMain(args []string) int {
	fd := open("rtg_0771_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0771 open failed\n")
		return 1
	}
	i := 0
	for i < 4 {
		b := []byte("a")
		b[0] = byte(65 + i)
		if write(fd, b, int64(i)) != 1 {
			print("RTG-0771 loop write failed\n")
			return 2
		}
		i = i + 1
	}
	if close(fd) != 0 {
		print("RTG-0771 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
