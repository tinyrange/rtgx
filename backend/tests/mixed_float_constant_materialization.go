package main

const renvoMixedFloatConstant = (10 + 0.0)

func appMain() int {
	if renvoMixedFloatConstant != 10.0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
