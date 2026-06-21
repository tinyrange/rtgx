package main

func appMain(args []string) int {
	if !rtg0530Less(3, 9) {
		print("RTG-0530 bool return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0530Less(a int, b int) bool {
	return a < b
}
