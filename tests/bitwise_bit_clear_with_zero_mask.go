package main

func appMain(args []string) int {
	if !(0x44&^0 == 0x44) {
		print("RTG-0215 bit_clear_with_zero_mask failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
