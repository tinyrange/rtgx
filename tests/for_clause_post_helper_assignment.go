package main

func rtg0408Next(x int) int { return x + 3 }
func appMain(args []string) int {
	sum := 0
	for i := 0; i < 7; i = rtg0408Next(i) {
		sum = sum + i
	}
	if sum != 9 {
		print("RTG-0408 post helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
