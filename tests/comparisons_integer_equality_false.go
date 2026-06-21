package main

func appMain(args []string) int {
	if !(4 == 5 == false) {
		print("RTG-0177 integer_equality_false failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
