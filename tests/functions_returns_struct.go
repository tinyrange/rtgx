package main

type rtg0488Pair struct {
	a int
	b int
}

func rtg0488Make() rtg0488Pair { return rtg0488Pair{a: 4, b: 5} }
func appMain(args []string) int {
	p := rtg0488Make()
	if p.a+p.b != 9 {
		print("RTG-0488 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
