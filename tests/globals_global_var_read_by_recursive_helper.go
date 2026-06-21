package main

var rtg0695Step int = 3

func rtg0695Sum(n int) int {
	if n == 0 {
		return 0
	}
	return rtg0695Step + rtg0695Sum(n-1)
}

func appMain(args []string) int {
	if rtg0695Sum(4) != 12 {
		print("RTG-0695 recursive global read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
