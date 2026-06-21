package main

const rtg0699A = 8
const rtg0699B = 9

var rtg0699Value int = rtg0699A*rtg0699B + 1

func appMain(args []string) int {
	if rtg0699Value != 73 {
		print("RTG-0699 global const init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
