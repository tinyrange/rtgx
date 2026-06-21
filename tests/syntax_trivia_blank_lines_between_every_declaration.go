package main

const rtg0801Value = 5

func appMain(args []string) int {

	if rtg0801Value != 5 {
		print("RTG-0801 blank lines failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
