package main

type Rtg0616Counter struct{ value int }

func appMain(args []string) int {
	var xs []int
	xs = append(xs, 5)
	c := Rtg0616Counter{value: xs[0]}
	c.value += 7
	if c.value != 12 {
		print("RTG-0616 compound field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
