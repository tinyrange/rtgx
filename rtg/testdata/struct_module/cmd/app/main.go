package main

import "example.com/structfixture/pkg/state"

func main() {
	if state.Score() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
