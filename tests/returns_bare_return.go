package main

var rtg0526Hit bool

func rtg0526Mark() {
	rtg0526Hit = true
	return
}

func appMain(args []string) int {
	rtg0526Mark()
	if !rtg0526Hit {
		print("RTG-0526 bare return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
