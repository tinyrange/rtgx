package main

func wasmFrameClobber(seed int) int {
	a := seed + 1
	b := seed + 2
	c := seed + 3
	return a + b + c
}

func wasmLargeCallerFrame() bool {
	var padding [1550]int
	values := make([]int, 1)
	values[0] = 31
	padding[12] = 11
	if wasmFrameClobber(10) != 36 {
		return false
	}
	return padding[12] == 11 && len(values) == 1 && cap(values) == 1 && values[0] == 31
}

func appMain() int {
	if !wasmLargeCallerFrame() {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
