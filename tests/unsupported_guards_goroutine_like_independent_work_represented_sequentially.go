package main

func rtg0835WorkA() int {
	return 3
}

func rtg0835WorkB() int {
	return 4
}

func appMain(args []string) int {
	if rtg0835WorkA()*rtg0835WorkB() != 12 {
		print("RTG-0835 sequential work failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
