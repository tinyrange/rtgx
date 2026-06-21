package main

var rtg0547Text string = "abcde"

func rtg0547Len(s string) int {
	return len(s) + len(rtg0547Text)
}

func appMain(args []string) int {
	if rtg0547Len("xy") != 7 {
		print("RTG-0547 len return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
