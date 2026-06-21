package main

type Rtg0623Box struct{ value int }

func rtg0623Value() int {
	return 31
}

func appMain(args []string) int {
	box := Rtg0623Box{value: rtg0623Value()}
	if box.value != 31 {
		print("RTG-0623 helper field init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
