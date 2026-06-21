package main

type Rtg0620Pair struct {
	a int
	b int
}

func rtg0620Make(ok bool) Rtg0620Pair {
	if ok && true {
		return Rtg0620Pair{a: 8, b: 1}
	}
	return Rtg0620Pair{a: 0, b: 0}
}

func appMain(args []string) int {
	p := rtg0620Make(true)
	if p.a-p.b != 7 {
		print("RTG-0620 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
