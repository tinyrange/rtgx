package main

func appMain(args []string) int {
	path := "rtg_0733_open.tmp"
	fd := open(path, O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0733 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0733 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
