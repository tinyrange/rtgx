package main

func rtg0545A(n int) int {
	return rtg0545B(n) + 2
}

func rtg0545B(n int) int {
	return n * 3
}

func appMain(args []string) int {
	if rtg0545A(5) == 17 && rtg0545B(2) == 6 {
		print("PASS\n")
		return 0
	}
	print("RTG-0545 nested return failed\n")
	return 1
}
