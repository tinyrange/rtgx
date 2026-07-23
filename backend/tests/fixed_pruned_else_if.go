package main

var renvoFixedTarget = 1

func fixedPrunedTrue() bool {
	return true
}

func fixedPrunedElseIfContinues() bool {
	if renvoFixedTarget == 0 && fixedPrunedTrue() {
		if !fixedPrunedTrue() {
			return false
		}
	} else if !fixedPrunedTrue() {
		return false
	}
	return true
}

func appMain(args []string) int {
	if fixedPrunedElseIfContinues() {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
