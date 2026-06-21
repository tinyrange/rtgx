package main

func rtg0833StepA(x int) int {
	return x + 2
}

func rtg0833StepB(x int) int {
	return x * 3
}

func appMain(args []string) int {
	x := rtg0833StepA(4)
	x = rtg0833StepB(x)
	if x != 18 {
		print("RTG-0833 sequencing failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
