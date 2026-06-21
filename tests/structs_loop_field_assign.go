package main

type Rtg0615Counter struct{ value int }

func appMain(args []string) int {
	c := Rtg0615Counter{}
	p := &c
	for i := 0; i < 4; i = i + 1 {
		p.value = p.value + i
	}
	if c.value != 6 {
		print("RTG-0615 loop field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
