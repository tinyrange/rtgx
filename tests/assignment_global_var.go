package main

var rtg0349Global int

func appMain(args []string) int {
	rtg0349Global = 17
	if rtg0349Global != 17 {
		print("RTG-0349 global assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
