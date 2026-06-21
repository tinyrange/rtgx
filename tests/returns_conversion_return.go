package main

func rtg0548Convert(b byte) int {
	return int(b) - int('a')
}

func appMain(args []string) int {
	if rtg0548Convert('d') != 3 {
		print("RTG-0548 conversion return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
