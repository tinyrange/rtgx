package main

type Rtg0619Pair struct {
	a int
	b int
}

func rtg0619Sum(p Rtg0619Pair) int {
	total := p.a
	total += p.b
	return total
}

func appMain(args []string) int {
	if rtg0619Sum(Rtg0619Pair{a: 3, b: 4}) != 7 {
		print("RTG-0619 struct by value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
