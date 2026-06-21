package main

func rtg0478Double(x int) int { return x * 2 }
func appMain(args []string) int {
	if rtg0478Double(6) != 12 {
		print("RTG-0478 one param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
