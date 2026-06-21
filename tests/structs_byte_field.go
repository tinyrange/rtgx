package main

type Rtg0604Byte struct{ b byte }

func rtg0604Make(b byte) Rtg0604Byte {
	return Rtg0604Byte{b: b}
}

func appMain(args []string) int {
	if rtg0604Make('m').b != 'm' {
		print("RTG-0604 byte field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
