package main

func appMain(args []string) int {
	x := 2
	goto after
	if true {
		x = 9
	}
after:
	if x != 2 {
		print("RTG-0455 goto over if failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
