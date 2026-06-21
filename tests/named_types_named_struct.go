package main

type Rtg0656Point struct {
	x int
	y int
}

func appMain(args []string) int {
	p := Rtg0656Point{x: 2, y: 5}
	if p.x < p.y {
		print("PASS\n")
		return 0
	}
	print("RTG-0656 named struct failed\n")
	return 1
}
