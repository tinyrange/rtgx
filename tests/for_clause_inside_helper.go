package main

func rtg0417Sum() int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		sum = sum + i
	}
	return sum
}
func appMain(args []string) int {
	if rtg0417Sum() != 6 {
		print("RTG-0417 helper for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
