package main

var rtg0528Want int = 42

func rtg0528Value() int {
	return 42
}

func appMain(args []string) int {
	if rtg0528Value() != rtg0528Want {
		print("RTG-0528 literal return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
