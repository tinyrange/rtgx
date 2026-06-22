package main

func rtgStringBytesArgEmit(s string) {
	write(1, []byte(s), -1)
}

func appMain(args []string) int {
	rtgStringBytesArgEmit("PASS\n")
	return 0
}
