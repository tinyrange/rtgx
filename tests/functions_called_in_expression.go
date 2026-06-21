package main

func rtg0498A() int { return 5 }
func appMain(args []string) int {
	x := rtg0498A()*2 + 3
	if x != 13 {
		print("RTG-0498 expression call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
