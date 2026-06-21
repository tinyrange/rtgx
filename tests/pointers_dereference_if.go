package main

const rtg0646Want = 33

func appMain(args []string) int {
	value := rtg0646Want
	p := &value
	if *p == 33 {
		print("PASS\n")
		return 0
	}
	print("RTG-0646 dereference if failed\n")
	return 1
}
