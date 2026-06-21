package main

func appMain(args []string) int {
	if !(-8 < -3) {
		print("RTG-0186 negative_integer_comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
