package main

type Rtg0534Pair struct {
	a int
	b bool
}

func rtg0534Make() Rtg0534Pair {
	return Rtg0534Pair{9, true}
}

func appMain(args []string) int {
	for i := 0; i < 1; i = i + 1 {
		p := rtg0534Make()
		if p.a != 9 || !p.b {
			print("RTG-0534 struct return failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
