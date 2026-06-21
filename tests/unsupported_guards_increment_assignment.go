package main

var guardCounter int = 2

func appMain(args []string) int {
	i := guardCounter
	for i < 6 {
		i = i + 1
	}
	if i != 6 {
		print("RTG-0847 increment assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
