package main

var rtg0683Value int

func appMain(args []string) int {
	if rtg0683Value != 0 {
		print("RTG-0683 int zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
