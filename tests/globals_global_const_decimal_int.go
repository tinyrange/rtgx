package main

const rtg0676Value = 42

func appMain(args []string) int {
	if rtg0676Value != 42 {
		print("RTG-0676 decimal const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
