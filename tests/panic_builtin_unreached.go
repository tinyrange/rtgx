package main

func panicIfRequested(requested bool) {
	if requested {
		panic("requested panic")
	}
}

func appMain(args []string) int {
	panicIfRequested(len(args) == 999)
	print("PASS\n")
	return 0
}
