package main

type rtg0828Pair struct {
	a int
	b int
}

func rtg0828Make() rtg0828Pair {
	return rtg0828Pair{a: 4, b: 5}
}

func appMain(args []string) int {
	p := rtg0828Make()
	if p.a+p.b != 9 {
		print("RTG-0828 struct pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
