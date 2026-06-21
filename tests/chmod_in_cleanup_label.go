package main

func appMain(args []string) int {
	fd := open("rtg_0791_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RTG-0791 open failed\n")
		return 1
	}
	goto cleanup
cleanup:
	if chmod(fd, 420) != 0 {
		print("RTG-0791 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RTG-0791 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
