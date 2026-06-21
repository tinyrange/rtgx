package main

type rtg0675Int int
type rtg0675Byte byte

func appMain(args []string) int {
	x := rtg0675Int(90)
	y := rtg0675Byte(int(x))
	z := int(byte(y))
	if z != 90 {
		print("RTG-0675 conversion chain failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
