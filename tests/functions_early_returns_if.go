package main

func rtg0496Check(x int) int {
	if x > 3 {
		return 21
	}
	return 0
}
func appMain(args []string) int {
	if rtg0496Check(4) != 21 {
		print("RTG-0496 early return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
