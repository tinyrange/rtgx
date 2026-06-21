package main

func rtg0831IntPath(x int) int {
	return x + 1
}

func rtg0831BytePath(x byte) int {
	return int(x) + 2
}

func appMain(args []string) int {
	if rtg0831IntPath(4)+rtg0831BytePath(byte(5)) != 12 {
		print("RTG-0831 split helpers failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
