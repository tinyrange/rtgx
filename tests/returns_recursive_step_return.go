package main

func rtg0544Step(n int) int {
	if n == 0 {
		return 1
	}
	v := rtg0544Step(n - 1)
	v *= 2
	return v
}

func appMain(args []string) int {
	if rtg0544Step(4) != 16 {
		print("RTG-0544 recursive step return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
