package main

func rtg0734Path() string {
	return "rtg_0734_open.tmp"
}

func appMain(args []string) int {
	fd := open(rtg0734Path(), O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0734 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0734 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
