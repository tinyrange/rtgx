package main

func rtg0529Value(a int, b int) int {
	return a*b + a - b
}

func appMain(args []string) int {
	if rtg0529Value(6, 4) != 26 {
		print("RTG-0529 expression return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
