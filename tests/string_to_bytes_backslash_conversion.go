package main

func appMain(args []string) int {
	bs := rtg0580Bytes("a\\b")
	if len(bs) != 3 || bs[1] != '\\' {
		print("RTG-0580 backslash conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0580Bytes(s string) []byte {
	return []byte(s)
}
