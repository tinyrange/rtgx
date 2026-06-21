package main

func rtg0819One() int { return 1 }
func appMain(args []string) int {
	if rtg0819One() != 1 {
		print("RTG-0819 dense helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
