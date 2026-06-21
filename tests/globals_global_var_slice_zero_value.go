package main

var rtg0691Values []int

func appMain(args []string) int {
	if len(rtg0691Values) != 0 {
		print("RTG-0691 slice zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
