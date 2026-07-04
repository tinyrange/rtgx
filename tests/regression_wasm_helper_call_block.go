package main

func rtgWasmHelperBlockSame(a string, b string) bool {
	return a == b
}

func appMain(args []string) int {
	ok := rtgWasmHelperBlockSame("rtg", "rtg")
	if ok {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
