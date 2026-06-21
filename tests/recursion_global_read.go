package main

var rtg0516Limit int = 4

func rtg0516Product(n int) int {
	if n > rtg0516Limit {
		return 1
	}
	return n * rtg0516Product(n+1)
}

func appMain(args []string) int {
	var checks []int
	checks = append(checks, rtg0516Product(1))
	if checks[0] != 24 {
		print("RTG-0516 global read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
