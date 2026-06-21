package main

func rtg0491Set(xs []int) { xs[0] = 16 }
func appMain(args []string) int {
	var xs []int
	xs = append(xs, 0)
	rtg0491Set(xs)
	if xs[0] != 16 {
		print("RTG-0491 slice mutate failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
