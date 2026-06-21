package main

var rtg0684Value int = 19

func appMain(args []string) int {
	if rtg0684Value != 19 {
		print("RTG-0684 int global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
