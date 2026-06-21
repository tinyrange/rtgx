package main

const rtg0807Value = 7

/* block comment between declarations */

func appMain(args []string) int {
	if rtg0807Value != 7 {
		print("RTG-0807 block comment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
