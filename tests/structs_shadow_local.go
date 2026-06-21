package main

type Rtg0622Node struct{ value int }

var rtg0622Start int = 2

func appMain(args []string) int {
	Rtg0622Node := rtg0622Start + 5
	if Rtg0622Node != 7 {
		print("RTG-0622 shadow local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
