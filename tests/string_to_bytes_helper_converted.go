package main

func rtg0588Sum(bs []byte, i int) int {
	if i >= len(bs) {
		return 0
	}
	return int(bs[i]) + rtg0588Sum(bs, i+1)
}

func appMain(args []string) int {
	if rtg0588Sum([]byte("AB"), 0) != 131 {
		print("RTG-0588 helper converted failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
