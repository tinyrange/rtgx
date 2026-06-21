package main

type Rtg0617Inner struct{ value int }
type Rtg0617Outer struct{ inner Rtg0617Inner }

func appMain(args []string) int {
	var xs []int
	outer := Rtg0617Outer{inner: Rtg0617Inner{value: 10}}
	xs = append(xs, outer.inner.value)
	if len(xs) != 1 || xs[0] != 10 {
		print("RTG-0617 nested field read failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
