package main

func rtg0554Make(a int, b int) []int {
	var xs []int
	xs = append(xs, a)
	xs = append(xs, b)
	return xs
}

func appMain(args []string) int {
	xs := rtg0554Make(3, 7)
	if len(xs) != 2 || xs[0]+xs[1] != 10 {
		print("RTG-0554 append two ints failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
