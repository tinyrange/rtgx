package main

type Rtg0652Wide int64

func appMain(args []string) int {
	var n Rtg0652Wide = 9
	if n != 9 {
		print("RTG-0652 named int64 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
