package main

func rtg0497Loop() int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		sum = sum + i
	}
	return sum
}
func appMain(args []string) int {
	if rtg0497Loop() != 6 {
		print("RTG-0497 return after loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
