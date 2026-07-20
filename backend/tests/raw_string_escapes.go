package main

var rawStringEscapesValue = `a\nb`

func appMain() int {
	if len(rawStringEscapesValue) != 4 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
