package main

func appMain(args []string) int {
	state := 2
	value := 0
	if state == 1 {
		value = 10
	} else if state == 2 {
		value = 20
	} else {
		value = 30
	}
	if value != 20 {
		print("RTG-0827 no-switch dispatch failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
