package main

const rtg0680Flag = true

func appMain(args []string) int {
	if !rtg0680Flag {
		print("RTG-0680 bool const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
