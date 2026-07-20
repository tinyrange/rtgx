package main

func appMain(args []string) int {
	value := "\b"
	if len(value) != 1 || value[0] != 8 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
