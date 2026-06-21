package main

var rtg0641Global int = 25

func appMain(args []string) int {
	var xs []int
	p := &rtg0641Global
	xs = append(xs, *p)
	if xs[0] != 25 {
		print("RTG-0641 global pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
