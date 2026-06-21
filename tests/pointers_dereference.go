package main

func appMain(args []string) int {
	value := 15
	p := &value
	got := *p
	if got != 15 {
		print("RTG-0627 dereference failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
