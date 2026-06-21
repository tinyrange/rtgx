package main

type Rtg0601Empty struct{}

func appMain(args []string) int {
	_ = Rtg0601Empty{}
	print("PASS\n")
	return 0
}
