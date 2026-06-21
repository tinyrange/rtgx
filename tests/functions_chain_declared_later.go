package main

func rtg0482First() int { return rtg0482Second() + 1 }
func appMain(args []string) int {
	if rtg0482First() != 9 {
		print("RTG-0482 chain later failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func rtg0482Second() int { return 8 }
