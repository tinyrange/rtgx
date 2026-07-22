package main

func moduleExit() bool {
	data := []byte{'P'}
	return data[0] == 'P'
}

func appMain() int {
	if moduleExit() {
		print("PASS\n")
		return 0
	}
	return 1
}
