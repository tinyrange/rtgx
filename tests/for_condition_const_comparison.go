package main

const rtg0394Limit = 4

func appMain(args []string) int {
	i := 0
	for i < rtg0394Limit {
		i = i + 1
	}
	if i != 4 {
		print("RTG-0394 const condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
