package main

type rtg0669Score int

func rtg0669Take(v rtg0669Score) int {
	return int(v) + 2
}

func appMain(args []string) int {
	if rtg0669Take(rtg0669Score(5)) != 7 {
		print("RTG-0669 named parameter failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
