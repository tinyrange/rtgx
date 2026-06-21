package main

func rtg0479Mix(a int, b int) int { return a*10 + b }
func appMain(args []string) int {
	if rtg0479Mix(3, 4) != 34 {
		print("RTG-0479 two param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
