package main

func appMain(args []string) int {
	if !(9<<0 == 9) {
		print("RTG-0226 left_shift_by_zero failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
