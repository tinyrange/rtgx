package main

func rtg0502Fact(n int) int {
	if n <= 1 {
		return 1
	}
	return n * rtg0502Fact(n-1)
}

func appMain(args []string) int {
	value := 5
	if rtg0502Fact(value) != 120 {
		print("RTG-0502 factorial failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
