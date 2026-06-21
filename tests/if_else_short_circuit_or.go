package main

var rtg0364Touched int

func rtg0364Mutate() bool { rtg0364Touched = 1; return false }
func appMain(args []string) int {
	ok := false
	if true || rtg0364Mutate() {
		ok = true
	}
	if !ok || rtg0364Touched != 0 {
		print("RTG-0364 short or failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
