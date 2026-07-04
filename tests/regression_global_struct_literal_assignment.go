package main

type rtgGlobalStructLiteralPair struct {
	left  int
	right int
}

var rtgGlobalStructLiteralValue rtgGlobalStructLiteralPair

func appMain(args []string) int {
	rtgGlobalStructLiteralValue = rtgGlobalStructLiteralPair{left: 4, right: 8}
	if rtgGlobalStructLiteralValue.left+rtgGlobalStructLiteralValue.right != 12 {
		print("global struct literal keyed assignment failed\n")
		return 1
	}
	rtgGlobalStructLiteralValue = rtgGlobalStructLiteralPair{9, 3}
	if rtgGlobalStructLiteralValue.left+rtgGlobalStructLiteralValue.right != 12 {
		print("global struct literal positional assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
