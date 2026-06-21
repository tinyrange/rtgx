package main

var rtg0517Trace []int

func rtg0517Push(n int) {
	if n == 0 {
		return
	}
	rtg0517Trace = append(rtg0517Trace, n)
	rtg0517Push(n - 1)
}

func appMain(args []string) int {
	rtg0517Push(3)
	if len(rtg0517Trace) != 3 {
		print("RTG-0517 global write len failed\n")
		return 1
	}
	if rtg0517Trace[0]+rtg0517Trace[2] != 4 {
		print("RTG-0517 global write order failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
