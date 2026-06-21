package main

var rtg0597Want byte = 'k'

func rtg0597Find(bs []byte, i int) int {
	if i >= len(bs) {
		return -1
	}
	if bs[i] == rtg0597Want {
		return i
	}
	return rtg0597Find(bs, i+1)
}

func appMain(args []string) int {
	if rtg0597Find([]byte("stack"), 0) != 4 {
		print("RTG-0597 recursive scanner failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
