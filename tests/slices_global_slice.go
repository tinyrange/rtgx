package main

var rtg0569Global []int

func appMain(args []string) int {
	rtg0569Global = append(rtg0569Global, 2)
	rtg0569Global = append(rtg0569Global, 3)
	rtg0569Global[0] += 5
	if rtg0569Global[0]+rtg0569Global[1] != 10 {
		print("RTG-0569 global slice failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
