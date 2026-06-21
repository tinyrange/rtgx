package main

const rtg0800Mode = 420

func appMain(args []string) int {
	fd := open("rtg_0800_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0800 open failed\n")
		return 1
	}
	if chmod(fd, rtg0800Mode) != 0 {
		print("RTG-0800 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0800 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
