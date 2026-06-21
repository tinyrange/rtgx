package main

type Rtg0634Box struct{ value int }

func appMain(args []string) int {
	box := Rtg0634Box{value: 18}
	var got int
	for i := 0; i < 1; i = i + 1 {
		p := &box
		got = p.value
	}
	if got != 18 {
		print("RTG-0634 struct address failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
