package main

var rtg0688Wide int64 = 123456

func appMain(args []string) int {
	if int(rtg0688Wide) != 123456 {
		print("RTG-0688 int64 global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
