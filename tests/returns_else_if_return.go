package main

type Rtg0539Box struct {
	value int
}

func rtg0539Pick(n int) int {
	if n < 0 {
		return 1
	} else if n == 0 {
		return 2
	}
	return 3
}

func appMain(args []string) int {
	b := Rtg0539Box{value: rtg0539Pick(0)}
	if b.value != 2 {
		print("RTG-0539 else-if return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
