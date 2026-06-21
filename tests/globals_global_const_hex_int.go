package main

const rtg0677Value = 0x2a

func appMain(args []string) int {
	if rtg0677Value != 42 {
		print("RTG-0677 hex const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
