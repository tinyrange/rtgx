package main

var rtg0687Byte byte = 'z'

func appMain(args []string) int {
	if int(rtg0687Byte) != 122 {
		print("RTG-0687 byte global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
