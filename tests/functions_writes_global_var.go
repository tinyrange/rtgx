package main

var rtg0494Global int

func rtg0494Set() { rtg0494Global = 20 }
func appMain(args []string) int {
	rtg0494Set()
	if rtg0494Global != 20 {
		print("RTG-0494 global var failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
