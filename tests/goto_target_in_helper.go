package main

func rtg0474Helper() int {
	x := 0
	goto target
	x = 9
target:
	return x + 14
}
func appMain(args []string) int {
	if rtg0474Helper() != 14 {
		print("RTG-0474 helper target failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
