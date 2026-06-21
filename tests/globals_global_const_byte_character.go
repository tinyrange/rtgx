package main

const rtg0681Byte = 'A'

func appMain(args []string) int {
	if int(rtg0681Byte) != 65 {
		print("RTG-0681 byte const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
