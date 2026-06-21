package main

func rtg0499Ok() bool { return true }
func appMain(args []string) int {
	if rtg0499Ok() {
		print("PASS\n")
		return 0
	}
	print("RTG-0499 condition call failed\n")
	return 1
}
