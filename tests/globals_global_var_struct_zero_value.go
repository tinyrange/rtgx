package main

type rtg0689Pair struct {
	left  int
	right int
}

var rtg0689Value rtg0689Pair

func appMain(args []string) int {
	if rtg0689Value.left != 0 {
		print("RTG-0689 struct zero left failed\n")
		return 1
	}
	if rtg0689Value.right != 0 {
		print("RTG-0689 struct zero right failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
