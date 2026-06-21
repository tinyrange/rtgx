package main

func rtg0510Caller(n int) int {
	for {
		if n <= 0 {
			break
		}
		return rtg0510Later(n)
	}
	return 0
}

func appMain(args []string) int {
	if rtg0510Caller(4) != 10 {
		print("RTG-0510 late helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0510Later(n int) int {
	if n == 0 {
		return 0
	}
	return n + rtg0510Later(n-1)
}
