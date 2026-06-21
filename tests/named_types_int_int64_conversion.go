package main

func rtg0663Convert(n int) int64 {
	if n == 0 {
		return int64(0)
	}
	return int64(1) + rtg0663Convert(n-1)
}

func appMain(args []string) int {
	if rtg0663Convert(5) != 5 {
		print("RTG-0663 int int64 conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
