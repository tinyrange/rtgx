package main

func appMain(args []string) int {
	if !((2+3)*4 == 20) {
		print("RTG-0259 parentheses_around_arithmetic_sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
