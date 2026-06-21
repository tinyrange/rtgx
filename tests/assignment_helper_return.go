package main

func rtg0345Value() int { return 12 }

func appMain(args []string) int {
	x := 0
	x = rtg0345Value()
	if x != 12 {
		print("RTG-0345 helper assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
