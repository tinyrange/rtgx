package main

func rtg0486Byte() byte { return byte(90) }
func appMain(args []string) int {
	if rtg0486Byte() != byte(90) {
		print("RTG-0486 byte return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
