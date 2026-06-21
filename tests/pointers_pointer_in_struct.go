package main

type Rtg0640Box struct{ p *int }

func appMain(args []string) int {
	value := 23
	box := Rtg0640Box{p: &value}
	q := &box
	*q.p = 24
	if value != 24 {
		print("RTG-0640 pointer in struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
