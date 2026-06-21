package main

type Rtg0651Count int

func appMain(args []string) int {
	var n Rtg0651Count = 4
	if n != 4 {
		print("RTG-0651 named int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
