package main

func rtg0815Value() int { return 8 }

func appMain(args []string) int {
	if rtg0815Value() != 8 {
		print("RTG-0815 same-line return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
