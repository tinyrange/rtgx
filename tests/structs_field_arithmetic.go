package main

type Rtg0612Box struct{ value int }

func appMain(args []string) int {
	b := Rtg0612Box{value: 9}
	goto check
check:
	if b.value*2-3 != 15 {
		print("RTG-0612 field arithmetic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
