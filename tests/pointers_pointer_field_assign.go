package main

type Rtg0636Box struct{ value int }

func appMain(args []string) int {
	box := Rtg0636Box{}
	p := &box
	for i := 0; i < 3; i = i + 1 {
		if i == 1 {
			continue
		}
		p.value = p.value + i
	}
	if box.value != 2 {
		print("RTG-0636 pointer field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
