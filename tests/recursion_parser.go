package main

func rtg0525Parse(bs []byte, i int) int {
	if i >= len(bs) {
		return 0
	}
	if bs[i] == '+' {
		return rtg0525Parse(bs, i+1)
	}
	return int(bs[i]-'0') + rtg0525Parse(bs, i+1)
}

func appMain(args []string) int {
	bs := []byte("2+3+4")
	if rtg0525Parse(bs, 0) != 9 {
		print("RTG-0525 parser failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
