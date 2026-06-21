package main

func appMain(args []string) int {
	if !(0x44|0 == 0x44) {
		print("RTG-0213 bitwise_or_with_zero failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
