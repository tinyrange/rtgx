package main

const rtg0546Want = 14

func rtg0546Value() int {
	return 14
}

func appMain(args []string) int {
	value := rtg0546Value()
	if value != rtg0546Want {
		print("RTG-0546 stored return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
