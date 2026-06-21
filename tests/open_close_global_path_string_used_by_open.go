package main

var rtg0746Path string = "rtg_0746_open.tmp"

func appMain(args []string) int {
	fd := open(rtg0746Path, O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0746 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0746 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
