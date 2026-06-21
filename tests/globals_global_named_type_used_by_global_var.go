package main

type rtg0698Named int

var rtg0698Value rtg0698Named = rtg0698Named(14)

func appMain(args []string) int {
	if int(rtg0698Value) != 14 {
		print("RTG-0698 named global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
