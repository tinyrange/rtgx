package main

type rtg0690Pair struct {
	left  int
	right int
}

var rtg0690Value rtg0690Pair = rtg0690Pair{left: 4, right: 8}

func appMain(args []string) int {
	if rtg0690Value.left+rtg0690Value.right != 12 {
		print("RTG-0690 struct literal global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
