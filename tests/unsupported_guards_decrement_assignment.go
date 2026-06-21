package main

func appMain(args []string) int {
	i := 9
	for i > 3 {
		i = i - 1
	}
	if i != 3 {
		print("RTG-0848 decrement assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
