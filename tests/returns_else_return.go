package main

func rtg0538Pick(flag bool) int {
	if flag {
		return 3
	} else {
		return 6
	}
}

func rtg0538Wrap(n int) int {
	if n == 0 {
		return rtg0538Pick(false)
	}
	return rtg0538Wrap(n - 1)
}

func appMain(args []string) int {
	if rtg0538Wrap(2) != 6 {
		print("RTG-0538 else return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
