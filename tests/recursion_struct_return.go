package main

type Rtg0518Pair struct {
	a int
	b int
}

func rtg0518Make(n int) Rtg0518Pair {
	if n == 0 {
		return Rtg0518Pair{a: 1, b: 1}
	}
	p := rtg0518Make(n - 1)
	return Rtg0518Pair{a: p.a + 1, b: p.b + int(byte(n))}
}

func appMain(args []string) int {
	p := rtg0518Make(3)
	if p.a != 4 || p.b != 7 {
		print("RTG-0518 struct return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
