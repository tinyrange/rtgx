package main

var rtg0692Ptr *int

func appMain(args []string) int {
	if rtg0692Ptr != nil {
		print("RTG-0692 pointer zero global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
