package main

func appMain(args []string) int {
	a := open("rtg_0737_open.tmpa", O_RDWR|O_CREATE|O_TRUNC)
	b := open("rtg_0737_open.tmpb", O_RDWR|O_CREATE|O_TRUNC)
	if a < 0 {
		print("RTG-0737 open a failed\n")
		return 1
	}
	if b < 0 {
		print("RTG-0737 open b failed\n")
		return 1
	}
	if close(b) != 0 {
		print("RTG-0737 close b failed\n")
		return 1
	}
	if close(a) != 0 {
		print("RTG-0737 close a failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
