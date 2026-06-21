package main

type counterBox struct{ value int }

func bumpBox(p *counterBox, amount int) { p.value += amount }

func appMain(args []string) int {
	box := counterBox{value: 5}
	bumpBox(&box, 7)
	if box.value != 12 {
		print("RTG-0850 function pointer receiver failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
