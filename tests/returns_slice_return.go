package main

func rtg0536Slice() []int {
	var xs []int
	for i := 0; i < 4; i = i + 1 {
		if i == 1 {
			continue
		}
		xs = append(xs, i)
	}
	return xs
}

func appMain(args []string) int {
	xs := rtg0536Slice()
	if len(xs) != 3 || xs[1] != 2 {
		print("RTG-0536 slice return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
