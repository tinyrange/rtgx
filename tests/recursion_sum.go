package main

var rtg0503Start int = 6

func rtg0503Sum(n int) int {
	if n == 0 {
		return 0
	}
	return n + rtg0503Sum(n-1)
}

func appMain(args []string) int {
	if rtg0503Sum(rtg0503Start) != 21 {
		print("RTG-0503 sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
