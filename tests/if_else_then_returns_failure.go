package main

func appMain(args []string) int {
	if 5 != 5 {
		print("RTG-0369 early then failure\n")
		return 1
	}
	print("PASS\n")
	return 0
}
