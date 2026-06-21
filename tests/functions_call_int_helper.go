package main

func rtg0477Value() int { return 21 }
func appMain(args []string) int {
	if rtg0477Value() != 21 {
		print("RTG-0477 int helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
