package main

type rtg0670Count int

func rtg0670Make() rtg0670Count {
	return rtg0670Count(12)
}

func appMain(args []string) int {
	if int(rtg0670Make()) != 12 {
		print("RTG-0670 named return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
