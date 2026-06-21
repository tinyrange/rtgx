package main

var rtg0363Touched int

func rtg0363Mutate() bool { rtg0363Touched = 1; return true }
func appMain(args []string) int {
	if false && rtg0363Mutate() {
		print("RTG-0363 impossible\n")
		return 1
	}
	if rtg0363Touched != 0 {
		print("RTG-0363 short and failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
