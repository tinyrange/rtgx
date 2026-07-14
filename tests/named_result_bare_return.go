package main

func namedResultFloatCall() float64 {
	return 3.5
}

func namedResultConversion() (result int) {
	result = int(namedResultFloatCall())
	return
}

func namedResultPair() (first int, second int) {
	first = 4
	second = 9
	return
}

func namedResultArray() (result [2]int) {
	result[0] = 6
	result[1] = 8
	return
}

func namedResultSlice() (result []int) {
	result = append(result, 7)
	return
}

func namedResultString() (result string) {
	result = "named"
	return
}

func appMain(args []string) int {
	if namedResultConversion() != 3 {
		print("FAIL\n")
		return 1
	}
	first, second := namedResultPair()
	if first != 4 || second != 9 {
		print("FAIL\n")
		return 1
	}
	array := namedResultArray()
	if array[0] != 6 || array[1] != 8 {
		print("FAIL\n")
		return 1
	}
	slice := namedResultSlice()
	if len(slice) != 1 || slice[0] != 7 || namedResultString() != "named" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
