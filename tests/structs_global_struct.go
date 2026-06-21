package main

const rtg0621Want = 23

type Rtg0621Box struct{ value int }

var rtg0621Global = Rtg0621Box{value: rtg0621Want}

func appMain(args []string) int {
	if rtg0621Global.value != 23 {
		print("RTG-0621 global struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
