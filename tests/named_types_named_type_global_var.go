package main

type rtg0671Level int

var rtg0671Global rtg0671Level = rtg0671Level(3)

func appMain(args []string) int {
	rtg0671Global = rtg0671Global + rtg0671Level(4)
	if int(rtg0671Global) != 7 {
		print("RTG-0671 named global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
