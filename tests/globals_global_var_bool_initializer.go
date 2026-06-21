package main

var rtg0685Flag bool = true

func appMain(args []string) int {
	if !rtg0685Flag {
		print("RTG-0685 bool global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
