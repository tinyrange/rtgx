package main

func appMain(args []string) int {
	if rtg0480Later() != 5 {
		print("RTG-0480 later helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func rtg0480Later() int { return 5 }
