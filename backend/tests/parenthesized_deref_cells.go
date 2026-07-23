package main

var parenthesizedSliceStorage = []int{35}
var parenthesizedSliceValue = &parenthesizedSliceStorage
var parenthesizedFuncStorage = func() int {
	return (*parenthesizedSliceValue)[0] + (*parenthesizedSliceValue)[1]
}
var parenthesizedFuncValue = &parenthesizedFuncStorage
var parenthesizedIntStorage = 40
var parenthesizedIntValue = &parenthesizedIntStorage

func appMain(args []string) int {
	(*parenthesizedSliceValue) = append((*parenthesizedSliceValue), 7)
	(*parenthesizedIntValue) += 2
	if len((*parenthesizedSliceValue)) == 2 && (*parenthesizedFuncValue)() == 42 && (*parenthesizedIntValue) == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
