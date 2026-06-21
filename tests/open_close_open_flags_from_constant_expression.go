package main

func appMain(args []string) int {
	flags := O_RDWR | O_CREATE | O_TRUNC
	fd := open("rtg_0735_open.tmp", flags)
	if fd < 0 {
		print("RTG-0735 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0735 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
