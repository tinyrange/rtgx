package main

func rtg0709Status() int {
	return 0
}

func appMain(args []string) int {
	if rtg0709Status() != 0 {
		print("RTG-0709 helper print status failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
