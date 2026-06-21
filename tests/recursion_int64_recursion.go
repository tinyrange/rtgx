package main

type Rtg0514Total struct {
	value int64
}

func rtg0514Accum(n int64) int64 {
	if n == 0 {
		return 0
	}
	return n + rtg0514Accum(n-1)
}

func appMain(args []string) int {
	t := Rtg0514Total{value: rtg0514Accum(6)}
	if t.value != 21 {
		print("RTG-0514 int64 recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
