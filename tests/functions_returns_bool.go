package main

func rtg0484Ok(x int) bool { return x == 4 }
func appMain(args []string) int {
	if !rtg0484Ok(4) {
		print("RTG-0484 bool return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
