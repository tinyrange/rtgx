package main

var rtg0527Hit int

func rtg0527Mark() {
	rtg0527Hit = 7
}

func appMain(args []string) int {
	rtg0527Mark()
	if rtg0527Hit != 7 {
		print("RTG-0527 fallthrough return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
