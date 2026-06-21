package main

func rtg0550Check(a int, b int) bool {
	return a+b == 9
}

func appMain(args []string) int {
	if !rtg0550Check(4, 5) {
		print("RTG-0550 first assertion failed\n")
		return 1
	}
	if rtg0550Check(3, 5) {
		print("RTG-0550 second assertion failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
