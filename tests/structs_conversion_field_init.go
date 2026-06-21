package main

type Rtg0624Box struct{ value int }

func appMain(args []string) int {
	box := Rtg0624Box{
		value: int(byte(33)),
	}
	if box.value != 33 {
		print("RTG-0624 conversion field init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
