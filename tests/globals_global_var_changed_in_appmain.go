package main

var rtg0693Value int = 4

func appMain(args []string) int {
	rtg0693Value = rtg0693Value + 6
	if rtg0693Value != 10 {
		print("RTG-0693 appMain global mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
