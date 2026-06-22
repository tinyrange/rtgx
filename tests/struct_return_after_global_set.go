package main

type rtgStructReturnAfterGlobalSetResult struct {
	data []byte
	ok   bool
}

var rtgStructReturnAfterGlobalSetState int

func rtgStructReturnAfterGlobalSetMark(v int) {
	if rtgStructReturnAfterGlobalSetState == 0 {
		rtgStructReturnAfterGlobalSetState = v
	}
}

func rtgStructReturnAfterGlobalSetFail(v int) rtgStructReturnAfterGlobalSetResult {
	rtgStructReturnAfterGlobalSetMark(v)
	var result rtgStructReturnAfterGlobalSetResult
	return result
}

func appMain(args []string) int {
	result := rtgStructReturnAfterGlobalSetFail(7)
	if rtgStructReturnAfterGlobalSetState != 7 || result.ok || len(result.data) != 0 {
		print("struct return after global set failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
