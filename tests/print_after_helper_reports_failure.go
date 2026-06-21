package main

func rtg0704Ok() int {
	return 0
}

func appMain(args []string) int {
	if rtg0704Ok() != 0 {
		print("RTG-0704 helper failure diagnostic\n")
		return 1
	}
	print("PASS\n")
	return 0
}
