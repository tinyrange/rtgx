package main

const rtg0823A = 2

var rtg0823B int = 5

func appMain(args []string) int {
	if rtg0823A+rtg0823B != 7 {
		print("RTG-0823 compact declarations failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
