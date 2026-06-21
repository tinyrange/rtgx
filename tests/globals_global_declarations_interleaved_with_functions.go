package main

var rtg0697Value int = 5

func rtg0697Get() int {
	return rtg0697Value
}

const rtg0697Add = 6

func appMain(args []string) int {
	if rtg0697Get()+rtg0697Add != 11 {
		print("RTG-0697 interleaved globals failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
