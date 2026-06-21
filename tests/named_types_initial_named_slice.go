package main

type Rtg0657List []int

func appMain(args []string) int {
	var xs Rtg0657List
	if len(xs) != 0 {
		print("RTG-0657 initial named slice failed\n")
		return 1
	} else {
		xs = append(xs, 6)
	}
	if xs[0] != 6 {
		print("RTG-0657 named slice failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
