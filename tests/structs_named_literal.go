package main

type Rtg0610Pair struct {
	a int
	b int
}

func appMain(args []string) int {
	var p Rtg0610Pair
	for {
		p = Rtg0610Pair{b: 8, a: 2}
		break
	}
	if p.a+p.b != 10 {
		print("RTG-0610 named literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
