package main

func rtg0371Noop() {}
func appMain(args []string) int {
	x := 1
	if x == 1 {
		rtg0371Noop()
	} else {
		x = 9
	}
	if x != 1 {
		print("RTG-0371 noop then failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
